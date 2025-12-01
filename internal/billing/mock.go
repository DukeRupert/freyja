package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MockProvider is a mock billing provider for testing.
// Simulates successful payment flows without calling Stripe API.
type MockProvider struct {
	// CreatePaymentIntentFunc allows customizing payment intent creation behavior
	CreatePaymentIntentFunc func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error)

	// GetPaymentIntentFunc allows customizing payment intent retrieval behavior
	GetPaymentIntentFunc func(ctx context.Context, params GetPaymentIntentParams) (*PaymentIntent, error)

	// VerifyWebhookSignatureFunc allows customizing webhook verification behavior
	VerifyWebhookSignatureFunc func(payload []byte, signature string, secret string) error

	// CreateCustomerFunc allows customizing customer creation behavior
	CreateCustomerFunc func(ctx context.Context, params CreateCustomerParams) (*Customer, error)

	// GetCustomerByEmailFunc allows customizing customer lookup behavior
	GetCustomerByEmailFunc func(ctx context.Context, email string) (*Customer, error)

	// PaymentIntents stores created payment intents for retrieval
	PaymentIntents map[string]*PaymentIntent

	// Customers stores created customers for retrieval
	Customers map[string]*Customer

	// CallLog tracks method calls for test assertions
	CallLog []string
}

// NewMockProvider creates a new mock billing provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		PaymentIntents: make(map[string]*PaymentIntent),
		Customers:      make(map[string]*Customer),
		CallLog:        []string{},
	}
}

// CreatePaymentIntent creates a mock payment intent.
func (m *MockProvider) CreatePaymentIntent(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("CreatePaymentIntent(%d, %s)", params.AmountCents, params.Currency))

	if m.CreatePaymentIntentFunc != nil {
		return m.CreatePaymentIntentFunc(ctx, params)
	}

	// Default mock behavior: create successful payment intent
	pi := &PaymentIntent{
		ID:           "pi_" + uuid.New().String(),
		ClientSecret: "pi_" + uuid.New().String() + "_secret_" + uuid.New().String(),
		AmountCents:  params.AmountCents,
		Currency:     params.Currency,
		Status:       "requires_payment_method",
		Metadata:     params.Metadata,
		CreatedAt:    time.Now(),
	}

	m.PaymentIntents[pi.ID] = pi
	return pi, nil
}

// GetPaymentIntent retrieves a mock payment intent.
func (m *MockProvider) GetPaymentIntent(ctx context.Context, params GetPaymentIntentParams) (*PaymentIntent, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("GetPaymentIntent(%s)", params.PaymentIntentID))

	if m.GetPaymentIntentFunc != nil {
		return m.GetPaymentIntentFunc(ctx, params)
	}

	// Default mock behavior: return stored payment intent
	pi, exists := m.PaymentIntents[params.PaymentIntentID]
	if !exists {
		return nil, ErrPaymentIntentNotFound
	}

	return pi, nil
}

// UpdatePaymentIntent updates a mock payment intent.
func (m *MockProvider) UpdatePaymentIntent(ctx context.Context, params UpdatePaymentIntentParams) (*PaymentIntent, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("UpdatePaymentIntent(%s, %d)", params.PaymentIntentID, params.AmountCents))

	pi, exists := m.PaymentIntents[params.PaymentIntentID]
	if !exists {
		return nil, ErrPaymentIntentNotFound
	}

	// Update fields
	if params.AmountCents > 0 {
		pi.AmountCents = params.AmountCents
	}
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			pi.Metadata[k] = v
		}
	}

	return pi, nil
}

// CancelPaymentIntent cancels a mock payment intent.
func (m *MockProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	m.CallLog = append(m.CallLog, fmt.Sprintf("CancelPaymentIntent(%s, %s)", paymentIntentID, tenantID))

	pi, exists := m.PaymentIntents[paymentIntentID]
	if !exists {
		return ErrPaymentIntentNotFound
	}

	// Validate tenant ownership
	if pi.Metadata == nil || pi.Metadata["tenant_id"] != tenantID {
		return ErrPaymentIntentNotFound
	}

	pi.Status = "canceled"
	return nil
}

// VerifyWebhookSignature verifies a mock webhook signature.
func (m *MockProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	m.CallLog = append(m.CallLog, "VerifyWebhookSignature")

	if m.VerifyWebhookSignatureFunc != nil {
		return m.VerifyWebhookSignatureFunc(payload, signature, secret)
	}

	// Default mock behavior: always verify successfully
	return nil
}

// CreateCustomer creates a mock customer.
func (m *MockProvider) CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("CreateCustomer(%s)", params.Email))

	if m.CreateCustomerFunc != nil {
		return m.CreateCustomerFunc(ctx, params)
	}

	// Default mock behavior: create successful customer
	customer := &Customer{
		ID:        "cus_" + uuid.New().String()[:8],
		Email:     params.Email,
		Name:      params.Name,
		CreatedAt: time.Now(),
	}

	m.Customers[customer.ID] = customer
	return customer, nil
}

// GetCustomer retrieves a mock customer.
func (m *MockProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("GetCustomer(%s)", customerID))

	customer, exists := m.Customers[customerID]
	if !exists {
		return nil, nil // Not found
	}
	return customer, nil
}

// GetCustomerByEmail searches for a mock customer by email.
func (m *MockProvider) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("GetCustomerByEmail(%s)", email))

	if m.GetCustomerByEmailFunc != nil {
		return m.GetCustomerByEmailFunc(ctx, email)
	}

	// Default mock behavior: search through customers
	for _, customer := range m.Customers {
		if customer.Email == email {
			return customer, nil
		}
	}
	return nil, nil // Not found
}

// UpdateCustomer updates a mock customer.
func (m *MockProvider) UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (*Customer, error) {
	m.CallLog = append(m.CallLog, "UpdateCustomer")
	return nil, ErrNotImplemented
}

// CreateSubscription creates a mock subscription.
func (m *MockProvider) CreateSubscription(ctx context.Context, params SubscriptionParams) (*Subscription, error) {
	m.CallLog = append(m.CallLog, "CreateSubscription")
	return nil, ErrNotImplemented
}

// CancelSubscription cancels a mock subscription.
func (m *MockProvider) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error {
	m.CallLog = append(m.CallLog, "CancelSubscription")
	return ErrNotImplemented
}

// RefundPayment refunds a mock payment.
func (m *MockProvider) RefundPayment(ctx context.Context, params RefundParams) (*Refund, error) {
	m.CallLog = append(m.CallLog, "RefundPayment")
	return nil, ErrNotImplemented
}

// SimulateSucceededPayment updates a payment intent to succeeded status.
// Used in tests to simulate successful payment confirmation.
func (m *MockProvider) SimulateSucceededPayment(paymentIntentID string) error {
	pi, exists := m.PaymentIntents[paymentIntentID]
	if !exists {
		return ErrPaymentIntentNotFound
	}

	pi.Status = "succeeded"
	return nil
}

// SimulateFailedPayment updates a payment intent to failed status.
// Used in tests to simulate payment failures.
func (m *MockProvider) SimulateFailedPayment(paymentIntentID string, errorCode string, errorMessage string) error {
	pi, exists := m.PaymentIntents[paymentIntentID]
	if !exists {
		return ErrPaymentIntentNotFound
	}

	pi.Status = "requires_payment_method"
	pi.LastPaymentError = &PaymentError{
		Code:    errorCode,
		Message: errorMessage,
	}
	return nil
}
