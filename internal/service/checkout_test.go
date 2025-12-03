package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// mockCartService implements CartService for testing
type mockCartService struct {
	summary *CartSummary
	err     error
}

func (m *mockCartService) GetOrCreateCart(ctx context.Context, sessionID string) (*Cart, string, error) {
	return nil, "", errors.New("not implemented in mock")
}

func (m *mockCartService) GetCart(ctx context.Context, sessionID string) (*Cart, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCartService) AddItem(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCartService) UpdateItemQuantity(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCartService) RemoveItem(ctx context.Context, cartID string, skuID string) (*CartSummary, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockCartService) GetCartSummary(ctx context.Context, cartID string) (*CartSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.summary, nil
}

// mockBillingProvider implements billing.Provider for testing
type mockBillingProvider struct {
	paymentIntent       *billing.PaymentIntent
	paymentIntentErr    error
	getPaymentIntentErr error
	createCalled        bool
	lastCreateParams    *billing.CreatePaymentIntentParams
}

func (m *mockBillingProvider) CreatePaymentIntent(ctx context.Context, params billing.CreatePaymentIntentParams) (*billing.PaymentIntent, error) {
	m.createCalled = true
	m.lastCreateParams = &params
	if m.paymentIntentErr != nil {
		return nil, m.paymentIntentErr
	}
	return m.paymentIntent, nil
}

func (m *mockBillingProvider) GetPaymentIntent(ctx context.Context, params billing.GetPaymentIntentParams) (*billing.PaymentIntent, error) {
	if m.getPaymentIntentErr != nil {
		return nil, m.getPaymentIntentErr
	}
	return m.paymentIntent, nil
}

func (m *mockBillingProvider) UpdatePaymentIntent(ctx context.Context, params billing.UpdatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockBillingProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	return errors.New("not implemented in mock")
}

func (m *mockBillingProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	return errors.New("not implemented in mock")
}

func (m *mockBillingProvider) CreateCustomer(ctx context.Context, params billing.CreateCustomerParams) (*billing.Customer, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockBillingProvider) GetCustomer(ctx context.Context, customerID string) (*billing.Customer, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockBillingProvider) UpdateCustomer(ctx context.Context, customerID string, params billing.UpdateCustomerParams) (*billing.Customer, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockBillingProvider) CreateSubscription(ctx context.Context, params billing.SubscriptionParams) (*billing.Subscription, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockBillingProvider) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error {
	return errors.New("not implemented in mock")
}

func (m *mockBillingProvider) RefundPayment(ctx context.Context, params billing.RefundParams) (*billing.Refund, error) {
	return nil, errors.New("not implemented in mock")
}

// mockShippingProviderCheckout implements shipping.Provider for testing
type mockShippingProviderCheckout struct {
	rates    []shipping.Rate
	ratesErr error
}

func (m *mockShippingProviderCheckout) GetRates(ctx context.Context, params shipping.RateParams) ([]shipping.Rate, error) {
	if m.ratesErr != nil {
		return nil, m.ratesErr
	}
	return m.rates, nil
}

func (m *mockShippingProviderCheckout) CreateLabel(ctx context.Context, params shipping.LabelParams) (*shipping.Label, error) {
	return nil, shipping.ErrNotImplemented
}

func (m *mockShippingProviderCheckout) VoidLabel(ctx context.Context, labelID string) error {
	return shipping.ErrNotImplemented
}

func (m *mockShippingProviderCheckout) TrackShipment(ctx context.Context, trackingNumber string) (*shipping.TrackingInfo, error) {
	return nil, shipping.ErrNotImplemented
}

// mockTaxCalculator implements tax.Calculator for testing
type mockTaxCalculator struct {
	result *tax.TaxResult
	err    error
}

