package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockQuerier implements repository.Querier for testing
type mockQuerier struct {
	// CreateOrderFromPaymentIntent mocks
	GetOrderByPaymentIntentIDFunc func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error)
	GetCartByIDFunc               func(ctx context.Context, id pgtype.UUID) (repository.Cart, error)
	GetCartItemsFunc              func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error)
	CreateAddressFunc             func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error)
	CreateBillingCustomerFunc     func(ctx context.Context, arg repository.CreateBillingCustomerParams) (repository.BillingCustomer, error)
	CreatePaymentFunc             func(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error)
	CreateOrderFunc               func(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error)
	CreateOrderItemFunc           func(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error)
	DecrementSKUStockFunc         func(ctx context.Context, arg repository.DecrementSKUStockParams) error
	UpdateCartStatusFunc          func(ctx context.Context, arg repository.UpdateCartStatusParams) error
	UpdateOrderPaymentIDFunc      func(ctx context.Context, arg repository.UpdateOrderPaymentIDParams) error

	// GetOrder mocks
	GetOrderFunc      func(ctx context.Context, arg repository.GetOrderParams) (repository.Order, error)
	GetOrderItemsFunc func(ctx context.Context, orderID pgtype.UUID) ([]repository.OrderItem, error)
	GetAddressByIDFunc func(ctx context.Context, id pgtype.UUID) (repository.Address, error)
	GetPaymentByIDFunc func(ctx context.Context, id pgtype.UUID) (repository.Payment, error)

	// GetOrderByNumber mocks
	GetOrderByNumberFunc func(ctx context.Context, arg repository.GetOrderByNumberParams) (repository.Order, error)
}

func (m *mockQuerier) GetOrderByPaymentIntentID(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
	if m.GetOrderByPaymentIntentIDFunc != nil {
		return m.GetOrderByPaymentIntentIDFunc(ctx, arg)
	}
	return repository.Order{}, sql.ErrNoRows
}

func (m *mockQuerier) GetCartByID(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
	if m.GetCartByIDFunc != nil {
		return m.GetCartByIDFunc(ctx, id)
	}
	return repository.Cart{}, sql.ErrNoRows
}

func (m *mockQuerier) GetCartItems(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
	if m.GetCartItemsFunc != nil {
		return m.GetCartItemsFunc(ctx, cartID)
	}
	return nil, nil
}

func (m *mockQuerier) CreateAddress(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
	if m.CreateAddressFunc != nil {
		return m.CreateAddressFunc(ctx, arg)
	}
	return repository.Address{}, errors.New("not implemented")
}

func (m *mockQuerier) CreateBillingCustomer(ctx context.Context, arg repository.CreateBillingCustomerParams) (repository.BillingCustomer, error) {
	if m.CreateBillingCustomerFunc != nil {
		return m.CreateBillingCustomerFunc(ctx, arg)
	}
	return repository.BillingCustomer{}, errors.New("not implemented")
}

func (m *mockQuerier) CreatePayment(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error) {
	if m.CreatePaymentFunc != nil {
		return m.CreatePaymentFunc(ctx, arg)
	}
	return repository.Payment{}, errors.New("not implemented")
}

func (m *mockQuerier) CreateOrder(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
	if m.CreateOrderFunc != nil {
		return m.CreateOrderFunc(ctx, arg)
	}
	return repository.Order{}, errors.New("not implemented")
}

func (m *mockQuerier) CreateOrderItem(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error) {
	if m.CreateOrderItemFunc != nil {
		return m.CreateOrderItemFunc(ctx, arg)
	}
	return repository.OrderItem{}, errors.New("not implemented")
}

func (m *mockQuerier) DecrementSKUStock(ctx context.Context, arg repository.DecrementSKUStockParams) error {
	if m.DecrementSKUStockFunc != nil {
		return m.DecrementSKUStockFunc(ctx, arg)
	}
	return nil
}

func (m *mockQuerier) UpdateCartStatus(ctx context.Context, arg repository.UpdateCartStatusParams) error {
	if m.UpdateCartStatusFunc != nil {
		return m.UpdateCartStatusFunc(ctx, arg)
	}
	return nil
}

func (m *mockQuerier) UpdateOrderPaymentID(ctx context.Context, arg repository.UpdateOrderPaymentIDParams) error {
	if m.UpdateOrderPaymentIDFunc != nil {
		return m.UpdateOrderPaymentIDFunc(ctx, arg)
	}
	return nil
}

