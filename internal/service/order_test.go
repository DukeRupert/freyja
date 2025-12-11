package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dukerupert/hiri/internal/billing"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/shipping"
	"github.com/dukerupert/hiri/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// newUUID creates a new pgtype.UUID from a random UUID
func newUUID() pgtype.UUID {
	id := uuid.New()
	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], id[:])
	pgUUID.Valid = true
	return pgUUID
}

// contextWithTenant creates a context with tenant ID
func contextWithTenant(tenantID pgtype.UUID) context.Context {
	ctx := context.Background()
	t := &tenant.Tenant{
		ID:     tenantID,
		Slug:   "test-tenant",
		Name:   "Test Tenant",
		Status: "active",
	}
	return tenant.NewContext(ctx, t)
}

// createTestPaymentIntent creates a test payment intent with standard fields
func createTestPaymentIntent(cartID string, status string) *billing.PaymentIntent {
	shippingJSON, _ := json.Marshal(map[string]string{
		"full_name":     "John Doe",
		"address_line1": "123 Main St",
		"city":          "Seattle",
		"state":         "WA",
		"postal_code":   "98101",
		"country":       "US",
	})

	billingJSON, _ := json.Marshal(map[string]string{
		"full_name":     "John Doe",
		"address_line1": "123 Main St",
		"city":          "Seattle",
		"state":         "WA",
		"postal_code":   "98101",
		"country":       "US",
	})

	return &billing.PaymentIntent{
		ID:            "pi_test_123",
		AmountCents:   5000,
		Currency:      "usd",
		Status:        status,
		TaxCents:      450,
		ShippingCents: 550,
		ReceiptEmail:  "customer@example.com",
		Metadata: map[string]string{
			"cart_id":          cartID,
			"shipping_address": string(shippingJSON),
			"billing_address":  string(billingJSON),
		},
		CreatedAt: time.Now(),
	}
}

// createTestCart creates a test cart with the given tenant and status
func createTestCart(tenantID pgtype.UUID, status string) repository.Cart {
	return repository.Cart{
		ID:       newUUID(),
		TenantID: tenantID,
		UserID:   pgtype.UUID{Valid: false}, // Guest cart
		Status:   status,
	}
}

// createTestCartItems creates test cart items
func createTestCartItems() []repository.GetCartItemsRow {
	return []repository.GetCartItemsRow{
		{
			ProductSkuID:   newUUID(),
			ProductName:    "Ethiopian Yirgacheffe",
			Sku:            "ETH-YIRG-12OZ-WB",
			Quantity:       2,
			UnitPriceCents: 1800,
			Grind:          "whole_bean",
		},
		{
			ProductSkuID:   newUUID(),
			ProductName:    "Colombian Supremo",
			Sku:            "COL-SUPR-12OZ-MED",
			Quantity:       1,
			UnitPriceCents: 1600,
			Grind:          "medium",
		},
	}
}

// setupMockDefaults configures reasonable defaults for a mock that allows the order creation flow to succeed
func setupMockDefaults(mockRepo *repository.MockQuerier, tenantID pgtype.UUID, cart repository.Cart, cartItems []repository.GetCartItemsRow) {
	// Most calls return successful defaults
	mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows).AnyTimes()
	mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil).AnyTimes()
	mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil).AnyTimes()
	mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
			return repository.Address{
				ID:           newUUID(),
				TenantID:     arg.TenantID,
				AddressType:  arg.AddressType,
				AddressLine1: arg.AddressLine1,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows).AnyTimes()
	mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
			return repository.User{
				ID:       newUUID(),
				TenantID: arg.TenantID,
				Email:    arg.Email,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows).AnyTimes()
	mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreateBillingCustomerParams) (repository.BillingCustomer, error) {
			return repository.BillingCustomer{
				ID:                 newUUID(),
				TenantID:           arg.TenantID,
				UserID:             arg.UserID,
				ProviderCustomerID: arg.ProviderCustomerID,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error) {
			return repository.Payment{
				ID:                newUUID(),
				TenantID:          arg.TenantID,
				ProviderPaymentID: arg.ProviderPaymentID,
				AmountCents:       arg.AmountCents,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
			return repository.Order{
				ID:            newUUID(),
				TenantID:      arg.TenantID,
				OrderNumber:   arg.OrderNumber,
				Status:        arg.Status,
				SubtotalCents: arg.SubtotalCents,
				TaxCents:      arg.TaxCents,
				ShippingCents: arg.ShippingCents,
				TotalCents:    arg.TotalCents,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error) {
			return repository.OrderItem{
				ID:       newUUID(),
				TenantID: arg.TenantID,
			}, nil
		}).AnyTimes()
	mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil).AnyTimes()
	mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRepo.EXPECT().GetAddressByID(gomock.Any(), gomock.Any()).Return(repository.Address{}, nil).AnyTimes()
	mockRepo.EXPECT().GetPaymentByID(gomock.Any(), gomock.Any()).Return(repository.Payment{}, nil).AnyTimes()
	mockRepo.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Return(repository.Order{}, nil).AnyTimes()
}

