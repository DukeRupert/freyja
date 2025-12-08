package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

// Checkout-specific errors (ErrCartNotFound and ErrCartAlreadyConverted defined in errors.go)
var (
	ErrCartEmpty              = domain.Errorf(domain.EINVALID, "", "Cart is empty")
	ErrNoShippingRates        = domain.Errorf(domain.EINVALID, "", "No shipping rates available for destination")
	ErrInvalidShippingAddress = domain.Errorf(domain.EINVALID, "", "Shipping address not serviceable")
	ErrInvalidTenantID        = domain.Errorf(domain.EINVALID, "", "Invalid tenant ID format")
)

// CheckoutService provides business logic for checkout operations.
type CheckoutService interface {
	// ValidateAndNormalizeAddress validates a shipping or billing address.
	ValidateAndNormalizeAddress(ctx context.Context, addr address.Address) (*address.ValidationResult, error)

	// GetShippingRates calculates available shipping options for the cart.
	GetShippingRates(ctx context.Context, cartID string, shippingAddr address.Address) ([]shipping.Rate, error)

	// CalculateOrderTotal computes the complete order total including tax and shipping.
	CalculateOrderTotal(ctx context.Context, params OrderTotalParams) (*OrderTotal, error)

	// CreatePaymentIntent initiates a Stripe Payment Intent.
	CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*billing.PaymentIntent, error)

	// CompleteCheckout converts a cart to an order after successful payment (idempotent).
	CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*Order, error)
}

// OrderTotalParams contains parameters for calculating order totals.
type OrderTotalParams struct {
	CartID               string
	ShippingAddress      address.Address
	BillingAddress       address.Address
	SelectedShippingRate shipping.Rate
	DiscountCode         string
}

// OrderTotal contains the breakdown of an order's total cost.
type OrderTotal struct {
	SubtotalCents int32
	ShippingCents int32
	TaxCents      int32
	DiscountCents int32
	TotalCents    int32

	TaxBreakdown      []tax.TaxBreakdown
	TaxCalculationID  string // Stripe tax calculation ID for audit trail
	ShippingRateID    string
	DiscountCodeID    pgtype.UUID
	DiscountCodeValue string
}

// PaymentIntentParams contains parameters for creating a payment intent.
type PaymentIntentParams struct {
	CartID          string
	OrderTotal      *OrderTotal
	ShippingAddress address.Address
	BillingAddress  address.Address
	CustomerEmail   string
	IdempotencyKey  string
}

// CompleteCheckoutParams contains parameters for completing checkout.
type CompleteCheckoutParams struct {
	CartID          string
	PaymentIntentID string
	ShippingAddress address.Address
	BillingAddress  address.Address
	ShippingRateID  string
	DiscountCodeID  pgtype.UUID
	CustomerNotes   string
	IdempotencyKey  string
}

// Order represents a completed order.
type Order struct {
	ID            pgtype.UUID
	OrderNumber   string
	Status        string
	SubtotalCents int32
	TaxCents      int32
	ShippingCents int32
	DiscountCents int32
	TotalCents    int32
	Currency      string
	CreatedAt     pgtype.Timestamptz

	PaymentID         pgtype.UUID
	ShippingAddressID pgtype.UUID
	BillingAddressID  pgtype.UUID
}

// checkoutService implements CheckoutService.
type checkoutService struct {
	repo             repository.Querier
	cartService      domain.CartService
	billingProvider  billing.Provider
	shippingProvider shipping.Provider
	taxCalculator    tax.Calculator
	addrValidator    address.Validator
	tenantID         pgtype.UUID
}

// NewCheckoutService creates a new CheckoutService instance.
func NewCheckoutService(
	repo repository.Querier,
	cartService domain.CartService,
	billingProvider billing.Provider,
	shippingProvider shipping.Provider,
	taxCalculator tax.Calculator,
	addrValidator address.Validator,
	tenantID string,
) (CheckoutService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTenantID, err)
	}

	return &checkoutService{
		repo:             repo,
		cartService:      cartService,
		billingProvider:  billingProvider,
		shippingProvider: shippingProvider,
		taxCalculator:    taxCalculator,
		addrValidator:    addrValidator,
		tenantID:         tenantUUID,
	}, nil
}

// ValidateAndNormalizeAddress validates a shipping or billing address.
func (s *checkoutService) ValidateAndNormalizeAddress(ctx context.Context, addr address.Address) (*address.ValidationResult, error) {
	return s.addrValidator.Validate(ctx, addr)
}

// GetShippingRates calculates available shipping options for the cart.
func (s *checkoutService) GetShippingRates(ctx context.Context, cartID string, shippingAddr address.Address) ([]shipping.Rate, error) {
	cartSummary, err := s.cartService.GetCartSummary(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart: %w", err)
	}

	if len(cartSummary.Items) == 0 {
		return nil, ErrCartEmpty
	}

	pkg := calculatePackage(cartSummary.Items)

	warehouseAddr, err := s.repo.GetTenantWarehouseAddress(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse address: %w", err)
	}

	origin := shipping.ShippingAddress{
		Name:       "",
		Line1:      warehouseAddr.AddressLine1,
		Line2:      warehouseAddr.AddressLine2.String,
		City:       warehouseAddr.City,
		State:      warehouseAddr.State,
		PostalCode: warehouseAddr.PostalCode,
		Country:    warehouseAddr.Country,
		Phone:      "",
	}

	destination := convertAddressToShipping(shippingAddr)

	// Convert tenant ID to string for shipping provider
	tenantIDStr := ""
	if s.tenantID.Valid {
		b := s.tenantID.Bytes
		tenantIDStr = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	}

	rates, err := s.shippingProvider.GetRates(ctx, shipping.RateParams{
		TenantID:           tenantIDStr,
		OriginAddress:      origin,
		DestinationAddress: destination,
		Packages:           []shipping.Package{pkg},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping rates: %w", err)
	}

	if len(rates) == 0 {
		return nil, ErrNoShippingRates
	}

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].CostCents < rates[j].CostCents
	})

	return rates, nil
}