func (m *mockTaxCalculator) CalculateTax(ctx context.Context, params tax.TaxParams) (*tax.TaxResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// mockAddressValidator implements address.Validator for testing
type mockAddressValidator struct {
	result *address.ValidationResult
	err    error
}

func (m *mockAddressValidator) Validate(ctx context.Context, addr address.Address) (*address.ValidationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// mockQuerierCheckout extends mockQuerier with checkout-specific overrides
type mockQuerierCheckout struct {
	mockQuerier
	warehouseAddress    *repository.GetTenantWarehouseAddressRow
	warehouseAddressErr error
}

func (m *mockQuerierCheckout) GetTenantWarehouseAddress(ctx context.Context, tenantID pgtype.UUID) (repository.GetTenantWarehouseAddressRow, error) {
	if m.warehouseAddressErr != nil {
		return repository.GetTenantWarehouseAddressRow{}, m.warehouseAddressErr
	}
	if m.warehouseAddress != nil {
		return *m.warehouseAddress, nil
	}
	return repository.GetTenantWarehouseAddressRow{}, errors.New("warehouse address not configured in mock")
}

// ============================================================================
// Test Fixtures
// ============================================================================

func makeTestAddress2() address.Address {
	return address.Address{
		Type:         "shipping",
		FullName:     "John Doe",
		Company:      "",
		AddressLine1: "123 Main St",
		AddressLine2: "Apt 4B",
		City:         "Portland",
		State:        "OR",
		PostalCode:   "97201",
		Country:      "US",
		Phone:        "503-555-1234",
	}
}

func makeTestWarehouseAddress() repository.GetTenantWarehouseAddressRow {
	return repository.GetTenantWarehouseAddressRow{
		AddressLine1: "456 Warehouse Rd",
		AddressLine2: pgtype.Text{Valid: false},
		City:         "Portland",
		State:        "OR",
		PostalCode:   "97202",
		Country:      "US",
	}
}

func makeTestCartSummary(itemCount int) *CartSummary {
	items := make([]CartItem, itemCount)
	var subtotal int32
	for i := 0; i < itemCount; i++ {
		item := CartItem{
			ID:             mustParseUUID("33333333-3333-3333-3333-333333333333"),
			SKUID:          mustParseUUID("44444444-4444-4444-4444-444444444444"),
			ProductName:    "Ethiopian Yirgacheffe",
			SKU:            "ETH-YRG-12OZ-WB",
			WeightValue:    "12oz",
			Grind:          "whole_bean",
			Quantity:       1,
			UnitPriceCents: 1800,
			LineSubtotal:   1800,
		}
		items[i] = item
		subtotal += item.LineSubtotal
	}

	return &CartSummary{
		Cart: Cart{
			ID:       makeTestCartID(),
			TenantID: makeTestTenantID(),
		},
		Items:     items,
		Subtotal:  subtotal,
		ItemCount: itemCount,
	}
}

func makeTestShippingRate(costCents int64, carrier string, serviceName string) shipping.Rate {
	return shipping.Rate{
		RateID:                "rate_" + carrier + "_001",
		Carrier:               carrier,
		ServiceName:           serviceName,
		ServiceCode:           "std",
		CostCents:             costCents,
		EstimatedDaysMin:      3,
		EstimatedDaysMax:      5,
		EstimatedDeliveryDate: time.Now().Add(4 * 24 * time.Hour),
	}
}

func makeTestTaxResult(taxCents int32) *tax.TaxResult {
	return &tax.TaxResult{
		TotalTaxCents: taxCents,
		Breakdown: []tax.TaxBreakdown{
			{
				Jurisdiction: "state",
				Name:         "Oregon",
				Rate:         0.00,
				AmountCents:  0,
			},
		},
		ProviderTxID: "tax_test_123",
		IsEstimate:   false,
	}
}

// ============================================================================
// Test NewCheckoutService
// ============================================================================

func TestNewCheckoutService_Success(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}
	tenantID := uuidToString(makeTestTenantID())

	svc, err := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, tenantID)

	// Currently panics with "not implemented"
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if svc == nil {
	//     t.Error("Expected non-nil service")
	// }
	_ = svc
}

func TestNewCheckoutService_InvalidTenantID(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	_, err := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, "invalid-uuid")

	// Currently panics, but once implemented should return error
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for invalid tenant ID, got nil")
	// }
	// if !errors.Is(err, ErrInvalidTenantID) {
	//     t.Errorf("Expected ErrInvalidTenantID, got: %v", err)
	// }
}

// ============================================================================
// Test ValidateAndNormalizeAddress
// ============================================================================