func (m *mockQuerier) GetOrder(ctx context.Context, arg repository.GetOrderParams) (repository.Order, error) {
	if m.GetOrderFunc != nil {
		return m.GetOrderFunc(ctx, arg)
	}
	return repository.Order{}, sql.ErrNoRows
}

func (m *mockQuerier) GetOrderItems(ctx context.Context, orderID pgtype.UUID) ([]repository.OrderItem, error) {
	if m.GetOrderItemsFunc != nil {
		return m.GetOrderItemsFunc(ctx, orderID)
	}
	return nil, nil
}

func (m *mockQuerier) GetAddressByID(ctx context.Context, id pgtype.UUID) (repository.Address, error) {
	if m.GetAddressByIDFunc != nil {
		return m.GetAddressByIDFunc(ctx, id)
	}
	return repository.Address{}, sql.ErrNoRows
}

func (m *mockQuerier) GetPaymentByID(ctx context.Context, id pgtype.UUID) (repository.Payment, error) {
	if m.GetPaymentByIDFunc != nil {
		return m.GetPaymentByIDFunc(ctx, id)
	}
	return repository.Payment{}, sql.ErrNoRows
}

func (m *mockQuerier) GetOrderByNumber(ctx context.Context, arg repository.GetOrderByNumberParams) (repository.Order, error) {
	if m.GetOrderByNumberFunc != nil {
		return m.GetOrderByNumberFunc(ctx, arg)
	}
	return repository.Order{}, sql.ErrNoRows
}