// =============================================================================
// TEST SUITE
// =============================================================================

// Test_CreateOrderFromPaymentIntent_Idempotency verifies that calling the same
// payment intent twice returns the existing order without creating a duplicate
func Test_CreateOrderFromPaymentIntent_Idempotency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	ctx := contextWithTenant(tenantID)

	cart := createTestCart(tenantID, "active")
	cartIDStr := uuidToString(cart.ID)
	pi := createTestPaymentIntent(cartIDStr, "succeeded")

	// Existing order that should be returned
	existingOrder := repository.Order{
		ID:          newUUID(),
		TenantID:    tenantID,
		OrderNumber: "ORD-20250101-ABCD",
		Status:      "pending",
	}

	mockRepo := repository.NewMockQuerier(ctrl)

	// First call - order already exists (returns existing order, not ErrNoRows)
	mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(existingOrder, nil).Times(2)
	mockRepo.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Return(existingOrder, nil).Times(2)
	mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil).Times(2)
	mockRepo.EXPECT().GetAddressByID(gomock.Any(), gomock.Any()).Return(repository.Address{}, nil).Times(4)
	mockRepo.EXPECT().GetPaymentByID(gomock.Any(), gomock.Any()).Return(repository.Payment{}, nil).Times(2)

	mockBilling := billing.NewMockProvider()
	mockShipping := shipping.NewMockProvider()

	svc := NewOrderService(mockRepo, mockBilling, mockShipping)

	// First call
	order1, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
	require.NoError(t, err)
	require.NotNil(t, order1)
	assert.Equal(t, existingOrder.ID, order1.Order.ID)

	// Second call should return the same order
	order2, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
	require.NoError(t, err)
	require.NotNil(t, order2)
	assert.Equal(t, existingOrder.ID, order2.Order.ID)
	assert.Equal(t, order1.Order.ID, order2.Order.ID, "should return same order on duplicate call")
}