func TestCheckoutService_ValidateAndNormalizeAddress_Success(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}

	addr := makeTestAddress2()
	normalizedAddr := addr
	normalizedAddr.PostalCode = "97201-1234" // Normalized with +4

	mockValidator := &mockAddressValidator{
		result: &address.ValidationResult{
			IsValid:           true,
			NormalizedAddress: &normalizedAddr,
			Errors:            []address.ValidationError{},
			Warnings:          []string{},
		},
	}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	result, err := svc.ValidateAndNormalizeAddress(ctx, addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if !result.IsValid {
	//     t.Error("Expected valid address")
	// }
	// if result.NormalizedAddress.PostalCode != "97201-1234" {
	//     t.Errorf("Expected normalized postal code, got: %s", result.NormalizedAddress.PostalCode)
	// }
	_ = result
}

func TestCheckoutService_ValidateAndNormalizeAddress_InvalidAddress(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}

	addr := makeTestAddress2()
	addr.City = "InvalidCity"

	mockValidator := &mockAddressValidator{
		result: &address.ValidationResult{
			IsValid:           false,
			NormalizedAddress: nil,
			Errors: []address.ValidationError{
				{Field: "city", Message: "City not found"},
			},
			Warnings: []string{},
		},
	}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	result, err := svc.ValidateAndNormalizeAddress(ctx, addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Unexpected error: %v", err)
	// }
	// if result.IsValid {
	//     t.Error("Expected invalid address")
	// }
	// if len(result.Errors) == 0 {
	//     t.Error("Expected validation errors")
	// }
	_ = result
}

func TestCheckoutService_ValidateAndNormalizeAddress_ValidatorError(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}

	validatorErr := errors.New("validation service unavailable")
	mockValidator := &mockAddressValidator{
		err: validatorErr,
	}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.ValidateAndNormalizeAddress(ctx, addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error from validator, got nil")
	// }
	// if !errors.Is(err, validatorErr) {
	//     t.Errorf("Expected validator error, got: %v", err)
	// }
}

// ============================================================================
// Test GetShippingRates
// ============================================================================

func TestCheckoutService_GetShippingRates_Success_SmallBox(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	// 2 bags = small box
	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	rates := []shipping.Rate{
		makeTestShippingRate(500, "USPS", "Standard"),
		makeTestShippingRate(1200, "FedEx", "Express"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	result, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if len(result) != 2 {
	//     t.Errorf("Expected 2 rates, got: %d", len(result))
	// }
	// // Verify rates are sorted by cost
	// if result[0].CostCents > result[1].CostCents {
	//     t.Error("Expected rates sorted by cost ascending")
	// }
	_ = result
}

func TestCheckoutService_GetShippingRates_Success_MediumBox(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	// 5 bags = medium box
	cartSummary := makeTestCartSummary(5)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	rates := []shipping.Rate{
		makeTestShippingRate(800, "USPS", "Standard"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented, verify package dimensions are calculated correctly for medium box
}

func TestCheckoutService_GetShippingRates_Success_LargeBox(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	// 8 bags = large box
	cartSummary := makeTestCartSummary(8)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	rates := []shipping.Rate{
		makeTestShippingRate(1200, "USPS", "Standard"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented, verify package dimensions are calculated correctly for large box
}

func TestCheckoutService_GetShippingRates_EmptyCart(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}

	// Empty cart
	cartSummary := &CartSummary{
		Cart: Cart{
			ID:       makeTestCartID(),
			TenantID: makeTestTenantID(),
		},
		Items:     []CartItem{},
		Subtotal:  0,
		ItemCount: 0,
	}
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrCartEmpty) {
	//     t.Errorf("Expected ErrCartEmpty, got: %v", err)
	// }
}

func TestCheckoutService_GetShippingRates_CartNotFound(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{
		err: ErrCartNotFound,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrCartNotFound) {
	//     t.Errorf("Expected ErrCartNotFound, got: %v", err)
	// }
}

func TestCheckoutService_GetShippingRates_WarehouseAddressNotFound(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddressErr: errors.New("warehouse address not configured"),
	}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for missing warehouse address, got nil")
	// }
}

func TestCheckoutService_GetShippingRates_NoRatesAvailable(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	// No rates available for destination
	mockShipping := &mockShippingProviderCheckout{
		rates: []shipping.Rate{},
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrNoShippingRates) {
	//     t.Errorf("Expected ErrNoShippingRates, got: %v", err)
	// }
}

func TestCheckoutService_GetShippingRates_ShippingProviderError(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	shippingErr := errors.New("shipping API unavailable")
	mockShipping := &mockShippingProviderCheckout{
		ratesErr: shippingErr,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected shipping provider error, got nil")
	// }
}

func TestCheckoutService_GetShippingRates_RatesSortedByCost(t *testing.T) {
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	// Rates returned in random order
	rates := []shipping.Rate{
		makeTestShippingRate(1500, "FedEx", "Overnight"),
		makeTestShippingRate(500, "USPS", "Standard"),
		makeTestShippingRate(1000, "UPS", "2-Day"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	result, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// // Verify sorted by cost ascending
	// if result[0].CostCents != 500 {
	//     t.Errorf("Expected cheapest rate first, got: %d", result[0].CostCents)
	// }
	// if result[1].CostCents != 1000 {
	//     t.Errorf("Expected second cheapest rate, got: %d", result[1].CostCents)
	// }
	// if result[2].CostCents != 1500 {
	//     t.Errorf("Expected most expensive rate last, got: %d", result[2].CostCents)
	// }
	_ = result
}

// ============================================================================
// Test CalculateOrderTotal
// ============================================================================

func TestCheckoutService_CalculateOrderTotal_Success(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}

	cartSummary := makeTestCartSummary(2)
	cartSummary.Subtotal = 3600 // 2 bags @ $18.00 each
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}

	taxResult := makeTestTaxResult(288) // 8% tax on $36
	mockTax := &mockTaxCalculator{
		result: taxResult,
	}

	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	shippingRate := makeTestShippingRate(500, "USPS", "Standard")
	params := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      makeTestAddress2(),
		BillingAddress:       makeTestAddress2(),
		SelectedShippingRate: shippingRate,
		DiscountCode:         "",
	}

	result, err := svc.CalculateOrderTotal(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if result.SubtotalCents != 3600 {
	//     t.Errorf("Expected subtotal 3600, got: %d", result.SubtotalCents)
	// }
	// if result.ShippingCents != 500 {
	//     t.Errorf("Expected shipping 500, got: %d", result.ShippingCents)
	// }
	// if result.TaxCents != 288 {
	//     t.Errorf("Expected tax 288, got: %d", result.TaxCents)
	// }
	// if result.TotalCents != 4388 {
	//     t.Errorf("Expected total 4388 (3600 + 500 + 288), got: %d", result.TotalCents)
	// }
	_ = result
}

func TestCheckoutService_CalculateOrderTotal_CartNotFound(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{
		err: ErrCartNotFound,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      makeTestAddress2(),
		BillingAddress:       makeTestAddress2(),
		SelectedShippingRate: makeTestShippingRate(500, "USPS", "Standard"),
	}

	_, err := svc.CalculateOrderTotal(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrCartNotFound) {
	//     t.Errorf("Expected ErrCartNotFound, got: %v", err)
	// }
}

func TestCheckoutService_CalculateOrderTotal_TaxCalculationError(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}

	taxErr := errors.New("tax service unavailable")
	mockTax := &mockTaxCalculator{
		err: taxErr,
	}

	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      makeTestAddress2(),
		BillingAddress:       makeTestAddress2(),
		SelectedShippingRate: makeTestShippingRate(500, "USPS", "Standard"),
	}

	_, err := svc.CalculateOrderTotal(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected tax calculation error, got nil")
	// }
}

func TestCheckoutService_CalculateOrderTotal_WithDiscount(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}

	cartSummary := makeTestCartSummary(2)
	cartSummary.Subtotal = 3600
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}

	taxResult := makeTestTaxResult(288)
	mockTax := &mockTaxCalculator{
		result: taxResult,
	}

	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      makeTestAddress2(),
		BillingAddress:       makeTestAddress2(),
		SelectedShippingRate: makeTestShippingRate(500, "USPS", "Standard"),
		DiscountCode:         "SAVE10",
	}

	result, err := svc.CalculateOrderTotal(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented (discount code handling is future work):
	// This test documents expected behavior when discount codes are implemented
	// For now, discount handling may not be implemented
	_ = result
}

// ============================================================================
// Test CreatePaymentIntent
// ============================================================================

func TestCheckoutService_CreatePaymentIntent_Success(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}

	mockBilling := &mockBillingProvider{
		paymentIntent: &billing.PaymentIntent{
			ID:           "pi_test_123456",
			ClientSecret: "pi_test_123456_secret_xyz",
			AmountCents:  4388,
			Currency:     "usd",
			Status:       "requires_payment_method",
			Metadata: map[string]string{
				"tenant_id": uuidToString(makeTestTenantID()),
				"cart_id":   uuidToString(makeTestCartID()),
			},
		},
	}

	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	orderTotal := &OrderTotal{
		SubtotalCents:  3600,
		ShippingCents:  500,
		TaxCents:       288,
		DiscountCents:  0,
		TotalCents:     4388,
		TaxBreakdown:   []tax.TaxBreakdown{},
		ShippingRateID: "rate_usps_001",
	}

	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      orderTotal,
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		CustomerEmail:   "customer@example.com",
		IdempotencyKey:  "idempotency_test_123",
	}

	result, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if result.ID != "pi_test_123456" {
	//     t.Errorf("Expected payment intent ID, got: %s", result.ID)
	// }
	// if result.ClientSecret == "" {
	//     t.Error("Expected client secret")
	// }
	// if result.AmountCents != 4388 {
	//     t.Errorf("Expected amount 4388, got: %d", result.AmountCents)
	// }
	//
	// // Verify metadata includes required fields
	// if mockBilling.createCalled {
	//     meta := mockBilling.lastCreateParams.Metadata
	//     if meta["tenant_id"] != uuidToString(makeTestTenantID()) {
	//         t.Error("Expected tenant_id in metadata")
	//     }
	//     if meta["cart_id"] != uuidToString(makeTestCartID()) {
	//         t.Error("Expected cart_id in metadata")
	//     }
	//     if meta["shipping_address"] == "" {
	//         t.Error("Expected shipping_address JSON in metadata")
	//     }
	//     if meta["billing_address"] == "" {
	//         t.Error("Expected billing_address JSON in metadata")
	//     }
	// }
	_ = result
}

func TestCheckoutService_CreatePaymentIntent_NilOrderTotal(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      nil, // Invalid: nil order total
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		CustomerEmail:   "customer@example.com",
		IdempotencyKey:  "idempotency_test_123",
	}

	_, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for nil order total, got nil")
	// }
}

func TestCheckoutService_CreatePaymentIntent_BillingProviderError(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}

	billingErr := errors.New("stripe API error")
	mockBilling := &mockBillingProvider{
		paymentIntentErr: billingErr,
	}

	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	orderTotal := &OrderTotal{
		SubtotalCents: 3600,
		ShippingCents: 500,
		TaxCents:      288,
		TotalCents:    4388,
	}

	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      orderTotal,
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		CustomerEmail:   "customer@example.com",
		IdempotencyKey:  "idempotency_test_123",
	}

	_, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected billing provider error, got nil")
	// }
}

func TestCheckoutService_CreatePaymentIntent_MetadataFormat(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}

	mockBilling := &mockBillingProvider{
		paymentIntent: &billing.PaymentIntent{
			ID:           "pi_test_metadata",
			ClientSecret: "pi_test_secret",
			AmountCents:  4388,
			Currency:     "usd",
			Status:       "requires_payment_method",
		},
	}

	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	orderTotal := &OrderTotal{
		SubtotalCents:  3600,
		ShippingCents:  500,
		TaxCents:       288,
		TotalCents:     4388,
		ShippingRateID: "rate_usps_std",
	}

	shippingAddr := makeTestAddress2()
	billingAddr := makeTestAddress2()

	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      orderTotal,
		ShippingAddress: shippingAddr,
		BillingAddress:  billingAddr,
		CustomerEmail:   "customer@example.com",
		IdempotencyKey:  "idempotency_test_meta",
	}

	_, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	//
	// // Verify metadata structure
	// if !mockBilling.createCalled {
	//     t.Fatal("Expected CreatePaymentIntent to be called")
	// }
	//
	// meta := mockBilling.lastCreateParams.Metadata
	//
	// // Verify JSON serialization of addresses
	// var shippingJSON map[string]string
	// if err := json.Unmarshal([]byte(meta["shipping_address"]), &shippingJSON); err != nil {
	//     t.Errorf("Expected valid JSON in shipping_address metadata: %v", err)
	// }
	// if shippingJSON["full_name"] != shippingAddr.FullName {
	//     t.Error("Shipping address full_name mismatch in metadata")
	// }
	//
	// // Verify billing address JSON
	// var billingJSON map[string]string
	// if err := json.Unmarshal([]byte(meta["billing_address"]), &billingJSON); err != nil {
	//     t.Errorf("Expected valid JSON in billing_address metadata: %v", err)
	// }
	//
	// // Verify numeric metadata
	// if meta["subtotal_cents"] != "3600" {
	//     t.Errorf("Expected subtotal_cents in metadata, got: %s", meta["subtotal_cents"])
	// }
	// if meta["shipping_cents"] != "500" {
	//     t.Errorf("Expected shipping_cents in metadata, got: %s", meta["shipping_cents"])
	// }
	// if meta["tax_cents"] != "288" {
	//     t.Errorf("Expected tax_cents in metadata, got: %s", meta["tax_cents"])
	// }
	// if meta["shipping_rate_id"] != "rate_usps_std" {
	//     t.Errorf("Expected shipping_rate_id in metadata, got: %s", meta["shipping_rate_id"])
	// }
}

func TestCheckoutService_CreatePaymentIntent_IdempotencyKey(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}

	mockBilling := &mockBillingProvider{
		paymentIntent: &billing.PaymentIntent{
			ID:           "pi_test_idempotent",
			ClientSecret: "pi_test_secret",
			AmountCents:  4388,
			Currency:     "usd",
			Status:       "requires_payment_method",
		},
	}

	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	orderTotal := &OrderTotal{
		TotalCents: 4388,
	}

	idempotencyKey := "unique_checkout_session_789"
	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      orderTotal,
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		CustomerEmail:   "customer@example.com",
		IdempotencyKey:  idempotencyKey,
	}

	_, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	//
	// // Verify idempotency key is passed through
	// if !mockBilling.createCalled {
	//     t.Fatal("Expected CreatePaymentIntent to be called")
	// }
	// if mockBilling.lastCreateParams.IdempotencyKey != idempotencyKey {
	//     t.Errorf("Expected idempotency key %s, got: %s", idempotencyKey, mockBilling.lastCreateParams.IdempotencyKey)
	// }
}

