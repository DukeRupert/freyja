package billing

import (
	"context"
)

// StripeProvider implements Provider using Stripe.
// This is a placeholder for future implementation.
type StripeProvider struct {
	apiKey        string
	webhookSecret string
	// stripe *stripe.Client // Stripe SDK client (not implemented yet)
}

// NewStripeProvider creates a new Stripe billing provider.
// This is a stub - full implementation will use Stripe Go SDK.
func NewStripeProvider(apiKey, webhookSecret string) *StripeProvider {
	return &StripeProvider{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
	}
}

// CreateCustomer creates a Stripe customer.
func (s *StripeProvider) CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, error) {
	// TODO: Implement using stripe.Customer.New()
	panic("StripeProvider not implemented yet")
}

// CreatePaymentIntent creates a Stripe payment intent.
func (s *StripeProvider) CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*PaymentIntent, error) {
	// TODO: Implement using stripe.PaymentIntent.New()
	panic("StripeProvider not implemented yet")
}

// CreateSubscription creates a Stripe subscription.
func (s *StripeProvider) CreateSubscription(ctx context.Context, params SubscriptionParams) (*Subscription, error) {
	// TODO: Implement using stripe.Subscription.New()
	panic("StripeProvider not implemented yet")
}

// CancelSubscription cancels a Stripe subscription.
func (s *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error {
	// TODO: Implement using stripe.Subscription.Cancel()
	panic("StripeProvider not implemented yet")
}

// RefundPayment refunds a Stripe payment.
func (s *StripeProvider) RefundPayment(ctx context.Context, paymentID string, amount int64) (*Refund, error) {
	// TODO: Implement using stripe.Refund.New()
	panic("StripeProvider not implemented yet")
}

// VerifyWebhookSignature verifies a Stripe webhook signature.
func (s *StripeProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	// TODO: Implement using stripe.webhook.ConstructEvent()
	panic("StripeProvider not implemented yet")
}
