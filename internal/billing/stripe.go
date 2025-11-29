package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v83"
	_ "github.com/stripe/stripe-go/v83/paymentintent"
	_ "github.com/stripe/stripe-go/v83/webhook"
)

// StripeProvider implements Provider using Stripe.
type StripeProvider struct {
	config StripeConfig
}

// NewStripeProvider creates a new Stripe billing provider.
//
// The apiKey should be a Stripe secret key:
//   - Test mode: sk_test_...
//   - Live mode: sk_live_...
//
// The webhookSecret is used to verify webhook signatures:
//   - Webhook signing secret: whsec_...
//
// This constructor sets the global Stripe API key. In a multi-tenant system,
// each tenant should have their own StripeProvider instance with their own keys.
func NewStripeProvider(config StripeConfig) (*StripeProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid stripe configuration: %w", err)
	}

	// Set global Stripe key
	// Note: For multi-tenant, consider using per-request keys via stripe.Key
	stripe.Key = config.APIKey

	// TODO: Configure SDK defaults when implementing
	// Note: MaxNetworkRetries configuration may need to be set differently
	// depending on Stripe SDK version
	_ = config.MaxRetries

	return &StripeProvider{
		config: config,
	}, nil
}

// CreatePaymentIntent creates a Stripe payment intent.
//
// This is the primary method for checkout flow. It creates a payment intent
// with the specified amount and returns a client_secret for frontend confirmation.
//
// Flow:
//  1. Checkout service calculates order total (items + shipping + tax)
//  2. Calls CreatePaymentIntent with total amount and metadata
//  3. Returns PaymentIntent.ClientSecret to frontend
//  4. Frontend uses Stripe.js to collect payment and confirm
//  5. Stripe calls webhook on confirmation
//  6. Frontend calls /checkout/complete with payment_intent_id
//  7. Backend calls GetPaymentIntent to verify status before creating order
//
// Idempotency:
//   - Uses params.IdempotencyKey to prevent duplicate payment intents
//   - Typically use cart_id as idempotency key
//   - If called twice with same key and matching params, returns existing intent
//
// Metadata:
//   - Always include: tenant_id, cart_id, order_type ("retail" or "wholesale")
//   - Metadata is searchable in Stripe dashboard and included in webhooks
//
// Tax calculation:
//   - If params.EnableStripeTax is true, Stripe calculates tax based on:
//   - Customer shipping address
//   - Line items with tax codes
//   - Tax amount included in PaymentIntent.AmountCents
//   - Tax breakdown available in PaymentIntent.TaxCents
func (s *StripeProvider) CreatePaymentIntent(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
	// TODO: Implementation
	//
	// Steps:
	// 1. Validate params (amount >= 50 cents, currency supported, required metadata)
	// 2. Build stripe.PaymentIntentParams:
	//    - Amount: params.AmountCents
	//    - Currency: params.Currency
	//    - AutomaticPaymentMethods.Enabled: true (supports card, ideal, etc.)
	//    - Metadata: params.Metadata (ensure tenant_id, cart_id present)
	//    - Description: params.Description
	//    - ReceiptEmail: params.CustomerEmail
	//    - SetupFutureUsage: params.SetupFutureUsage (if saving payment method)
	// 3. If params.CustomerID set:
	//    - Customer: params.CustomerID
	// 4. If params.EnableStripeTax:
	//    - Configure automatic tax calculation
	//    - Include params.ShippingAddress
	//    - Include params.LineItems with tax codes
	// 5. Set idempotency key: params.IdempotencyKey
	// 6. Call paymentintent.New()
	// 7. Handle errors (wrap in StripeError)
	// 8. Map Stripe response to PaymentIntent
	// 9. Return PaymentIntent with ClientSecret
	//
	// Error handling:
	// - Invalid amount -> ErrAmountTooSmall
	// - Invalid API key -> ErrInvalidAPIKey
	// - Idempotency conflict -> ErrIdempotencyConflict
	// - Stripe API errors -> StripeError
	panic("not implemented")
}

