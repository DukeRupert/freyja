package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/webhook"
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
	if err := validateAmount(params.AmountCents, params.Currency); err != nil {
		return nil, err
	}

	piParams := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(params.AmountCents)),
		Currency: stripe.String(strings.ToLower(params.Currency)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	if params.Description != "" {
		piParams.Description = stripe.String(params.Description)
	}

	if params.CustomerEmail != "" {
		piParams.ReceiptEmail = stripe.String(params.CustomerEmail)
	}

	if params.CustomerID != "" {
		piParams.Customer = stripe.String(params.CustomerID)
	}

	if params.SetupFutureUsage != "" {
		piParams.SetupFutureUsage = stripe.String(params.SetupFutureUsage)
	}

	if params.CaptureMethod != "" {
		piParams.CaptureMethod = stripe.String(params.CaptureMethod)
	}

	if params.Metadata != nil {
		piParams.Metadata = params.Metadata
	}

	if params.EnableStripeTax && params.ShippingAddress != nil {
		piParams.Shipping = &stripe.ShippingDetailsParams{
			Address: &stripe.AddressParams{
				Line1:      stripe.String(params.ShippingAddress.Line1),
				Line2:      stripe.String(params.ShippingAddress.Line2),
				City:       stripe.String(params.ShippingAddress.City),
				State:      stripe.String(params.ShippingAddress.State),
				PostalCode: stripe.String(params.ShippingAddress.PostalCode),
				Country:    stripe.String(params.ShippingAddress.Country),
			},
		}
	}

	if params.IdempotencyKey != "" {
		piParams.IdempotencyKey = stripe.String(params.IdempotencyKey)
	}

	stripePI, err := paymentintent.New(piParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return buildPaymentIntent(stripePI), nil
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
	if params.PaymentIntentID == "" {
		return nil, fmt.Errorf("payment intent ID is required")
	}

	piParams := &stripe.PaymentIntentParams{}

	if len(params.Expand) > 0 {
		for _, expand := range params.Expand {
			piParams.AddExpand(expand)
		}
	} else {
		piParams.AddExpand("latest_charge")
	}

	stripePI, err := paymentintent.Get(params.PaymentIntentID, piParams)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return nil, ErrPaymentIntentNotFound
		}
		return nil, wrapStripeError(err)
	}

	return buildPaymentIntent(stripePI), nil
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
	if params.PaymentIntentID == "" {
		return nil, fmt.Errorf("payment intent ID is required")
	}

	piParams := &stripe.PaymentIntentParams{}

	if params.AmountCents > 0 {
		piParams.Amount = stripe.Int64(int64(params.AmountCents))
	}

	if params.Description != "" {
		piParams.Description = stripe.String(params.Description)
	}

	if params.Metadata != nil {
		for k, v := range params.Metadata {
			piParams.AddMetadata(k, v)
		}
	}

	stripePI, err := paymentintent.Update(params.PaymentIntentID, piParams)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return nil, ErrPaymentIntentNotFound
		}
		return nil, wrapStripeError(err)
	}

	return buildPaymentIntent(stripePI), nil
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
	if paymentIntentID == "" {
		return fmt.Errorf("payment intent ID is required")
	}

	cancelParams := &stripe.PaymentIntentCancelParams{}

	_, err := paymentintent.Cancel(paymentIntentID, cancelParams)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok {
			if stripeErr.Code == stripe.ErrorCodeResourceMissing {
				return ErrPaymentIntentNotFound
			}
			if stripeErr.Code == "payment_intent_unexpected_state" && strings.Contains(stripeErr.Msg, "canceled") {
				return nil
			}
		}
		return wrapStripeError(err)
	}

	return nil
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
	if len(payload) == 0 {
		return ErrInvalidWebhookSignature
	}

	if signature == "" {
		return ErrInvalidWebhookSignature
	}

	if secret == "" {
		return ErrInvalidWebhookSignature
	}

	_, err := webhook.ConstructEvent(payload, signature, secret)
	if err != nil {
		return ErrInvalidWebhookSignature
	}

	return nil
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
	if stripePI == nil {
		return nil
	}

	pi := &PaymentIntent{
		ID:           stripePI.ID,
		ClientSecret: stripePI.ClientSecret,
		AmountCents:  int32(stripePI.Amount),
		Currency:     string(stripePI.Currency),
		Status:       string(stripePI.Status),
		Metadata:     stripePI.Metadata,
		CreatedAt:    time.Unix(stripePI.Created, 0),
	}

	// Extract tax amount from latest charge if available
	if stripePI.LatestCharge != nil {
		charge := stripePI.LatestCharge
		if charge.Metadata != nil {
			if taxStr, ok := charge.Metadata["tax_amount"]; ok {
				var taxAmount int64
				fmt.Sscanf(taxStr, "%d", &taxAmount)
				pi.TaxCents = int32(taxAmount)
			}
		}
	}

	// Extract shipping amount from metadata if available
	if stripePI.Metadata != nil {
		if shippingStr, ok := stripePI.Metadata["shipping_amount"]; ok {
			var shippingAmount int64
			fmt.Sscanf(shippingStr, "%d", &shippingAmount)
			pi.ShippingCents = int32(shippingAmount)
		}
	}

	// Map last payment error if present
	if stripePI.LastPaymentError != nil {
		pi.LastPaymentError = &PaymentError{
			Code:        string(stripePI.LastPaymentError.Code),
			Message:     stripePI.LastPaymentError.Msg,
			DeclineCode: string(stripePI.LastPaymentError.DeclineCode),
		}
	}

	return pi
}

// wrapStripeError converts a Stripe SDK error to our StripeError type.
// Provides consistent error handling across all methods.
func wrapStripeError(err error) error {
	if err == nil {
		return nil
	}

	stripeErr, ok := err.(*stripe.Error)
	if !ok {
		return err
	}

	return &StripeError{
		Message:       stripeErr.Msg,
		Code:          string(stripeErr.Code),
		DeclineCode:   string(stripeErr.DeclineCode),
		StripeCode:    fmt.Sprintf("%d", stripeErr.HTTPStatusCode),
		RequestID:     stripeErr.RequestID,
		OriginalError: err,
	}
}

// validateAmount checks if amount meets Stripe's minimum requirements.
func validateAmount(amountCents int32, currency string) error {
	currencyLower := strings.ToLower(currency)

	var minAmount int32
	switch currencyLower {
	case "usd", "eur":
		minAmount = 50
	case "gbp":
		minAmount = 30
	default:
		minAmount = 50
	}

	if amountCents < minAmount {
		return ErrAmountTooSmall
	}

	return nil
}