// ============================================================================
// Test CompleteCheckout (DEPRECATED)
// ============================================================================

func TestCheckoutService_CompleteCheckout_Panics(t *testing.T) {
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}
	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := CompleteCheckoutParams{
		CartID:          uuidToString(makeTestCartID()),
		PaymentIntentID: "pi_test_123",
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		ShippingRateID:  "rate_usps_001",
		IdempotencyKey:  "idem_test",
	}

	// Should panic with deprecation message
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected CompleteCheckout to panic with deprecation message")
		} else {
			msg := r.(string)
			if msg != "CompleteCheckout is deprecated - order creation happens via Stripe webhook" {
				t.Errorf("Expected deprecation panic message, got: %s", msg)
			}
		}
	}()

	_, _ = svc.CompleteCheckout(ctx, params)
}

// ============================================================================
// Test Helper Functions
// ============================================================================

func TestCalculatePackage_SmallBox(t *testing.T) {
	// 2 bags should result in small box
	items := []CartItem{
		{Quantity: 1, ProductName: "Coffee A"},
		{Quantity: 1, ProductName: "Coffee B"},
	}

	// Currently panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic (not implemented)")
		}
	}()

	pkg := calculatePackage(items)

	// Once implemented:
	// Expected weight: 2 bags × 340g = 680g
	// Expected dimensions: 20cm × 15cm × 10cm (small box)
	_ = pkg
}