// GetPaymentIntent retrieves an existing payment intent.
//
// This is called during checkout completion to verify payment succeeded
// before creating an order in the database.
//
// Usage:
//
//	intent, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
//	    PaymentIntentID: "pi_...",
//	})
//	if err != nil {
//	    return err
//	}
//	if intent.Status != "succeeded" {
//	    return ErrPaymentFailed
//	}
//	// Safe to create order
//
// The returned PaymentIntent includes:
//   - Status: requires_payment_method, requires_confirmation, succeeded, canceled
//   - AmountCents: Total amount including any Stripe Tax
//   - TaxCents: Tax amount (if Stripe Tax enabled)
//   - Metadata: Includes tenant_id, cart_id from creation
//   - LastPaymentError: Details if payment failed
func (s *StripeProvider) GetPaymentIntent(ctx context.Context, params GetPaymentIntentParams) (*PaymentIntent, error) {
	// TODO: Implementation
	//
	// Steps:
	// 1. Validate params.PaymentIntentID is not empty
	// 2. Build stripe.PaymentIntentParams for Get:
	//    - Expand: params.Expand (e.g., ["latest_charge"])
	// 3. Call paymentintent.Get(params.PaymentIntentID, getParams)
	// 4. Handle errors:
	//    - Not found -> ErrPaymentIntentNotFound
	//    - Invalid API key -> ErrInvalidAPIKey
	//    - Other errors -> StripeError
	// 5. Map Stripe response to PaymentIntent:
	//    - Extract tax amount from charges.data[0].amount_tax (if Stripe Tax)
	//    - Extract shipping from metadata or charges
	//    - Map status
	//    - Extract last payment error if present
	// 6. Return PaymentIntent
	panic("not implemented")
}

// UpdatePaymentIntent updates a payment intent before confirmation.
//
// Used when customer modifies cart during checkout (changes quantity, adds item).
// Can only update payment intents that haven't been confirmed yet.
//
// Common updates:
//   - AmountCents: New total after cart changes
//   - Metadata: Additional tracking information
//   - Description: Updated order description
//
// Returns the updated PaymentIntent.
func (s *StripeProvider) UpdatePaymentIntent(ctx context.Context, params UpdatePaymentIntentParams) (*PaymentIntent, error) {
	// TODO: Implementation
	//
	// Steps:
	// 1. Validate params.PaymentIntentID
	// 2. Build stripe.PaymentIntentParams for Update:
	//    - Amount: params.AmountCents (if changed)
	//    - Metadata: params.Metadata (merge with existing)
	//    - Description: params.Description (if changed)
	// 3. Call paymentintent.Update(params.PaymentIntentID, updateParams)
	// 4. Handle errors:
	//    - Already confirmed -> return error
	//    - Not found -> ErrPaymentIntentNotFound
	//    - Other errors -> StripeError
	// 5. Map response to PaymentIntent
	// 6. Return updated PaymentIntent
	panic("not implemented")
}

// CancelPaymentIntent cancels a payment intent that hasn't been confirmed.
//
// Used for:
//   - Abandoned checkouts (cleanup via background job)
//   - Customer explicitly cancels checkout
//   - Cart expires during checkout
//
// Returns error if payment intent is already succeeded or canceled.
func (s *StripeProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	// TODO: Implementation
	//
	// Steps:
	// 1. Validate paymentIntentID not empty
	// 2. Build stripe.PaymentIntentCancelParams
	// 3. Call paymentintent.Cancel(paymentIntentID, cancelParams)
	// 4. Handle errors:
	//    - Already succeeded -> return error
	//    - Already canceled -> return nil (idempotent)
	//    - Not found -> ErrPaymentIntentNotFound
	//    - Other errors -> StripeError
	// 5. Return nil on success
	panic("not implemented")
}