// CalculateOrderTotal computes the complete order total including tax and shipping.
func (s *checkoutService) CalculateOrderTotal(ctx context.Context, params OrderTotalParams) (*OrderTotal, error) {
	cartSummary, err := s.cartService.GetCartSummary(ctx, params.CartID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart: %w", err)
	}

	// Convert int64 to int32 - safe for shipping costs which are always < $21M
	shippingCents := int32(params.SelectedShippingRate.CostCents)

	lineItems := make([]tax.LineItem, len(cartSummary.Items))
	for i, item := range cartSummary.Items {
		lineItems[i] = tax.LineItem{
			ProductID:   item.SKUID,
			Description: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPriceCents,
			TotalPrice:  item.LineSubtotal,
			TaxCategory: "food",
		}
	}

	taxResult, err := s.taxCalculator.CalculateTax(ctx, tax.TaxParams{
		ShippingAddress: convertAddressToTax(params.ShippingAddress),
		LineItems:       lineItems,
		ShippingCents:   shippingCents,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tax: %w", err)
	}

	total := cartSummary.Subtotal + shippingCents + taxResult.TotalTaxCents

	return &OrderTotal{
		SubtotalCents:     cartSummary.Subtotal,
		ShippingCents:     shippingCents,
		TaxCents:          taxResult.TotalTaxCents,
		DiscountCents:     0,
		TotalCents:        total,
		TaxBreakdown:      taxResult.Breakdown,
		TaxCalculationID:  taxResult.ProviderTxID,
		ShippingRateID:    params.SelectedShippingRate.RateID,
		DiscountCodeID:    pgtype.UUID{},
		DiscountCodeValue: "",
	}, nil
}

// CreatePaymentIntent initiates a Stripe Payment Intent.
func (s *checkoutService) CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*billing.PaymentIntent, error) {
	if params.OrderTotal == nil {
		return nil, errors.New("order total is required")
	}

	shippingAddrJSON, err := json.Marshal(params.ShippingAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize shipping address: %w", err)
	}

	billingAddrJSON, err := json.Marshal(params.BillingAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize billing address: %w", err)
	}

	metadata := map[string]string{
		"tenant_id":          uuidToString(s.tenantID),
		"cart_id":            params.CartID,
		"customer_email":     params.CustomerEmail,
		"shipping_address":   string(shippingAddrJSON),
		"billing_address":    string(billingAddrJSON),
		"shipping_rate_id":   params.OrderTotal.ShippingRateID,
		"subtotal_cents":     strconv.FormatInt(int64(params.OrderTotal.SubtotalCents), 10),
		"shipping_cents":     strconv.FormatInt(int64(params.OrderTotal.ShippingCents), 10),
		"tax_cents":          strconv.FormatInt(int64(params.OrderTotal.TaxCents), 10),
		"tax_calculation_id": params.OrderTotal.TaxCalculationID,
	}

	paymentIntent, err := s.billingProvider.CreatePaymentIntent(ctx, billing.CreatePaymentIntentParams{
		AmountCents:    params.OrderTotal.TotalCents,
		Currency:       "usd",
		CustomerEmail:  params.CustomerEmail,
		IdempotencyKey: params.IdempotencyKey,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return paymentIntent, nil
}

// CompleteCheckout is DEPRECATED - order creation happens via webhook.
// This method should panic with a deprecation message.
// Order creation flow: Stripe webhook → OrderService.CreateOrderFromPaymentIntent
// Frontend polls GET /api/orders/by-payment-intent/{id} to get order after payment.
func (s *checkoutService) CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*Order, error) {
	panic("CompleteCheckout is deprecated - order creation happens via Stripe webhook")
}

// Helper functions (package-private)

// calculatePackage estimates package dimensions and weight from cart items.
func calculatePackage(items []domain.CartItem) shipping.Package {
	var totalBags int32
	for _, item := range items {
		totalBags += item.Quantity
	}

	weightGrams := totalBags * 340

	var length, width, height int
	if totalBags <= 3 {
		length = 20
		width = 15
		height = 10
	} else if totalBags <= 6 {
		length = 30
		width = 25
		height = 15
	} else {
		length = 40
		width = 30
		height = 20
	}

	return shipping.Package{
		WeightGrams: weightGrams,
		LengthCm:    int32(length),
		WidthCm:     int32(width),
		HeightCm:    int32(height),
	}
}

// convertAddressToShipping converts address.Address to shipping.ShippingAddress.
func convertAddressToShipping(addr address.Address) shipping.ShippingAddress {
	return shipping.ShippingAddress{
		Name:       addr.FullName,
		Line1:      addr.AddressLine1,
		Line2:      addr.AddressLine2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.PostalCode,
		Country:    addr.Country,
		Phone:      addr.Phone,
	}
}

// convertAddressToTax converts address.Address to tax.Address.
func convertAddressToTax(addr address.Address) tax.Address {
	return tax.Address{
		Line1:      addr.AddressLine1,
		Line2:      addr.AddressLine2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.PostalCode,
		Country:    addr.Country,
	}
}

// Note: uuidToString already defined in order.go
