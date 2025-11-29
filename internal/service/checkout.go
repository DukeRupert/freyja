package service

import (
	"context"
	"errors"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

// Checkout-specific errors (ErrCartNotFound and ErrCartAlreadyConverted defined in errors.go)
var (
	ErrCartEmpty              = errors.New("cart is empty")
	ErrNoShippingRates        = errors.New("no shipping rates available for destination")
	ErrInvalidShippingAddress = errors.New("shipping address not serviceable")
	ErrInvalidTenantID        = errors.New("invalid tenant ID format")
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
	repo            repository.Querier
	cartService     CartService
	billingProvider billing.Provider
	shippingProvider shipping.Provider
	taxCalculator   tax.Calculator
	addrValidator   address.Validator
	tenantID        pgtype.UUID
}

// NewCheckoutService creates a new CheckoutService instance.
// TODO: Implementation notes:
// 1. Parse tenantID string to pgtype.UUID
// 2. Return error if parsing fails
// 3. Return checkoutService with all dependencies injected
func NewCheckoutService(
	repo repository.Querier,
	cartService CartService,
	billingProvider billing.Provider,
	shippingProvider shipping.Provider,
	taxCalculator tax.Calculator,
	addrValidator address.Validator,
	tenantID string,
) (CheckoutService, error) {
	panic("not implemented")
}

// ValidateAndNormalizeAddress validates a shipping or billing address.
// TODO: Implementation notes:
// 1. Call addrValidator.Validate(ctx, addr)
// 2. Return validation result (may contain errors but still have normalized address)
// 3. Let handler decide how to present validation errors to user
func (s *checkoutService) ValidateAndNormalizeAddress(ctx context.Context, addr address.Address) (*address.ValidationResult, error) {
	panic("not implemented")
}

// GetShippingRates calculates available shipping options for the cart.
// TODO: Implementation notes:
// 1. Load cart via CartService.GetCartSummary(cartID)
// 2. Validate cart is not empty
// 3. Calculate package weight from cart items (assume 340g per 12oz bag)
// 4. Determine package dimensions using standard box sizes:
//    - 1-3 bags: Small box (8x6x4 inches)
//    - 4-6 bags: Medium box (12x10x6 inches)
//    - 7+ bags: Large box (16x12x8 inches)
// 5. Get tenant warehouse address via repo.GetTenantWarehouseAddress
// 6. Convert addresses to shipping.ShippingAddress format
// 7. Call shippingProvider.GetRates with origin, destination, packages
// 8. Return rates sorted by cost (ascending)
func (s *checkoutService) GetShippingRates(ctx context.Context, cartID string, shippingAddr address.Address) ([]shipping.Rate, error) {
	panic("not implemented")
}

// CalculateOrderTotal computes the complete order total including tax and shipping.
// TODO: Implementation notes:
// 1. Load cart summary via CartService.GetCartSummary (provides subtotal)
// 2. Extract shipping cost from params.SelectedShippingRate
// 3. Build tax.LineItems from cart items
// 4. Call taxCalculator.CalculateTax with line items, shipping address, shipping cost
// 5. Calculate total: subtotal + shipping + tax - discount
// 6. Return OrderTotal with full breakdown including TaxBreakdown array
func (s *checkoutService) CalculateOrderTotal(ctx context.Context, params OrderTotalParams) (*OrderTotal, error) {
	panic("not implemented")
}

// CreatePaymentIntent initiates a Stripe Payment Intent.
// TODO: Implementation notes:
// 1. Validate params.OrderTotal is not nil
// 2. Serialize shipping and billing addresses to JSON for metadata
// 3. Build billing.CreatePaymentIntentParams with:
//    - AmountCents: OrderTotal.TotalCents
//    - Currency: "usd"
//    - CustomerEmail: params.CustomerEmail
//    - IdempotencyKey: params.IdempotencyKey
//    - Metadata: Include tenant_id, cart_id, addresses (JSON), shipping_rate_id, tax/shipping/subtotal cents
// 4. Call billingProvider.CreatePaymentIntent
// 5. Return payment intent with client_secret for frontend
// Note: Stripe handles idempotency automatically via IdempotencyKey
func (s *checkoutService) CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*billing.PaymentIntent, error) {
	panic("not implemented")
}

// CompleteCheckout is DEPRECATED - order creation happens via webhook.
// This method should panic with a deprecation message.
// Order creation flow: Stripe webhook → OrderService.CreateOrderFromPaymentIntent
// Frontend polls GET /api/orders/by-payment-intent/{id} to get order after payment.
// TODO: Implementation notes:
// panic("CompleteCheckout is deprecated - order creation happens via Stripe webhook")
func (s *checkoutService) CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*Order, error) {
	panic("not implemented")
}

// Helper functions (package-private)

// calculatePackage estimates package dimensions and weight from cart items.
// TODO: Implementation notes:
// 1. Sum total weight: count bags × 340g per 12oz bag
// 2. Determine box size based on bag count:
//    - 1-3 bags: Small box (20cm × 15cm × 10cm)
//    - 4-6 bags: Medium box (30cm × 25cm × 15cm)
//    - 7+ bags: Large box (40cm × 30cm × 20cm)
// 3. Return shipping.Package with weight and dimensions
func calculatePackage(items []CartItem) shipping.Package {
	panic("not implemented")
}

// convertAddressToShipping converts address.Address to shipping.ShippingAddress.
// TODO: Implementation notes:
// Simple field mapping between types
func convertAddressToShipping(addr address.Address) shipping.ShippingAddress {
	panic("not implemented")
}

// convertAddressToTax converts address.Address to tax.Address.
// TODO: Implementation notes:
// Simple field mapping between types
func convertAddressToTax(addr address.Address) tax.Address {
	panic("not implemented")
}

// Note: uuidToString already defined in order.go