func TestCalculatePackage_MediumBox(t *testing.T) {
	// 5 bags should result in medium box
	items := []CartItem{
		{Quantity: 5, ProductName: "Coffee A"},
	}

	// Currently panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic (not implemented)")
		}
	}()

	pkg := calculatePackage(items)

	// Once implemented:
	// Expected weight: 5 bags × 340g = 1700g
	// Expected dimensions: 30cm × 25cm × 15cm (medium box)
	_ = pkg
}

func TestCalculatePackage_LargeBox(t *testing.T) {
	// 10 bags should result in large box
	items := []CartItem{
		{Quantity: 10, ProductName: "Coffee A"},
	}

	// Currently panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic (not implemented)")
		}
	}()

	pkg := calculatePackage(items)

	// Once implemented:
	// Expected weight: 10 bags × 340g = 3400g
	// Expected dimensions: 40cm × 30cm × 20cm (large box)
	_ = pkg
}

func TestConvertAddressToShipping(t *testing.T) {
	addr := makeTestAddress2()

	// Currently panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic (not implemented)")
		}
	}()

	shippingAddr := convertAddressToShipping(addr)

	// Once implemented:
	// Verify field mapping
	// if shippingAddr.Name != addr.FullName {
	//     t.Error("Name field mapping incorrect")
	// }
	// if shippingAddr.Line1 != addr.AddressLine1 {
	//     t.Error("Line1 field mapping incorrect")
	// }
	// if shippingAddr.City != addr.City {
	//     t.Error("City field mapping incorrect")
	// }
	_ = shippingAddr
}