// Test_CreateOrderFromPaymentIntent_PaymentValidation tests all payment validation scenarios
func Test_CreateOrderFromPaymentIntent_PaymentValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupPayment  func() *billing.PaymentIntent
		expectedError error
	}{
		{
			name: "rejects payment status requires_payment_method",
			setupPayment: func() *billing.PaymentIntent {
				pi := createTestPaymentIntent("cart_123", "requires_payment_method")
				return pi
			},
			expectedError: ErrPaymentNotSucceeded,
		},
		{
			name: "rejects payment status processing",
			setupPayment: func() *billing.PaymentIntent {
				pi := createTestPaymentIntent("cart_123", "processing")
				return pi
			},
			expectedError: ErrPaymentNotSucceeded,
		},
		{
			name: "rejects payment status canceled",
			setupPayment: func() *billing.PaymentIntent {
				pi := createTestPaymentIntent("cart_123", "canceled")
				return pi
			},
			expectedError: ErrPaymentNotSucceeded,
		},
		{
			name: "rejects missing cart_id in metadata",
			setupPayment: func() *billing.PaymentIntent {
				pi := createTestPaymentIntent("cart_123", "succeeded")
				delete(pi.Metadata, "cart_id")
				return pi
			},
			expectedError: ErrMissingCartID,
		},
		{
			name: "rejects empty cart_id in metadata",
			setupPayment: func() *billing.PaymentIntent {
				pi := createTestPaymentIntent("cart_123", "succeeded")
				pi.Metadata["cart_id"] = ""
				return pi
			},
			expectedError: ErrMissingCartID,
		},
		{
			name: "rejects missing shipping_address in metadata",
			setupPayment: func() *billing.PaymentIntent {
				// Use valid UUID format for cart_id to pass that validation first
				validCartID := uuid.New().String()
				pi := createTestPaymentIntent(validCartID, "succeeded")
				delete(pi.Metadata, "shipping_address")
				return pi
			},
			expectedError: ErrMissingShippingAddress,
		},
		{
			name: "rejects empty shipping_address in metadata",
			setupPayment: func() *billing.PaymentIntent {
				validCartID := uuid.New().String()
				pi := createTestPaymentIntent(validCartID, "succeeded")
				pi.Metadata["shipping_address"] = ""
				return pi
			},
			expectedError: ErrMissingShippingAddress,
		},
		{
			name: "rejects missing billing_address in metadata",
			setupPayment: func() *billing.PaymentIntent {
				validCartID := uuid.New().String()
				pi := createTestPaymentIntent(validCartID, "succeeded")
				delete(pi.Metadata, "billing_address")
				return pi
			},
			expectedError: ErrMissingBillingAddress,
		},
		{
			name: "rejects empty billing_address in metadata",
			setupPayment: func() *billing.PaymentIntent {
				validCartID := uuid.New().String()
				pi := createTestPaymentIntent(validCartID, "succeeded")
				pi.Metadata["billing_address"] = ""
				return pi
			},
			expectedError: ErrMissingBillingAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			ctx := contextWithTenant(tenantID)

			pi := tt.setupPayment()

			mockRepo := repository.NewMockQuerier(ctrl)
			mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows).AnyTimes()

			// For tests that get past cart_id validation, we need to mock the cart lookup
			// and cart items retrieval since address validation happens after those steps
			if tt.expectedError == ErrMissingShippingAddress || tt.expectedError == ErrMissingBillingAddress {
				cart := createTestCart(tenantID, "active")
				cartItems := createTestCartItems()
				mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil).AnyTimes()
				mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil).AnyTimes()
			}

			mockBilling := billing.NewMockProvider()
			mockBilling.PaymentIntents[pi.ID] = pi

			mockShipping := shipping.NewMockProvider()

			svc := NewOrderService(mockRepo, mockBilling, mockShipping)

			order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
			assert.Error(t, err)
			assert.Nil(t, order)
			assert.True(t, errors.Is(err, tt.expectedError), "expected error %v, got %v", tt.expectedError, err)
		})
	}
}

// Test_CreateOrderFromPaymentIntent_TenantIsolation verifies security-critical tenant isolation
func Test_CreateOrderFromPaymentIntent_TenantIsolation(t *testing.T) {
	tenantID1 := newUUID()
	tenantID2 := newUUID()

	t.Run("rejects cart from different tenant", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Context has tenantID1
		ctx := contextWithTenant(tenantID1)

		// But cart belongs to tenantID2
		cart := createTestCart(tenantID2, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.True(t, errors.Is(err, ErrTenantMismatch), "should reject cart from different tenant")
	})

	t.Run("created records have correct tenant_id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		cartItems := createTestCartItems()

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)

		// Verify tenant_id is passed correctly to all create methods
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				assert.Equal(t, tenantID, arg.TenantID, "address must have correct tenant_id")
				return repository.Address{ID: newUUID(), TenantID: arg.TenantID, AddressType: arg.AddressType}, nil
			}).Times(2)

		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
				assert.Equal(t, tenantID, arg.TenantID, "user must have correct tenant_id")
				return repository.User{ID: newUUID(), TenantID: arg.TenantID, Email: arg.Email}, nil
			})

		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateBillingCustomerParams) (repository.BillingCustomer, error) {
				assert.Equal(t, tenantID, arg.TenantID, "billing customer must have correct tenant_id")
				return repository.BillingCustomer{ID: newUUID(), TenantID: arg.TenantID}, nil
			})

		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error) {
				assert.Equal(t, tenantID, arg.TenantID, "payment must have correct tenant_id")
				return repository.Payment{ID: newUUID(), TenantID: arg.TenantID}, nil
			})

		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
				assert.Equal(t, tenantID, arg.TenantID, "order must have correct tenant_id")
				return repository.Order{ID: newUUID(), TenantID: arg.TenantID}, nil
			})

		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error) {
				assert.Equal(t, tenantID, arg.TenantID, "order item must have correct tenant_id")
				return repository.OrderItem{ID: newUUID(), TenantID: arg.TenantID}, nil
			}).Times(len(cartItems))

		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil).Times(len(cartItems))
		mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).Return(nil)
		mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		_, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		require.NoError(t, err, "order creation should succeed with correct tenant_id on all records")
	})
}