// Stub methods to satisfy repository.Querier interface
func (m *mockQuerier) AddCartItem(ctx context.Context, arg repository.AddCartItemParams) (repository.CartItem, error) {
	return repository.CartItem{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) ClearCart(ctx context.Context, cartID pgtype.UUID) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateCart(ctx context.Context, arg repository.CreateCartParams) (repository.Cart, error) {
	return repository.Cart{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateSession(ctx context.Context, arg repository.CreateSessionParams) (repository.Session, error) {
	return repository.Session{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
	return repository.User{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) DeleteExpiredSessions(ctx context.Context) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) DeleteSession(ctx context.Context, token string) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) GetBaseProductForWhiteLabel(ctx context.Context, id pgtype.UUID) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetCartBySessionID(ctx context.Context, sessionID pgtype.UUID) (repository.Cart, error) {
	return repository.Cart{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetCartItemCount(ctx context.Context, cartID pgtype.UUID) (int32, error) {
	return 0, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetDefaultPriceList(ctx context.Context, tenantID pgtype.UUID) (repository.PriceList, error) {
	return repository.PriceList{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetPriceForSKU(ctx context.Context, arg repository.GetPriceForSKUParams) (repository.PriceListEntry, error) {
	return repository.PriceListEntry{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetPriceListByID(ctx context.Context, id pgtype.UUID) (repository.PriceList, error) {
	return repository.PriceList{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetPricesForProduct(ctx context.Context, arg repository.GetPricesForProductParams) ([]repository.GetPricesForProductRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetPricesForSKUs(ctx context.Context, arg repository.GetPricesForSKUsParams) ([]repository.GetPricesForSKUsRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetPrimaryImage(ctx context.Context, productID pgtype.UUID) (repository.ProductImage, error) {
	return repository.ProductImage{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductByID(ctx context.Context, arg repository.GetProductByIDParams) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductBySlug(ctx context.Context, arg repository.GetProductBySlugParams) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetShipmentsByOrderID(ctx context.Context, orderID pgtype.UUID) ([]repository.Shipment, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductImages(ctx context.Context, productID pgtype.UUID) ([]repository.ProductImage, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductSKU(ctx context.Context, id pgtype.UUID) (repository.ProductSku, error) {
	return repository.ProductSku{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetSKUByID(ctx context.Context, id pgtype.UUID) (repository.ProductSku, error) {
	return repository.ProductSku{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductSKUs(ctx context.Context, productID pgtype.UUID) ([]repository.ProductSku, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetProductsForCustomer(ctx context.Context, arg repository.GetProductsForCustomerParams) ([]repository.Product, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetSession(ctx context.Context, token string) (repository.Session, error) {
	return repository.Session{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetSessionByToken(ctx context.Context, token string) (repository.Session, error) {
	return repository.Session{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetUser(ctx context.Context, id pgtype.UUID) (repository.User, error) {
	return repository.User{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetUserByID(ctx context.Context, id pgtype.UUID) (repository.User, error) {
	return repository.User{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, arg repository.GetUserByEmailParams) (repository.User, error) {
	return repository.User{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetUserStats(ctx context.Context, tenantID pgtype.UUID) (repository.GetUserStatsRow, error) {
	return repository.GetUserStatsRow{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListActiveProducts(ctx context.Context, tenantID pgtype.UUID) ([]repository.ListActiveProductsRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListAllPriceLists(ctx context.Context, tenantID pgtype.UUID) ([]repository.PriceList, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListAllProducts(ctx context.Context, tenantID pgtype.UUID) ([]repository.ListAllProductsRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListOrders(ctx context.Context, arg repository.ListOrdersParams) ([]repository.ListOrdersRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListOrdersByStatus(ctx context.Context, arg repository.ListOrdersByStatusParams) ([]repository.ListOrdersByStatusRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListUsers(ctx context.Context, arg repository.ListUsersParams) ([]repository.ListUsersRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListUsersByAccountType(ctx context.Context, arg repository.ListUsersByAccountTypeParams) ([]repository.ListUsersByAccountTypeRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) ListWholesaleApplications(ctx context.Context, tenantID pgtype.UUID) ([]repository.ListWholesaleApplicationsRow, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) RemoveCartItem(ctx context.Context, arg repository.RemoveCartItemParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateCartItemQuantity(ctx context.Context, arg repository.UpdateCartItemQuantityParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) GetWhiteLabelProductsForCustomer(ctx context.Context, arg repository.GetWhiteLabelProductsForCustomerParams) ([]repository.Product, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateSessionLastActivity(ctx context.Context, token string) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateSessionData(ctx context.Context, arg repository.UpdateSessionDataParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateUserPassword(ctx context.Context, arg repository.UpdateUserPasswordParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateUserProfile(ctx context.Context, arg repository.UpdateUserProfileParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateUserStatus(ctx context.Context, arg repository.UpdateUserStatusParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateWholesaleApplication(ctx context.Context, arg repository.UpdateWholesaleApplicationParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) VerifyUserEmail(ctx context.Context, id pgtype.UUID) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) SetPrimaryImage(ctx context.Context, arg repository.SetPrimaryImageParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateOrderFulfillmentStatus(ctx context.Context, arg repository.UpdateOrderFulfillmentStatusParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateOrderStatus(ctx context.Context, arg repository.UpdateOrderStatusParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdatePriceListEntry(ctx context.Context, arg repository.UpdatePriceListEntryParams) (repository.PriceListEntry, error) {
	return repository.PriceListEntry{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateProduct(ctx context.Context, arg repository.UpdateProductParams) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateProductImage(ctx context.Context, arg repository.UpdateProductImageParams) (repository.ProductImage, error) {
	return repository.ProductImage{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateProductSKU(ctx context.Context, arg repository.UpdateProductSKUParams) (repository.ProductSku, error) {
	return repository.ProductSku{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) UpdateShipmentStatus(ctx context.Context, arg repository.UpdateShipmentStatusParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) CreatePriceListEntry(ctx context.Context, arg repository.CreatePriceListEntryParams) (repository.PriceListEntry, error) {
	return repository.PriceListEntry{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateProduct(ctx context.Context, arg repository.CreateProductParams) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateProductSKU(ctx context.Context, arg repository.CreateProductSKUParams) (repository.ProductSku, error) {
	return repository.ProductSku{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateProductImage(ctx context.Context, arg repository.CreateProductImageParams) (repository.ProductImage, error) {
	return repository.ProductImage{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) DeleteProduct(ctx context.Context, arg repository.DeleteProductParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) DeleteProductSKU(ctx context.Context, arg repository.DeleteProductSKUParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) DeleteProductImage(ctx context.Context, arg repository.DeleteProductImageParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockQuerier) GetOrderStats(ctx context.Context, arg repository.GetOrderStatsParams) (repository.GetOrderStatsRow, error) {
	return repository.GetOrderStatsRow{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetOrderWithDetails(ctx context.Context, arg repository.GetOrderWithDetailsParams) (repository.GetOrderWithDetailsRow, error) {
	return repository.GetOrderWithDetailsRow{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CreateShipment(ctx context.Context, arg repository.CreateShipmentParams) (repository.Shipment, error) {
	return repository.Shipment{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) GetTenantWarehouseAddress(ctx context.Context, tenantID pgtype.UUID) (repository.GetTenantWarehouseAddressRow, error) {
	return repository.GetTenantWarehouseAddressRow{}, errors.New("not implemented in mock")
}

func (m *mockQuerier) CountOrders(ctx context.Context, tenantID pgtype.UUID) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

func (m *mockQuerier) CountUsers(ctx context.Context, tenantID pgtype.UUID) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

// Test fixtures and helpers

func mustParseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		panic(err)
	}
	return u
}

func makeTestTenantID() pgtype.UUID {
	return mustParseUUID("01234567-89ab-cdef-0123-456789abcdef")
}

func makeTestCartID() pgtype.UUID {
	return mustParseUUID("11111111-1111-1111-1111-111111111111")
}

func makeTestOrderID() pgtype.UUID {
	return mustParseUUID("22222222-2222-2222-2222-222222222222")
}

func makeTestSKUID() pgtype.UUID {
	return mustParseUUID("33333333-3333-3333-3333-333333333333")
}

func makeTestAddressID() pgtype.UUID {
	return mustParseUUID("44444444-4444-4444-4444-444444444444")
}

func makeTestPaymentID() pgtype.UUID {
	return mustParseUUID("55555555-5555-5555-5555-555555555555")
}

func makeTestBillingCustomerID() pgtype.UUID {
	return mustParseUUID("66666666-6666-6666-6666-666666666666")
}

func makeTestUserID() pgtype.UUID {
	return mustParseUUID("77777777-7777-7777-7777-777777777777")
}

func makeTestAddress(addressType string) repository.Address {
	return repository.Address{
		ID:           makeTestAddressID(),
		TenantID:     makeTestTenantID(),
		FullName:     pgtype.Text{String: "John Doe", Valid: true},
		AddressLine1: "123 Main St",
		City:         "Portland",
		State:        "OR",
		PostalCode:   "97201",
		Country:      "US",
		Phone:        pgtype.Text{String: "503-555-1234", Valid: true},
		AddressType:  addressType,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func makeTestPaymentIntent(status string, metadata map[string]string) *billing.PaymentIntent {
	return &billing.PaymentIntent{
		ID:            "pi_test_123456",
		ClientSecret:  "pi_test_123456_secret_test",
		AmountCents:   2500, // $25.00
		Currency:      "usd",
		Status:        status,
		TaxCents:      200,
		ShippingCents: 500,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}
}

func makeTestCart() repository.Cart {
	return repository.Cart{
		ID:       makeTestCartID(),
		TenantID: makeTestTenantID(),
		UserID:   makeTestUserID(),
		Status:   "active",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func makeTestGuestCart() repository.Cart {
	cart := makeTestCart()
	cart.UserID = pgtype.UUID{Valid: false} // NULL for guest cart
	return cart
}

func makeTestCartItems() []repository.GetCartItemsRow {
	return []repository.GetCartItemsRow{
		{
			ID:                makeTestSKUID(),
			CartID:            makeTestCartID(),
			ProductSkuID:      makeTestSKUID(),
			Quantity:          2,
			UnitPriceCents:    1000,
			ProductName:       "Ethiopian Yirgacheffe",
			Sku:               "ETH-YRG-12OZ-WB",
			WeightValue:       pgtype.Numeric{Int: big.NewInt(12), Valid: true},
			WeightUnit:        "oz",
			Grind:             "whole_bean",
			InventoryQuantity: 100,
		},
	}
}

func makeTestOrder() repository.Order {
	return repository.Order{
		ID:                makeTestOrderID(),
		TenantID:          makeTestTenantID(),
		UserID:            makeTestUserID(),
		OrderNumber:       "ORD-20250129-TEST",
		OrderType:         "retail",
		Status:            "pending",
		SubtotalCents:     2000,
		TaxCents:          200,
		ShippingCents:     500,
		TotalCents:        2700,
		Currency:          "usd",
		PaymentID:         makeTestPaymentID(),
		ShippingAddressID: makeTestAddressID(),
		BillingAddressID:  makeTestAddressID(),
		CreatedAt:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

// Test CreateOrderFromPaymentIntent

func TestOrderService_CreateOrderFromPaymentIntent_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_123456"

	// Setup mocks
	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Mock payment intent with metadata
	metadata := map[string]string{
		"tenant_id":        uuidToString(tenantID),
		"cart_id":          uuidToString(cartID),
		"shipping_address": mustMarshalJSON(map[string]string{
			"full_name":      "John Doe",
			"address_line1":  "123 Main St",
			"city":           "Portland",
			"state":          "OR",
			"postal_code":    "97201",
			"country":        "US",
			"phone":          "503-555-1234",
		}),
		"billing_address": mustMarshalJSON(map[string]string{
			"full_name":     "John Doe",
			"address_line1": "123 Main St",
			"city":          "Portland",
			"state":         "OR",
			"postal_code":   "97201",
			"country":       "US",
			"phone":         "503-555-1234",
		}),
	}

	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	// Mock repository calls
	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows // No existing order
	}

	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return makeTestCart(), nil
	}

	mockRepo.GetCartItemsFunc = func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
		return makeTestCartItems(), nil
	}

	// Create service
	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Note: This test will fail until implementation is complete
	// It serves as a specification for the expected behavior
	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented, this should succeed:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_Idempotency(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	paymentIntentID := "pi_test_123456"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Mock that order already exists for this payment intent
	existingOrder := makeTestOrder()
	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		if arg.TenantID.Bytes != tenantID.Bytes {
			t.Error("Tenant ID mismatch in idempotency check")
		}
		if arg.ProviderPaymentID != paymentIntentID {
			t.Error("Payment intent ID mismatch in idempotency check")
		}
		return existingOrder, nil
	}

	// These should NOT be called if idempotency works
	mockBilling.GetPaymentIntentFunc = func(ctx context.Context, params billing.GetPaymentIntentParams) (*billing.PaymentIntent, error) {
		t.Error("GetPaymentIntent should not be called when order already exists")
		return nil, errors.New("should not be called")
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// This should return the existing order without creating a new one
	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
	// Verify returned order matches existing order
}

func TestOrderService_CreateOrderFromPaymentIntent_PaymentNotSucceeded(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	paymentIntentID := "pi_test_pending"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Payment intent is pending, not succeeded
	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		"cart_id":   uuidToString(makeTestCartID()),
	}
	paymentIntent := makeTestPaymentIntent("pending", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrPaymentNotSucceeded) {
	//     t.Errorf("Expected ErrPaymentNotSucceeded, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_MissingCartID(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	paymentIntentID := "pi_test_no_cart"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Payment intent missing cart_id in metadata
	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		// Missing cart_id
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrMissingCartID) {
	//     t.Errorf("Expected ErrMissingCartID, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_CartNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_cart_not_found"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		"cart_id":   uuidToString(cartID),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return repository.Cart{}, sql.ErrNoRows
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrCartNotFound) {
	//     t.Errorf("Expected ErrCartNotFound, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_TenantMismatch(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	wrongTenantID := mustParseUUID("99999999-9999-9999-9999-999999999999")
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_tenant_mismatch"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		"cart_id":   uuidToString(cartID),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	// Cart belongs to different tenant
	cart := makeTestCart()
	cart.TenantID = wrongTenantID
	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return cart, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrTenantMismatch) {
	//     t.Errorf("Expected ErrTenantMismatch, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_CartAlreadyConverted(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_converted_cart"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		"cart_id":   uuidToString(cartID),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	// Cart already converted
	cart := makeTestCart()
	cart.Status = "converted"
	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return cart, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrCartAlreadyConverted) {
	//     t.Errorf("Expected ErrCartAlreadyConverted, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_EmptyCart(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_empty_cart"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id": uuidToString(tenantID),
		"cart_id":   uuidToString(cartID),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return makeTestCart(), nil
	}

	// Empty cart items
	mockRepo.GetCartItemsFunc = func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
		return []repository.GetCartItemsRow{}, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for empty cart, got nil")
	// }
	// if !strings.Contains(err.Error(), "empty cart") {
	//     t.Errorf("Expected 'empty cart' error, got: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_GuestCheckout(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_guest"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id":        uuidToString(tenantID),
		"cart_id":          uuidToString(cartID),
		"shipping_address": mustMarshalJSON(map[string]string{
			"full_name":     "Guest User",
			"address_line1": "456 Oak Ave",
			"city":          "Eugene",
			"state":         "OR",
			"postal_code":   "97401",
			"country":       "US",
		}),
		"billing_address": mustMarshalJSON(map[string]string{
			"full_name":     "Guest User",
			"address_line1": "456 Oak Ave",
			"city":          "Eugene",
			"state":         "OR",
			"postal_code":   "97401",
			"country":       "US",
		}),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	// Guest cart (NULL user_id)
	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return makeTestGuestCart(), nil
	}

	mockRepo.GetCartItemsFunc = func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
		return makeTestCartItems(), nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Should handle guest checkout (NULL user_id) successfully
	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Guest checkout should succeed, got error: %v", err)
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_InvalidAddressJSON(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_bad_address"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id":        uuidToString(tenantID),
		"cart_id":          uuidToString(cartID),
		"shipping_address": "invalid json{",
		"billing_address":  "also invalid",
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return makeTestCart(), nil
	}

	mockRepo.GetCartItemsFunc = func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
		return makeTestCartItems(), nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for invalid address JSON, got nil")
	// }
}

func TestOrderService_CreateOrderFromPaymentIntent_MultipleItems(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	cartID := makeTestCartID()
	paymentIntentID := "pi_test_multiple_items"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	metadata := map[string]string{
		"tenant_id":        uuidToString(tenantID),
		"cart_id":          uuidToString(cartID),
		"shipping_address": mustMarshalJSON(map[string]string{"full_name": "Test", "address_line1": "123 St", "city": "Portland", "state": "OR", "postal_code": "97201", "country": "US"}),
		"billing_address":  mustMarshalJSON(map[string]string{"full_name": "Test", "address_line1": "123 St", "city": "Portland", "state": "OR", "postal_code": "97201", "country": "US"}),
	}
	paymentIntent := makeTestPaymentIntent("succeeded", metadata)
	mockBilling.PaymentIntents[paymentIntentID] = paymentIntent

	mockRepo.GetOrderByPaymentIntentIDFunc = func(ctx context.Context, arg repository.GetOrderByPaymentIntentIDParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	mockRepo.GetCartByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Cart, error) {
		return makeTestCart(), nil
	}

	// Multiple cart items
	mockRepo.GetCartItemsFunc = func(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartItemsRow, error) {
		return []repository.GetCartItemsRow{
			{
				ID:                mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				ProductSkuID:      mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				Quantity:          2,
				UnitPriceCents:    1000,
				ProductName:       "Ethiopian Yirgacheffe",
				Sku:               "ETH-YRG-12OZ-WB",
				Grind:             "whole_bean",
				InventoryQuantity: 100,
			},
			{
				ID:                mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				ProductSkuID:      mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				Quantity:          1,
				UnitPriceCents:    1200,
				ProductName:       "Colombian Supremo",
				Sku:               "COL-SUP-16OZ-GR",
				Grind:             "medium",
				InventoryQuantity: 50,
			},
		}, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Should handle multiple items correctly
	_, err = svc.CreateOrderFromPaymentIntent(ctx, paymentIntentID)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented, verify all items are created
}

// Test GetOrder

func TestOrderService_GetOrder_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderID := makeTestOrderID()

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	order := makeTestOrder()
	mockRepo.GetOrderFunc = func(ctx context.Context, arg repository.GetOrderParams) (repository.Order, error) {
		if arg.TenantID.Bytes != tenantID.Bytes {
			t.Error("Tenant ID mismatch")
		}
		if arg.ID.Bytes != orderID.Bytes {
			t.Error("Order ID mismatch")
		}
		return order, nil
	}

	mockRepo.GetOrderItemsFunc = func(ctx context.Context, orderID pgtype.UUID) ([]repository.OrderItem, error) {
		return []repository.OrderItem{}, nil
	}

	mockRepo.GetAddressByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Address, error) {
		return makeTestAddress("shipping"), nil
	}

	mockRepo.GetPaymentByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Payment, error) {
		return repository.Payment{
			ID:                makeTestPaymentID(),
			TenantID:          tenantID,
			ProviderPaymentID: "pi_test_123456",
			AmountCents:       2700,
			Currency:          "usd",
			Status:            "succeeded",
		}, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrder(ctx, uuidToString(orderID))
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
}

func TestOrderService_GetOrder_NotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderID := makeTestOrderID()

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	mockRepo.GetOrderFunc = func(ctx context.Context, arg repository.GetOrderParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrder(ctx, uuidToString(orderID))
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrOrderNotFound) {
	//     t.Errorf("Expected ErrOrderNotFound, got: %v", err)
	// }
}

func TestOrderService_GetOrder_TenantMismatch(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderID := makeTestOrderID()

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Simulate accessing order from different tenant
	mockRepo.GetOrderFunc = func(ctx context.Context, arg repository.GetOrderParams) (repository.Order, error) {
		if arg.TenantID.Bytes != tenantID.Bytes {
			return repository.Order{}, sql.ErrNoRows // Proper tenant scoping
		}
		return makeTestOrder(), nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test verifies that GetOrder properly scopes by tenant_id
	_, err = svc.GetOrder(ctx, uuidToString(orderID))
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
}

func TestOrderService_GetOrder_InvalidUUID(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrder(ctx, "invalid-uuid-format")
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err == nil {
	//     t.Error("Expected error for invalid UUID, got nil")
	// }
}

// Test GetOrderByNumber

func TestOrderService_GetOrderByNumber_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderNumber := "ORD-20250129-TEST"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	order := makeTestOrder()
	mockRepo.GetOrderByNumberFunc = func(ctx context.Context, arg repository.GetOrderByNumberParams) (repository.Order, error) {
		if arg.TenantID.Bytes != tenantID.Bytes {
			t.Error("Tenant ID mismatch")
		}
		if arg.OrderNumber != orderNumber {
			t.Error("Order number mismatch")
		}
		return order, nil
	}

	mockRepo.GetOrderItemsFunc = func(ctx context.Context, orderID pgtype.UUID) ([]repository.OrderItem, error) {
		return []repository.OrderItem{}, nil
	}

	mockRepo.GetAddressByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Address, error) {
		return makeTestAddress("shipping"), nil
	}

	mockRepo.GetPaymentByIDFunc = func(ctx context.Context, id pgtype.UUID) (repository.Payment, error) {
		return repository.Payment{}, nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrderByNumber(ctx, orderNumber)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if err != nil {
	//     t.Fatalf("Expected success, got error: %v", err)
	// }
}

func TestOrderService_GetOrderByNumber_NotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderNumber := "ORD-20250129-NOTFOUND"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	mockRepo.GetOrderByNumberFunc = func(ctx context.Context, arg repository.GetOrderByNumberParams) (repository.Order, error) {
		return repository.Order{}, sql.ErrNoRows
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrderByNumber(ctx, orderNumber)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
	// Once implemented:
	// if !errors.Is(err, ErrOrderNotFound) {
	//     t.Errorf("Expected ErrOrderNotFound, got: %v", err)
	// }
}

func TestOrderService_GetOrderByNumber_TenantScoping(t *testing.T) {
	ctx := context.Background()
	tenantID := makeTestTenantID()
	orderNumber := "ORD-20250129-OTHER"

	mockRepo := &mockQuerier{}
	mockBilling := billing.NewMockProvider()
	mockShipping := &mockShippingProvider{}

	// Verify tenant scoping is enforced
	mockRepo.GetOrderByNumberFunc = func(ctx context.Context, arg repository.GetOrderByNumberParams) (repository.Order, error) {
		if arg.TenantID.Bytes != tenantID.Bytes {
			return repository.Order{}, sql.ErrNoRows
		}
		return makeTestOrder(), nil
	}

	svc, err := NewOrderService(mockRepo, uuidToString(tenantID), mockBilling, mockShipping)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = svc.GetOrderByNumber(ctx, orderNumber)
	if err == nil {
		t.Error("Expected error (not implemented), got nil")
	}
}

// Helper functions

func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// mockShippingProvider implements shipping.Provider for testing
type mockShippingProvider struct{}

func (m *mockShippingProvider) GetRates(ctx context.Context, params shipping.RateParams) ([]shipping.Rate, error) {
	return []shipping.Rate{
		{
			RateID:      "flat_rate",
			Carrier:     "USPS",
			ServiceName: "Standard Shipping",
			CostCents:   500,
		},
	}, nil
}

func (m *mockShippingProvider) CreateLabel(ctx context.Context, params shipping.LabelParams) (*shipping.Label, error) {
	return nil, shipping.ErrNotImplemented
}

func (m *mockShippingProvider) VoidLabel(ctx context.Context, labelID string) error {
	return shipping.ErrNotImplemented
}

func (m *mockShippingProvider) TrackShipment(ctx context.Context, trackingNumber string) (*shipping.TrackingInfo, error) {
	return nil, shipping.ErrNotImplemented
}

// uuidToString is now defined in order.go and shared between implementation and tests