func TestConvertAddressToTax(t *testing.T) {
	addr := makeTestAddress2()

	// Currently panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic (not implemented)")
		}
	}()

	taxAddr := convertAddressToTax(addr)

	// Once implemented:
	// Verify field mapping
	// if taxAddr.Line1 != addr.AddressLine1 {
	//     t.Error("Line1 field mapping incorrect")
	// }
	// if taxAddr.City != addr.City {
	//     t.Error("City field mapping incorrect")
	// }
	_ = taxAddr
}

// ============================================================================
// Integration-Style Tests
// ============================================================================

func TestCheckoutService_CompleteFlow_Success(t *testing.T) {
	// This test simulates the complete checkout flow:
	// 1. Validate address
	// 2. Get shipping rates
	// 3. Calculate order total
	// 4. Create payment intent

	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	cartSummary := makeTestCartSummary(3)
	cartSummary.Subtotal = 5400 // 3 bags @ $18 each
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{
		paymentIntent: &billing.PaymentIntent{
			ID:           "pi_test_flow",
			ClientSecret: "pi_test_flow_secret",
			AmountCents:  6332,
			Currency:     "usd",
			Status:       "requires_payment_method",
		},
	}

	rates := []shipping.Rate{
		makeTestShippingRate(500, "USPS", "Standard"),
		makeTestShippingRate(1200, "FedEx", "Express"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	taxResult := makeTestTaxResult(432) // Tax on $54
	mockTax := &mockTaxCalculator{
		result: taxResult,
	}

	addr := makeTestAddress2()
	normalizedAddr := addr
	normalizedAddr.PostalCode = "97201-1234"
	mockValidator := &mockAddressValidator{
		result: &address.ValidationResult{
			IsValid:           true,
			NormalizedAddress: &normalizedAddr,
			Errors:            []address.ValidationError{},
			Warnings:          []string{},
		},
	}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))
	ctx := context.Background()

	// Step 1: Validate address
	_, err := svc.ValidateAndNormalizeAddress(ctx, addr)
	if err == nil {
		t.Error("Expected error (not implemented)")
	}

	// Step 2: Get shipping rates
	_, err = svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), normalizedAddr)
	if err == nil {
		t.Error("Expected error (not implemented)")
	}

	// Step 3: Calculate order total
	totalParams := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      normalizedAddr,
		BillingAddress:       normalizedAddr,
		SelectedShippingRate: rates[0], // Choose standard shipping
	}
	_, err = svc.CalculateOrderTotal(ctx, totalParams)
	if err == nil {
		t.Error("Expected error (not implemented)")
	}

	// Step 4: Create payment intent
	// paymentParams := PaymentIntentParams{
	//     CartID:          uuidToString(makeTestCartID()),
	//     OrderTotal:      orderTotal,
	//     ShippingAddress: normalizedAddr,
	//     BillingAddress:  normalizedAddr,
	//     CustomerEmail:   "customer@example.com",
	//     IdempotencyKey:  "flow_test_123",
	// }
	// _, err = svc.CreatePaymentIntent(ctx, paymentParams)
	// if err == nil {
	//     t.Error("Expected error (not implemented)")
	// }

	// Once implemented, verify the complete flow works end-to-end
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestCheckoutService_GetShippingRates_InternationalAddress(t *testing.T) {
	// Test shipping to international address (Canada)
	mockRepo := &mockQuerierCheckout{
		warehouseAddress: &repository.GetTenantWarehouseAddressRow{
			AddressLine1: "456 Warehouse Rd",
			City:         "Portland",
			State:        "OR",
			PostalCode:   "97202",
			Country:      "US",
		},
	}

	cartSummary := makeTestCartSummary(2)
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	rates := []shipping.Rate{
		makeTestShippingRate(2500, "USPS", "International Standard"),
	}
	mockShipping := &mockShippingProviderCheckout{
		rates: rates,
	}

	mockBilling := &mockBillingProvider{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	addr := makeTestAddress2()
	addr.Country = "CA"
	addr.State = "BC"
	addr.PostalCode = "V6B 1A1"

	_, err := svc.GetShippingRates(ctx, uuidToString(makeTestCartID()), addr)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented)")
	}
	// Once implemented, verify international shipping works
}