// VerifyWebhookSignature verifies that a webhook request is authentic.
//
// Called at the start of webhook handlers to ensure request came from Stripe.
// Protects against:
//   - Replay attacks
//   - Forged webhook events
//   - MITM attacks
//
// Usage in webhook handler:
//
//	payload, _ := ioutil.ReadAll(r.Body)
//	sig := r.Header.Get("Stripe-Signature")
//	if err := provider.VerifyWebhookSignature(payload, sig, webhookSecret); err != nil {
//	    http.Error(w, "Invalid signature", http.StatusUnauthorized)
//	    return
//	}
//	// Safe to process webhook
//
// The secret parameter should be the webhook signing secret (whsec_...)
// from Stripe dashboard, not the API key.
func (s *StripeProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	// TODO: Implementation
	//
	// Steps:
	// 1. Validate inputs (payload, signature, secret not empty)
	// 2. Call webhook.ConstructEvent(payload, signature, secret)
	// 3. If error (invalid signature, timestamp too old):
	//    - Return ErrInvalidWebhookSignature
	// 4. Return nil if valid
	//
	// Note: ConstructEvent also parses the event, but we only care about
	// signature verification here. Webhook handler will parse event separately.
	panic("not implemented")
}

// CreateCustomer creates a Stripe customer.
//
// Post-MVP feature for:
//   - Saving payment methods for future purchases
//   - Creating subscriptions
//   - Viewing customer payment history in Stripe dashboard
//
// In multi-tenant system:
//   - Include tenant_id in metadata
//   - Use tenant's Stripe account (not platform account)
//
// Returns Customer with Stripe ID for future reference.
func (s *StripeProvider) CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, error) {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate params.Email (required)
	// 2. Build stripe.CustomerParams:
	//    - Email: params.Email
	//    - Name: params.Name
	//    - Phone: params.Phone
	//    - Description: params.Description
	//    - Metadata: params.Metadata (ensure tenant_id)
	// 3. Call customer.New()
	// 4. Map response to Customer
	// 5. Return Customer
	return nil, ErrNotImplemented
}

// GetCustomer retrieves an existing customer.
//
// Post-MVP feature for subscription management and payment method retrieval.
func (s *StripeProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate customerID not empty
	// 2. Call customer.Get(customerID, nil)
	// 3. Handle errors (not found, etc.)
	// 4. Map response to Customer
	// 5. Return Customer
	return nil, ErrNotImplemented
}

// UpdateCustomer updates customer information.
//
// Post-MVP feature for account management.
func (s *StripeProvider) UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (*Customer, error) {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate customerID not empty
	// 2. Build stripe.CustomerParams for update
	// 3. Call customer.Update(customerID, updateParams)
	// 4. Map response to Customer
	// 5. Return updated Customer
	return nil, ErrNotImplemented
}

// CreateSubscription creates a recurring subscription.
//
// Post-MVP feature for coffee subscriptions.
//
// Subscription flow:
//  1. Customer selects subscription product and frequency
//  2. Creates Stripe customer (via CreateCustomer)
//  3. Collects payment method and saves for future use
//  4. Creates subscription with customer and price ID
//  5. Stripe automatically charges customer each billing period
//  6. Webhook notifies us of successful/failed charges
//  7. We fulfill orders on successful charge
//
// Metadata should include:
//   - tenant_id
//   - product_id
//   - frequency (weekly, biweekly, monthly, etc.)
func (s *StripeProvider) CreateSubscription(ctx context.Context, params SubscriptionParams) (*Subscription, error) {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate params.CustomerID and params.PriceID
	// 2. Build stripe.SubscriptionParams:
	//    - Customer: params.CustomerID
	//    - Items: []{Price: params.PriceID, Quantity: params.Quantity}
	//    - Metadata: params.Metadata (ensure tenant_id)
	//    - PaymentBehavior: "default_incomplete" (requires payment method)
	//    - PaymentSettings: {SaveDefaultPaymentMethod: "on_subscription"}
	// 3. Call subscription.New()
	// 4. Map response to Subscription
	// 5. Return Subscription
	return nil, ErrNotImplemented
}