// Test_CreateOrderFromPaymentIntent_CartStateManagement verifies cart state transitions
func Test_CreateOrderFromPaymentIntent_CartStateManagement(t *testing.T) {
	t.Run("rejects already-converted cart", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "converted")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.True(t, errors.Is(err, ErrCartAlreadyConverted), "should reject already-converted cart")
	})

	t.Run("rejects empty cart", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return([]repository.GetCartItemsRow{}, nil) // Empty cart

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.True(t, errors.Is(err, ErrEmptyCart), "should reject empty cart")
	})

	t.Run("marks cart as converted after order creation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		cartItems := createTestCartItems()

		cartStatusUpdated := false

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				return repository.Address{ID: newUUID(), AddressType: arg.AddressType}, nil
			}).Times(2)
		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(repository.User{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{ID: newUUID()}, nil).Times(len(cartItems))
		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil).Times(len(cartItems))
		mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.UpdateCartStatusParams) error {
				assert.Equal(t, cart.ID, arg.ID)
				assert.Equal(t, "converted", arg.Status, "cart should be marked as converted")
				cartStatusUpdated = true
				return nil
			})
		mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		_, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		require.NoError(t, err)
		assert.True(t, cartStatusUpdated, "cart status should have been updated to 'converted'")
	})
}

// Test_CreateOrderFromPaymentIntent_InventoryManagement verifies inventory decrements
func Test_CreateOrderFromPaymentIntent_InventoryManagement(t *testing.T) {
	t.Run("decrements inventory for each item", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		cartItems := createTestCartItems()

		decrementedSKUs := make(map[string]int32)

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				return repository.Address{ID: newUUID(), AddressType: arg.AddressType}, nil
			}).Times(2)
		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(repository.User{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{ID: newUUID()}, nil).Times(len(cartItems))
		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.DecrementSKUStockParams) error {
				skuKey := uuidToString(arg.ID)
				decrementedSKUs[skuKey] = arg.InventoryQuantity
				return nil
			}).Times(len(cartItems))
		mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).Return(nil)
		mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		_, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		require.NoError(t, err)

		// Verify all items were decremented
		assert.Len(t, decrementedSKUs, len(cartItems), "all cart items should decrement inventory")
		for _, item := range cartItems {
			skuKey := uuidToString(item.ProductSkuID)
			assert.Equal(t, item.Quantity, decrementedSKUs[skuKey], "SKU %s should be decremented by quantity %d", skuKey, item.Quantity)
		}
	})

	t.Run("returns error on insufficient stock", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		cartItems := createTestCartItems()

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				return repository.Address{ID: newUUID(), AddressType: arg.AddressType}, nil
			}).Times(2)
		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(repository.User{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{ID: newUUID()}, nil).Times(len(cartItems))
		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		// Simulate insufficient stock error on first decrement
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(errors.New("insufficient stock"))

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "failed to decrement stock")
	})
}