func TestCheckoutService_CalculateOrderTotal_ZeroTax(t *testing.T) {
	// Test order in state with no sales tax (Oregon for coffee)
	mockRepo := &mockQuerierCheckout{}

	cartSummary := makeTestCartSummary(2)
	cartSummary.Subtotal = 3600
	mockCart := &mockCartService{
		summary: cartSummary,
	}

	mockBilling := &mockBillingProvider{}
	mockShipping := &mockShippingProviderCheckout{}

	// Oregon has no sales tax on coffee
	taxResult := makeTestTaxResult(0)
	mockTax := &mockTaxCalculator{
		result: taxResult,
	}

	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	params := OrderTotalParams{
		CartID:               uuidToString(makeTestCartID()),
		ShippingAddress:      makeTestAddress2(),
		BillingAddress:       makeTestAddress2(),
		SelectedShippingRate: makeTestShippingRate(500, "USPS", "Standard"),
	}

	result, err := svc.CalculateOrderTotal(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented)")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// if result.TaxCents != 0 {
	//     t.Errorf("Expected zero tax, got: %d", result.TaxCents)
	// }
	// if result.TotalCents != 4100 { // 3600 + 500 + 0
	//     t.Errorf("Expected total 4100, got: %d", result.TotalCents)
	// }
	_ = result
}