// CancelSubscription cancels a subscription.
//
// Post-MVP feature for subscription management.
//
// If cancelAtPeriodEnd is true:
//   - Subscription remains active until end of current billing period
//   - Customer can still access benefits until period ends
//   - Useful for allowing customer to finish out paid period
//
// If cancelAtPeriodEnd is false:
//   - Subscription canceled immediately
//   - Customer loses access immediately
//   - May issue prorated refund (configurable in Stripe dashboard)
func (s *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate subscriptionID not empty
	// 2. If cancelAtPeriodEnd:
	//    - Build stripe.SubscriptionParams with CancelAtPeriodEnd: true
	//    - Call subscription.Update()
	// 3. Else:
	//    - Build stripe.SubscriptionCancelParams
	//    - Call subscription.Cancel()
	// 4. Handle errors
	// 5. Return nil on success
	return ErrNotImplemented
}

// RefundPayment refunds a completed payment.
//
// Post-MVP feature for order cancellations and returns.
//
// Refund types:
//   - Full refund: params.AmountCents = 0
//   - Partial refund: params.AmountCents = specific amount
//
// Refund reasons (for reporting):
//   - "duplicate": Accidental duplicate charge
//   - "fraudulent": Fraudulent charge
//   - "requested_by_customer": Customer requested refund
//
// Note: Stripe fees are not refunded for partial refunds.
func (s *StripeProvider) RefundPayment(ctx context.Context, params RefundParams) (*Refund, error) {
	// TODO: Post-MVP implementation
	//
	// Steps:
	// 1. Validate params.PaymentIntentID
	// 2. Build stripe.RefundParams:
	//    - PaymentIntent: params.PaymentIntentID
	//    - Amount: params.AmountCents (omit for full refund)
	//    - Reason: params.Reason
	//    - Metadata: params.Metadata
	// 3. Call refund.New()
	// 4. Map response to Refund
	// 5. Return Refund
	return nil, ErrNotImplemented
}

// Helper functions (not exported)

// buildPaymentIntent maps Stripe PaymentIntent to our PaymentIntent type.
// Centralizes mapping logic used by Create, Get, and Update methods.
func buildPaymentIntent(stripePI *stripe.PaymentIntent) *PaymentIntent {
	// TODO: Implementation
	//
	// Map fields:
	// - ID
	// - ClientSecret
	// - Amount (as AmountCents int32)
	// - Currency
	// - Status
	// - Metadata
	// - Created (as CreatedAt time.Time)
	//
	// Extract tax and shipping:
	// - If charges exist, extract from latest_charge
	// - Check metadata for shipping amount
	//
	// Map last payment error if present
	panic("not implemented")
}

// wrapStripeError converts a Stripe SDK error to our StripeError type.
// Provides consistent error handling across all methods.
func wrapStripeError(err error) error {
	// TODO: Implementation
	//
	// Steps:
	// 1. Type assert to *stripe.Error
	// 2. Extract fields:
	//    - Msg -> Message
	//    - Code -> Code
	//    - DeclineCode -> DeclineCode
	//    - HTTPStatusCode -> StripeCode
	//    - RequestID -> RequestID
	// 3. Return &StripeError{...}
	panic("not implemented")
}

// validateAmount checks if amount meets Stripe's minimum requirements.
func validateAmount(amountCents int32, currency string) error {
	// TODO: Implementation
	//
	// Stripe minimum amounts by currency:
	// - USD: 50 cents ($0.50)
	// - EUR: 50 cents (€0.50)
	// - GBP: 30 pence (£0.30)
	//
	// Return ErrAmountTooSmall if below minimum
	panic("not implemented")
}