// Test_CreateOrderFromPaymentIntent_OrderDataIntegrity verifies order data is captured correctly
func Test_CreateOrderFromPaymentIntent_OrderDataIntegrity(t *testing.T) {
	t.Run("order totals match payment intent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		pi.AmountCents = 6000
		pi.TaxCents = 450
		pi.ShippingCents = 550

		cartItems := []repository.GetCartItemsRow{
			{
				ProductSkuID:   newUUID(),
				ProductName:    "Test Coffee",
				Sku:            "TEST-12OZ",
				Quantity:       1,
				UnitPriceCents: 5000,
				Grind:          "whole_bean",
			},
		}

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				return repository.Address{ID: newUUID(), AddressType: arg.AddressType}, nil
			}).Times(2)
		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(repository.User{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
				// Verify order totals
				assert.Equal(t, int32(5000), arg.SubtotalCents, "subtotal should match cart items total")
				assert.Equal(t, pi.TaxCents, arg.TaxCents, "tax should match payment intent")
				assert.Equal(t, pi.ShippingCents, arg.ShippingCents, "shipping should match payment intent")
				expectedTotal := int32(5000) + pi.TaxCents + pi.ShippingCents
				assert.Equal(t, expectedTotal, arg.TotalCents, "total should equal subtotal + tax + shipping")
				return repository.Order{ID: newUUID(), TenantID: arg.TenantID}, nil
			})
		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil)
		mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).Return(nil)
		mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		_, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		require.NoError(t, err)
	})

	t.Run("order items capture product details correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tenantID := newUUID()
		ctx := contextWithTenant(tenantID)

		cart := createTestCart(tenantID, "active")
		cartIDStr := uuidToString(cart.ID)
		pi := createTestPaymentIntent(cartIDStr, "succeeded")
		cartItems := createTestCartItems()

		capturedItems := []repository.CreateOrderItemParams{}

		mockRepo := repository.NewMockQuerier(ctrl)
		mockRepo.EXPECT().GetOrderByPaymentIntentID(gomock.Any(), gomock.Any()).Return(repository.Order{}, sql.ErrNoRows)
		mockRepo.EXPECT().GetCartByID(gomock.Any(), gomock.Any()).Return(cart, nil)
		mockRepo.EXPECT().GetCartItems(gomock.Any(), gomock.Any()).Return(cartItems, nil)
		mockRepo.EXPECT().CreateAddress(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateAddressParams) (repository.Address, error) {
				return repository.Address{ID: newUUID(), AddressType: arg.AddressType}, nil
			}).Times(2)
		mockRepo.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Return(repository.User{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(repository.User{ID: newUUID()}, nil)
		mockRepo.EXPECT().GetBillingCustomerByUserID(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{}, pgx.ErrNoRows)
		mockRepo.EXPECT().CreateBillingCustomer(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)
		mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error) {
				capturedItems = append(capturedItems, arg)
				return repository.OrderItem{ID: newUUID()}, nil
			}).Times(len(cartItems))
		mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
		mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil).Times(len(cartItems))
		mockRepo.EXPECT().UpdateCartStatus(gomock.Any(), gomock.Any()).Return(nil)
		mockRepo.EXPECT().UpdateOrderPaymentID(gomock.Any(), gomock.Any()).Return(nil)

		mockBilling := billing.NewMockProvider()
		mockBilling.PaymentIntents[pi.ID] = pi

		mockShipping := shipping.NewMockProvider()

		svc := NewOrderService(mockRepo, mockBilling, mockShipping)

		_, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
		require.NoError(t, err)

		// Verify each order item
		require.Len(t, capturedItems, len(cartItems))
		for i, item := range capturedItems {
			cartItem := cartItems[i]
			assert.Equal(t, cartItem.ProductSkuID, item.ProductSkuID, "SKU ID should match")
			assert.Equal(t, cartItem.ProductName, item.ProductName, "product name should match")
			assert.Equal(t, cartItem.Sku, item.Sku, "SKU code should match")
			assert.Equal(t, cartItem.Quantity, item.Quantity, "quantity should match")
			assert.Equal(t, cartItem.UnitPriceCents, item.UnitPriceCents, "unit price should match")
			expectedTotal := cartItem.Quantity * cartItem.UnitPriceCents
			assert.Equal(t, expectedTotal, item.TotalPriceCents, "total price should equal quantity * unit price")
		}
	})
}

// Test_CreateOrderFromPaymentIntent_SuccessfulFlow tests the complete happy path
func Test_CreateOrderFromPaymentIntent_SuccessfulFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	ctx := contextWithTenant(tenantID)

	cart := createTestCart(tenantID, "active")
	cartIDStr := uuidToString(cart.ID)
	pi := createTestPaymentIntent(cartIDStr, "succeeded")
	cartItems := createTestCartItems()

	mockRepo := repository.NewMockQuerier(ctrl)
	setupMockDefaults(mockRepo, tenantID, cart, cartItems)

	mockBilling := billing.NewMockProvider()
	mockBilling.PaymentIntents[pi.ID] = pi

	mockShipping := shipping.NewMockProvider()

	svc := NewOrderService(mockRepo, mockBilling, mockShipping)

	order, err := svc.CreateOrderFromPaymentIntent(ctx, pi.ID)
	require.NoError(t, err, "successful order creation should not error")
	require.NotNil(t, order, "order should be returned")
	assert.NotEqual(t, uuid.Nil, order.Order.ID, "order should have a valid ID")
	assert.Equal(t, tenantID, order.Order.TenantID, "order should belong to correct tenant")
}