func TestCheckoutService_CreatePaymentIntent_LargeOrder(t *testing.T) {
	// Test payment intent for large order amount
	mockRepo := &mockQuerierCheckout{}
	mockCart := &mockCartService{}

	mockBilling := &mockBillingProvider{
		paymentIntent: &billing.PaymentIntent{
			ID:           "pi_test_large",
			ClientSecret: "pi_test_large_secret",
			AmountCents:  50000, // $500.00
			Currency:     "usd",
			Status:       "requires_payment_method",
		},
	}

	mockShipping := &mockShippingProviderCheckout{}
	mockTax := &mockTaxCalculator{}
	mockValidator := &mockAddressValidator{}

	svc, _ := NewCheckoutService(mockRepo, mockCart, mockBilling, mockShipping, mockTax, mockValidator, uuidToString(makeTestTenantID()))

	ctx := context.Background()
	orderTotal := &OrderTotal{
		SubtotalCents: 45000,
		ShippingCents: 2000,
		TaxCents:      3000,
		TotalCents:    50000,
	}

	params := PaymentIntentParams{
		CartID:          uuidToString(makeTestCartID()),
		OrderTotal:      orderTotal,
		ShippingAddress: makeTestAddress2(),
		BillingAddress:  makeTestAddress2(),
		CustomerEmail:   "wholesale@example.com",
		IdempotencyKey:  "large_order_test",
	}

	result, err := svc.CreatePaymentIntent(ctx, params)

	// Currently panics
	if err == nil {
		t.Error("Expected error (not implemented)")
	}
	// Once implemented, verify large amounts are handled correctly
	_ = result
}

// ============================================================================
// Test JSON Serialization Helpers
// ============================================================================

func TestAddressJSONSerialization(t *testing.T) {
	addr := makeTestAddress2()

	// Test that address can be serialized to JSON for metadata
	jsonData, err := json.Marshal(addr)
	if err != nil {
		t.Fatalf("Failed to marshal address: %v", err)
	}

	// Verify it can be deserialized
	var decoded address.Address
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal address: %v", err)
	}

	if decoded.FullName != addr.FullName {
		t.Error("Address full name not preserved in JSON round-trip")
	}
	if decoded.AddressLine1 != addr.AddressLine1 {
		t.Error("Address line 1 not preserved in JSON round-trip")
	}
}
