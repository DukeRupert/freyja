package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/hiri/internal/billing"
	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v83"
)

// mockBillingProvider implements billing.Provider for testing
type mockBillingProvider struct {
	verifyWebhookSignatureFunc func(payload []byte, signature string, secret string) error
}

func (m *mockBillingProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	if m.verifyWebhookSignatureFunc != nil {
		return m.verifyWebhookSignatureFunc(payload, signature, secret)
	}
	return nil
}

// Stub implementations for other required interface methods
func (m *mockBillingProvider) CreatePaymentIntent(ctx context.Context, params billing.CreatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) GetPaymentIntent(ctx context.Context, params billing.GetPaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) UpdatePaymentIntent(ctx context.Context, params billing.UpdatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	return errors.New("not implemented")
}
func (m *mockBillingProvider) CreateCustomer(ctx context.Context, params billing.CreateCustomerParams) (*billing.Customer, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) GetCustomer(ctx context.Context, customerID string) (*billing.Customer, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) GetCustomerByEmail(ctx context.Context, email string) (*billing.Customer, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) UpdateCustomer(ctx context.Context, customerID string, params billing.UpdateCustomerParams) (*billing.Customer, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CreateSubscription(ctx context.Context, params billing.CreateSubscriptionParams) (*billing.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CreateProduct(ctx context.Context, params billing.CreateProductParams) (*billing.Product, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CreateRecurringPrice(ctx context.Context, params billing.CreateRecurringPriceParams) (*billing.Price, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) GetSubscription(ctx context.Context, params billing.GetSubscriptionParams) (*billing.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) PauseSubscription(ctx context.Context, params billing.PauseSubscriptionParams) (*billing.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) ResumeSubscription(ctx context.Context, params billing.ResumeSubscriptionParams) (*billing.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CancelSubscription(ctx context.Context, params billing.CancelSubscriptionParams) error {
	return errors.New("not implemented")
}
func (m *mockBillingProvider) CreateCustomerPortalSession(ctx context.Context, params billing.CreatePortalSessionParams) (*billing.PortalSession, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) GetInvoice(ctx context.Context, params billing.GetInvoiceParams) (*billing.Invoice, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) RefundPayment(ctx context.Context, params billing.RefundParams) (*billing.Refund, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) CreateInvoice(ctx context.Context, params billing.CreateInvoiceParams) (*billing.Invoice, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) AddInvoiceItem(ctx context.Context, params billing.AddInvoiceItemParams) error {
	return errors.New("not implemented")
}
func (m *mockBillingProvider) FinalizeInvoice(ctx context.Context, params billing.FinalizeInvoiceParams) (*billing.Invoice, error) {
	return nil, errors.New("not implemented")
}
func (m *mockBillingProvider) SendInvoice(ctx context.Context, params billing.SendInvoiceParams) error {
	return errors.New("not implemented")
}
func (m *mockBillingProvider) VoidInvoice(ctx context.Context, params billing.VoidInvoiceParams) error {
	return errors.New("not implemented")
}
func (m *mockBillingProvider) PayInvoice(ctx context.Context, params billing.PayInvoiceParams) (*billing.Invoice, error) {
	return nil, errors.New("not implemented")
}

// mockOrderService implements domain.OrderService for testing
type mockOrderService struct {
	createOrderFromPaymentIntentFunc func(ctx context.Context, paymentIntentID string) (*domain.OrderDetail, error)
}

func (m *mockOrderService) CreateOrderFromPaymentIntent(ctx context.Context, paymentIntentID string) (*domain.OrderDetail, error) {
	if m.createOrderFromPaymentIntentFunc != nil {
		return m.createOrderFromPaymentIntentFunc(ctx, paymentIntentID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOrderService) GetOrder(ctx context.Context, orderID string) (*domain.OrderDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockOrderService) GetOrderByNumber(ctx context.Context, orderNumber string) (*domain.OrderDetail, error) {
	return nil, errors.New("not implemented")
}

// mockSubscriptionService implements domain.SubscriptionService for testing
type mockSubscriptionService struct {
	createOrderFromSubscriptionInvoiceFunc func(ctx context.Context, invoiceID string, tenantID pgtype.UUID) (*domain.OrderDetail, error)
	syncSubscriptionFromWebhookFunc        func(ctx context.Context, params domain.SyncSubscriptionParams) error
}

func (m *mockSubscriptionService) CreateOrderFromSubscriptionInvoice(ctx context.Context, invoiceID string, tenantID pgtype.UUID) (*domain.OrderDetail, error) {
	if m.createOrderFromSubscriptionInvoiceFunc != nil {
		return m.createOrderFromSubscriptionInvoiceFunc(ctx, invoiceID, tenantID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) SyncSubscriptionFromWebhook(ctx context.Context, params domain.SyncSubscriptionParams) error {
	if m.syncSubscriptionFromWebhookFunc != nil {
		return m.syncSubscriptionFromWebhookFunc(ctx, params)
	}
	return errors.New("not implemented")
}

func (m *mockSubscriptionService) CreateSubscription(ctx context.Context, params domain.CreateSubscriptionParams) (*domain.SubscriptionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) GetSubscription(ctx context.Context, params domain.GetSubscriptionParams) (*domain.SubscriptionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) ListSubscriptionsForUser(ctx context.Context, params domain.ListSubscriptionsParams) ([]domain.SubscriptionSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) PauseSubscription(ctx context.Context, params domain.PauseSubscriptionParams) (*domain.SubscriptionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) ResumeSubscription(ctx context.Context, params domain.ResumeSubscriptionParams) (*domain.SubscriptionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) CancelSubscription(ctx context.Context, params domain.CancelSubscriptionParams) (*domain.SubscriptionDetail, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSubscriptionService) CreateCustomerPortalSession(ctx context.Context, params domain.PortalSessionParams) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSubscriptionService) GetSubscriptionCountsForUser(ctx context.Context, tenantID, userID pgtype.UUID) (domain.SubscriptionCounts, error) {
	return domain.SubscriptionCounts{}, errors.New("not implemented")
}

// Helper functions

func mustMarshalEvent(t *testing.T, event stripe.Event) []byte {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	return data
}

func createTestPaymentIntentEvent(tenantID, cartID string) stripe.Event {
	eventType := stripe.EventType("payment_intent.succeeded")
	return stripe.Event{
		ID:   "evt_test_123",
		Type: eventType,
		Data: &stripe.EventData{
			Raw: json.RawMessage(`{
				"id": "pi_test_123",
				"amount": 2500,
				"currency": "usd",
				"status": "succeeded",
				"metadata": {
					"tenant_id": "` + tenantID + `",
					"cart_id": "` + cartID + `",
					"order_type": "retail"
				}
			}`),
		},
	}
}

func createTestInvoiceEvent(eventType, tenantID, subscriptionID string) stripe.Event {
	evtType := stripe.EventType(eventType)
	return stripe.Event{
		ID:   "evt_test_invoice_123",
		Type: evtType,
		Data: &stripe.EventData{
			Raw: json.RawMessage(`{
				"id": "in_test_123",
				"amount_paid": 2500,
				"currency": "usd",
				"parent": {
					"subscription_details": {
						"subscription": {
							"id": "` + subscriptionID + `",
							"metadata": {
								"tenant_id": "` + tenantID + `"
							}
						}
					}
				}
			}`),
		},
	}
}

func createTestSubscriptionEvent(eventType, tenantID, subscriptionID, status string) stripe.Event {
	evtType := stripe.EventType(eventType)
	return stripe.Event{
		ID:   "evt_test_sub_123",
		Type: evtType,
		Data: &stripe.EventData{
			Raw: json.RawMessage(`{
				"id": "` + subscriptionID + `",
				"status": "` + status + `",
				"metadata": {
					"tenant_id": "` + tenantID + `"
				}
			}`),
		},
	}
}

// Tests

func TestStripeHandler_HandleWebhook_Security(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		signature      string
		verifyError    error
		expectedStatus int
		description    string
	}{
		{
			name:           "rejects_GET_request",
			method:         http.MethodGet,
			signature:      "valid_signature",
			verifyError:    nil,
			expectedStatus: http.StatusBadRequest,
			description:    "Only POST requests should be accepted",
		},
		{
			name:           "rejects_PUT_request",
			method:         http.MethodPut,
			signature:      "valid_signature",
			verifyError:    nil,
			expectedStatus: http.StatusBadRequest,
			description:    "Only POST requests should be accepted",
		},
		{
			name:           "rejects_missing_signature",
			method:         http.MethodPost,
			signature:      "",
			verifyError:    nil,
			expectedStatus: http.StatusBadRequest,
			description:    "Missing Stripe-Signature header must be rejected",
		},
		{
			name:           "rejects_invalid_signature",
			method:         http.MethodPost,
			signature:      "invalid_signature",
			verifyError:    errors.New("signature verification failed"),
			expectedStatus: http.StatusUnauthorized,
			description:    "Invalid signature must be rejected with 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return tt.verifyError
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				&mockOrderService{},
				&mockSubscriptionService{},
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      "tenant_123",
					TestMode:      false,
				},
			)

			// Create test event
			event := createTestPaymentIntentEvent("tenant_123", "cart_123")
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(tt.method, "/webhooks/stripe", bytes.NewReader(payload))
			if tt.signature != "" {
				req.Header.Set("Stripe-Signature", tt.signature)
			}

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert
			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_PaymentIntentSucceeded(t *testing.T) {
	tests := []struct {
		name                string
		tenantID            string
		cartID              string
		configTenantID      string
		testMode            bool
		orderServiceError   error
		expectServiceCall   bool
		expectOrderCreation bool
		description         string
	}{
		{
			name:                "creates_order_on_matching_tenant",
			tenantID:            "tenant_123",
			cartID:              "cart_123",
			configTenantID:      "tenant_123",
			testMode:            false,
			orderServiceError:   nil,
			expectServiceCall:   true,
			expectOrderCreation: true,
			description:         "Order should be created when tenant_id matches",
		},
		{
			name:                "rejects_mismatched_tenant_in_production",
			tenantID:            "tenant_456",
			cartID:              "cart_123",
			configTenantID:      "tenant_123",
			testMode:            false,
			orderServiceError:   nil,
			expectServiceCall:   false,
			expectOrderCreation: false,
			description:         "Mismatched tenant_id should skip processing in production",
		},
		{
			name:                "allows_mismatched_tenant_in_test_mode",
			tenantID:            "tenant_456",
			cartID:              "cart_123",
			configTenantID:      "tenant_123",
			testMode:            true,
			orderServiceError:   nil,
			expectServiceCall:   false,
			expectOrderCreation: false,
			description:         "Test mode should log warning but not fail on tenant mismatch",
		},
		{
			name:                "handles_idempotent_duplicate",
			tenantID:            "tenant_123",
			cartID:              "cart_123",
			configTenantID:      "tenant_123",
			testMode:            false,
			orderServiceError:   domain.ErrPaymentAlreadyProcessed,
			expectServiceCall:   true,
			expectOrderCreation: false,
			description:         "ErrPaymentAlreadyProcessed should be treated as success",
		},
		{
			name:                "handles_service_error",
			tenantID:            "tenant_123",
			cartID:              "cart_123",
			configTenantID:      "tenant_123",
			testMode:            false,
			orderServiceError:   errors.New("database error"),
			expectServiceCall:   true,
			expectOrderCreation: false,
			description:         "Service errors should be logged but webhook returns 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			serviceCalled := false
			orderCreated := false

			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil // Valid signature
				},
			}

			mockOrderSvc := &mockOrderService{
				createOrderFromPaymentIntentFunc: func(ctx context.Context, paymentIntentID string) (*domain.OrderDetail, error) {
					serviceCalled = true
					if tt.orderServiceError != nil {
						return nil, tt.orderServiceError
					}
					orderCreated = true
					return &domain.OrderDetail{
						Order: repository.Order{
							OrderNumber: "ORD-123",
							TotalCents:  2500,
							Currency:    "usd",
						},
					}, nil
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				mockOrderSvc,
				&mockSubscriptionService{},
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      tt.configTenantID,
					TestMode:      tt.testMode,
				},
			)

			// Create test event
			event := createTestPaymentIntentEvent(tt.tenantID, tt.cartID)
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert - Always returns 200 to Stripe
			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tt.description, rr.Code)
			}

			// Assert service call expectations
			if serviceCalled != tt.expectServiceCall {
				t.Errorf("%s: expected service call = %v, got %v", tt.description, tt.expectServiceCall, serviceCalled)
			}

			if orderCreated != tt.expectOrderCreation {
				t.Errorf("%s: expected order creation = %v, got %v", tt.description, tt.expectOrderCreation, orderCreated)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_InvoicePaymentSucceeded(t *testing.T) {
	validTenantUUID := pgtype.UUID{}
	_ = validTenantUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name                string
		tenantID            string
		subscriptionID      string
		configTenantID      string
		testMode            bool
		serviceError        error
		expectServiceCall   bool
		expectOrderCreation bool
		description         string
	}{
		{
			name:                "creates_subscription_order_on_matching_tenant",
			tenantID:            "123e4567-e89b-12d3-a456-426614174000",
			subscriptionID:      "sub_123",
			configTenantID:      "123e4567-e89b-12d3-a456-426614174000",
			testMode:            false,
			serviceError:        nil,
			expectServiceCall:   true,
			expectOrderCreation: true,
			description:         "Subscription order should be created when tenant_id matches",
		},
		{
			name:                "rejects_mismatched_tenant",
			tenantID:            "123e4567-e89b-12d3-a456-426614174999",
			subscriptionID:      "sub_123",
			configTenantID:      "123e4567-e89b-12d3-a456-426614174000",
			testMode:            false,
			serviceError:        nil,
			expectServiceCall:   false,
			expectOrderCreation: false,
			description:         "Mismatched tenant_id should skip processing",
		},
		{
			name:                "handles_idempotent_duplicate",
			tenantID:            "123e4567-e89b-12d3-a456-426614174000",
			subscriptionID:      "sub_123",
			configTenantID:      "123e4567-e89b-12d3-a456-426614174000",
			testMode:            false,
			serviceError:        domain.ErrInvoiceAlreadyProcessed,
			expectServiceCall:   true,
			expectOrderCreation: false,
			description:         "ErrInvoiceAlreadyProcessed should be treated as success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			serviceCalled := false
			orderCreated := false

			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil
				},
			}

			mockSubSvc := &mockSubscriptionService{
				createOrderFromSubscriptionInvoiceFunc: func(ctx context.Context, invoiceID string, tenantID pgtype.UUID) (*domain.OrderDetail, error) {
					serviceCalled = true
					if tt.serviceError != nil {
						return nil, tt.serviceError
					}
					orderCreated = true
					return &domain.OrderDetail{
						Order: repository.Order{
							OrderNumber: "ORD-SUB-123",
							TotalCents:  2500,
							Currency:    "usd",
						},
					}, nil
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				&mockOrderService{},
				mockSubSvc,
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      tt.configTenantID,
					TestMode:      tt.testMode,
				},
			)

			// Create test event
			event := createTestInvoiceEvent("invoice.payment_succeeded", tt.tenantID, tt.subscriptionID)
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert
			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tt.description, rr.Code)
			}

			if serviceCalled != tt.expectServiceCall {
				t.Errorf("%s: expected service call = %v, got %v", tt.description, tt.expectServiceCall, serviceCalled)
			}

			if orderCreated != tt.expectOrderCreation {
				t.Errorf("%s: expected order creation = %v, got %v", tt.description, tt.expectOrderCreation, orderCreated)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_SubscriptionUpdated(t *testing.T) {
	validTenantUUID := pgtype.UUID{}
	_ = validTenantUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name              string
		tenantID          string
		subscriptionID    string
		status            string
		configTenantID    string
		testMode          bool
		syncError         error
		expectServiceCall bool
		description       string
	}{
		{
			name:              "syncs_subscription_on_matching_tenant",
			tenantID:          "123e4567-e89b-12d3-a456-426614174000",
			subscriptionID:    "sub_123",
			status:            "active",
			configTenantID:    "123e4567-e89b-12d3-a456-426614174000",
			testMode:          false,
			syncError:         nil,
			expectServiceCall: true,
			description:       "Subscription should be synced when tenant_id matches",
		},
		{
			name:              "rejects_mismatched_tenant",
			tenantID:          "123e4567-e89b-12d3-a456-426614174999",
			subscriptionID:    "sub_123",
			status:            "active",
			configTenantID:    "123e4567-e89b-12d3-a456-426614174000",
			testMode:          false,
			syncError:         nil,
			expectServiceCall: false,
			description:       "Mismatched tenant_id should skip sync",
		},
		{
			name:              "allows_mismatched_tenant_in_test_mode",
			tenantID:          "123e4567-e89b-12d3-a456-426614174999",
			subscriptionID:    "sub_123",
			status:            "active",
			configTenantID:    "123e4567-e89b-12d3-a456-426614174000",
			testMode:          true,
			syncError:         nil,
			expectServiceCall: false,
			description:       "Test mode should allow tenant mismatch without calling service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			serviceCalled := false

			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil
				},
			}

			mockSubSvc := &mockSubscriptionService{
				syncSubscriptionFromWebhookFunc: func(ctx context.Context, params domain.SyncSubscriptionParams) error {
					serviceCalled = true
					if params.ProviderSubscriptionID != tt.subscriptionID {
						t.Errorf("expected subscription ID %s, got %s", tt.subscriptionID, params.ProviderSubscriptionID)
					}
					if params.EventType != "customer.subscription.updated" {
						t.Errorf("expected event type customer.subscription.updated, got %s", params.EventType)
					}
					return tt.syncError
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				&mockOrderService{},
				mockSubSvc,
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      tt.configTenantID,
					TestMode:      tt.testMode,
				},
			)

			// Create test event
			event := createTestSubscriptionEvent("customer.subscription.updated", tt.tenantID, tt.subscriptionID, tt.status)
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert
			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tt.description, rr.Code)
			}

			if serviceCalled != tt.expectServiceCall {
				t.Errorf("%s: expected service call = %v, got %v", tt.description, tt.expectServiceCall, serviceCalled)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_SubscriptionDeleted(t *testing.T) {
	validTenantUUID := pgtype.UUID{}
	_ = validTenantUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name              string
		tenantID          string
		subscriptionID    string
		configTenantID    string
		testMode          bool
		syncError         error
		expectServiceCall bool
		description       string
	}{
		{
			name:              "syncs_deleted_subscription",
			tenantID:          "123e4567-e89b-12d3-a456-426614174000",
			subscriptionID:    "sub_123",
			configTenantID:    "123e4567-e89b-12d3-a456-426614174000",
			testMode:          false,
			syncError:         nil,
			expectServiceCall: true,
			description:       "Deleted subscription should be synced when tenant_id matches",
		},
		{
			name:              "rejects_mismatched_tenant",
			tenantID:          "123e4567-e89b-12d3-a456-426614174999",
			subscriptionID:    "sub_123",
			configTenantID:    "123e4567-e89b-12d3-a456-426614174000",
			testMode:          false,
			syncError:         nil,
			expectServiceCall: false,
			description:       "Mismatched tenant_id should skip sync",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			serviceCalled := false

			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil
				},
			}

			mockSubSvc := &mockSubscriptionService{
				syncSubscriptionFromWebhookFunc: func(ctx context.Context, params domain.SyncSubscriptionParams) error {
					serviceCalled = true
					if params.ProviderSubscriptionID != tt.subscriptionID {
						t.Errorf("expected subscription ID %s, got %s", tt.subscriptionID, params.ProviderSubscriptionID)
					}
					if params.EventType != "customer.subscription.deleted" {
						t.Errorf("expected event type customer.subscription.deleted, got %s", params.EventType)
					}
					return tt.syncError
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				&mockOrderService{},
				mockSubSvc,
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      tt.configTenantID,
					TestMode:      tt.testMode,
				},
			)

			// Create test event
			event := createTestSubscriptionEvent("customer.subscription.deleted", tt.tenantID, tt.subscriptionID, "canceled")
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert
			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tt.description, rr.Code)
			}

			if serviceCalled != tt.expectServiceCall {
				t.Errorf("%s: expected service call = %v, got %v", tt.description, tt.expectServiceCall, serviceCalled)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		eventType      string
		malformedJSON  bool
		expectedStatus int
		description    string
	}{
		{
			name:           "handles_malformed_json",
			eventType:      "payment_intent.succeeded",
			malformedJSON:  true,
			expectedStatus: http.StatusBadRequest,
			description:    "Malformed JSON should return 400",
		},
		{
			name:           "handles_unhandled_event_type",
			eventType:      string(stripe.EventTypeAccountUpdated),
			malformedJSON:  false,
			expectedStatus: http.StatusOK,
			description:    "Unknown event types should return 200 (logged, not failed)",
		},
		{
			name:           "handles_payment_intent_created",
			eventType:      string(stripe.EventTypePaymentIntentCreated),
			malformedJSON:  false,
			expectedStatus: http.StatusOK,
			description:    "payment_intent.created should return 200 (no action needed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				&mockOrderService{},
				&mockSubscriptionService{},
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      "tenant_123",
					TestMode:      false,
				},
			)

			// Create payload
			var payload []byte
			if tt.malformedJSON {
				payload = []byte(`{"invalid json"`)
			} else {
				event := stripe.Event{
					ID:   "evt_test_123",
					Type: stripe.EventType(tt.eventType),
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{}`),
					},
				}
				payload = mustMarshalEvent(t, event)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert
			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_AlwaysReturns200ForValidEvents(t *testing.T) {
	// This test ensures that webhook handler ALWAYS returns 200 to Stripe
	// for valid, authenticated events - even when internal processing fails.
	// This prevents Stripe from retrying events that will fail again.

	tests := []struct {
		name           string
		eventType      string
		serviceError   error
		expectedStatus int
		description    string
	}{
		{
			name:           "returns_200_on_order_service_error",
			eventType:      "payment_intent.succeeded",
			serviceError:   errors.New("database connection lost"),
			expectedStatus: http.StatusOK,
			description:    "Service errors should log but return 200 to prevent retries",
		},
		{
			name:           "returns_200_on_idempotent_duplicate",
			eventType:      "payment_intent.succeeded",
			serviceError:   domain.ErrPaymentAlreadyProcessed,
			expectedStatus: http.StatusOK,
			description:    "Duplicate events should return 200 (idempotent)",
		},
		{
			name:           "returns_200_on_success",
			eventType:      "payment_intent.succeeded",
			serviceError:   nil,
			expectedStatus: http.StatusOK,
			description:    "Successful processing should return 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockProvider := &mockBillingProvider{
				verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
					return nil
				},
			}

			mockOrderSvc := &mockOrderService{
				createOrderFromPaymentIntentFunc: func(ctx context.Context, paymentIntentID string) (*domain.OrderDetail, error) {
					if tt.serviceError != nil {
						return nil, tt.serviceError
					}
					return &domain.OrderDetail{
						Order: repository.Order{
							OrderNumber: "ORD-123",
							TotalCents:  2500,
							Currency:    "usd",
						},
					}, nil
				},
			}

			handler := NewStripeHandler(
				mockProvider,
				mockOrderSvc,
				&mockSubscriptionService{},
				StripeWebhookConfig{
					WebhookSecret: "test_secret",
					TenantID:      "tenant_123",
					TestMode:      false,
				},
			)

			// Create test event
			event := createTestPaymentIntentEvent("tenant_123", "cart_123")
			payload := mustMarshalEvent(t, event)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
			req.Header.Set("Stripe-Signature", "valid_signature")

			// Execute
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Assert - MUST always return 200 for valid, authenticated events
			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, rr.Code)
			}

			// Verify response body contains "received": true
			var response map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if received, ok := response["received"].(bool); !ok || !received {
				t.Errorf("%s: expected response {\"received\": true}, got %v", tt.description, response)
			}
		})
	}
}

func TestStripeHandler_HandleWebhook_TestModeSkipsWithoutMetadata(t *testing.T) {
	// In test mode, events triggered by Stripe CLI may not have required metadata.
	// These should be logged but not fail.

	mockProvider := &mockBillingProvider{
		verifyWebhookSignatureFunc: func(payload []byte, signature string, secret string) error {
			return nil
		},
	}

	serviceCalled := false
	mockOrderSvc := &mockOrderService{
		createOrderFromPaymentIntentFunc: func(ctx context.Context, paymentIntentID string) (*domain.OrderDetail, error) {
			serviceCalled = true
			return nil, errors.New("should not be called")
		},
	}

	handler := NewStripeHandler(
		mockProvider,
		mockOrderSvc,
		&mockSubscriptionService{},
		StripeWebhookConfig{
			WebhookSecret: "test_secret",
			TenantID:      "tenant_123",
			TestMode:      true, // Test mode enabled
		},
	)

	// Create event with missing cart_id metadata
	event := createTestPaymentIntentEvent("tenant_123", "") // Empty cart_id
	payload := mustMarshalEvent(t, event)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "valid_signature")

	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	// Should return 200 and skip order creation
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if serviceCalled {
		t.Error("expected service not to be called when cart_id is missing in test mode")
	}
}
