package billing

import (
	"context"
	"time"
)

// Provider defines the interface for payment processing.
// Implementations can use Stripe, PayPal, Square, etc.
//
// For MVP, only Stripe implementation is required.
type Provider interface {
	// CreateCustomer creates a customer record in the billing provider.
	CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, error)

	// CreatePaymentIntent creates a payment intent for one-time charges.
	CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*PaymentIntent, error)

	// CreateSubscription creates a recurring subscription.
	CreateSubscription(ctx context.Context, params SubscriptionParams) (*Subscription, error)

	// CancelSubscription cancels a subscription.
	CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error

	// RefundPayment refunds a completed payment.
	RefundPayment(ctx context.Context, paymentID string, amount int64) (*Refund, error)

	// VerifyWebhookSignature verifies that a webhook request is authentic.
	VerifyWebhookSignature(payload []byte, signature string, secret string) error
}

// CreateCustomerParams contains parameters for creating a customer.
type CreateCustomerParams struct {
	Email       string
	Name        string
	Phone       string
	Description string
	Metadata    map[string]string
}

// Customer represents a billing customer.
type Customer struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// PaymentIntentParams contains parameters for creating a payment intent.
type PaymentIntentParams struct {
	Amount      int64  // Amount in cents
	Currency    string // e.g., "usd"
	CustomerID  string
	Description string
	Metadata    map[string]string
}

// PaymentIntent represents a payment intent.
type PaymentIntent struct {
	ID           string
	Amount       int64
	Currency     string
	Status       string // requires_payment_method, requires_confirmation, succeeded, etc.
	ClientSecret string // For frontend integration
	CreatedAt    time.Time
}

// SubscriptionParams contains parameters for creating a subscription.
type SubscriptionParams struct {
	CustomerID string
	PriceID    string // Provider's price/plan identifier
	Quantity   int
	Metadata   map[string]string
}

// Subscription represents a recurring subscription.
type Subscription struct {
	ID                string
	CustomerID        string
	Status            string // active, past_due, canceled, etc.
	CurrentPeriodEnd  time.Time
	CancelAtPeriodEnd bool
	CreatedAt         time.Time
}

// Refund represents a payment refund.
type Refund struct {
	ID        string
	PaymentID string
	Amount    int64
	Status    string // succeeded, pending, failed
	CreatedAt time.Time
}
