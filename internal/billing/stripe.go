package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/billingportal/session"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/price"
	"github.com/stripe/stripe-go/v83/subscription"
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

	// CRITICAL: Validate tenant_id is present for multi-tenant isolation
	if params.Metadata == nil || params.Metadata["tenant_id"] == "" {
		return nil, fmt.Errorf("tenant_id is required in metadata for multi-tenant isolation")
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

	// Validate shipping address when Stripe Tax is enabled
	if params.EnableStripeTax {
		if params.ShippingAddress == nil {
			return nil, fmt.Errorf("shipping address is required when EnableStripeTax is true")
		}
		if params.ShippingAddress.Country == "" {
			return nil, fmt.Errorf("shipping address country is required for tax calculation")
		}
		if params.ShippingAddress.PostalCode == "" {
			return nil, fmt.Errorf("shipping address postal code is required for tax calculation")
		}

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
// SECURITY: In multi-tenant systems, this method validates that the payment
// intent belongs to the requesting tenant via metadata.tenant_id. Always
// provide TenantID in GetPaymentIntentParams.
//
// This is called during checkout completion to verify payment succeeded
// before creating an order in the database.
//
// By default, expands "latest_charge" to include charge details (tax breakdown).
// Override by providing custom Expand values in params.
//
// Usage:
//
//	intent, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
//	    PaymentIntentID: "pi_...",
//	    TenantID: "tenant_123",
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

	// CRITICAL: Validate tenant_id is provided for multi-tenant isolation
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
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

	// CRITICAL: Verify the payment intent belongs to the requesting tenant
	if stripePI.Metadata == nil || stripePI.Metadata["tenant_id"] != params.TenantID {
		return nil, ErrPaymentIntentNotFound // Don't leak existence to other tenants
	}

	return buildPaymentIntent(stripePI), nil
}

// UpdatePaymentIntent updates a payment intent before confirmation.
//
// SECURITY: In multi-tenant systems, this method validates that the payment
// intent belongs to the requesting tenant before allowing updates.
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

	// CRITICAL: Validate tenant_id is provided for multi-tenant isolation
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// First, verify the payment intent belongs to the requesting tenant
	_, err := s.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: params.PaymentIntentID,
		TenantID:        params.TenantID,
	})
	if err != nil {
		return nil, err // Returns ErrPaymentIntentNotFound if tenant mismatch
	}

	// Verified tenant ownership, safe to proceed with update
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

	// Build the updated payment intent, preserving tenant validation
	updated := buildPaymentIntent(stripePI)

	// Defensive: Ensure tenant_id still matches after update
	if updated.Metadata["tenant_id"] != params.TenantID {
		return nil, fmt.Errorf("tenant_id mismatch after update")
	}

	return updated, nil
}

// CancelPaymentIntent cancels a payment intent that hasn't been confirmed.
//
// SECURITY: In multi-tenant systems, this method validates that the payment
// intent belongs to the requesting tenant before allowing cancellation.
//
// Used for:
//   - Abandoned checkouts (cleanup via background job)
//   - Customer explicitly cancels checkout
//   - Cart expires during checkout
//
// Returns error if payment intent is already succeeded or canceled.
func (s *StripeProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	if paymentIntentID == "" {
		return fmt.Errorf("payment intent ID is required")
	}

	// CRITICAL: Validate tenant_id is provided for multi-tenant isolation
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// Verify the payment intent belongs to the requesting tenant before canceling
	_, err := s.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: paymentIntentID,
		TenantID:        tenantID,
	})
	if err != nil {
		return err // Returns ErrPaymentIntentNotFound if tenant mismatch
	}

	// Verified tenant ownership, safe to proceed with cancellation
	cancelParams := &stripe.PaymentIntentCancelParams{}

	_, err = paymentintent.Cancel(paymentIntentID, cancelParams)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok {
			if stripeErr.Code == stripe.ErrorCodeResourceMissing {
				return ErrPaymentIntentNotFound
			}
			// Idempotent: Already canceled is success
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

	// Use ConstructEventWithOptions to allow API version mismatch
	// This is necessary because the Stripe CLI may send events with a different
	// API version than what the SDK expects. The signature verification still
	// works correctly, but we need to be careful when deserializing event data.
	_, err := webhook.ConstructEventWithOptions(payload, signature, secret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		fmt.Printf("[WEBHOOK DEBUG] ConstructEvent error: %v\n", err)
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
	if params.Email == "" {
		return nil, fmt.Errorf("email is required to create customer")
	}

	customerParams := &stripe.CustomerParams{
		Email: stripe.String(params.Email),
	}

	if params.Name != "" {
		customerParams.Name = stripe.String(params.Name)
	}
	if params.Phone != "" {
		customerParams.Phone = stripe.String(params.Phone)
	}
	if params.Description != "" {
		customerParams.Description = stripe.String(params.Description)
	}

	// Add metadata
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			customerParams.AddMetadata(k, v)
		}
	}

	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return &Customer{
		ID:        stripeCustomer.ID,
		Email:     stripeCustomer.Email,
		Name:      stripeCustomer.Name,
		CreatedAt: time.Unix(stripeCustomer.Created, 0),
	}, nil
}

// GetCustomer retrieves an existing customer.
//
// Post-MVP feature for subscription management and payment method retrieval.
func (s *StripeProvider) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}

	stripeCustomer, err := customer.Get(customerID, nil)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return nil, nil // Customer not found - not an error
		}
		return nil, wrapStripeError(err)
	}

	return &Customer{
		ID:        stripeCustomer.ID,
		Email:     stripeCustomer.Email,
		Name:      stripeCustomer.Name,
		CreatedAt: time.Unix(stripeCustomer.Created, 0),
	}, nil
}

// GetCustomerByEmail searches for an existing customer by email.
//
// Used for reconciliation - linking existing Stripe customers to local users.
// Returns nil, nil if no customer found (not an error).
// If multiple customers exist with the same email, returns the most recent one.
func (s *StripeProvider) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	params := &stripe.CustomerListParams{
		Email: stripe.String(email),
	}
	params.Limit = stripe.Int64(1) // We only need the most recent one

	iter := customer.List(params)

	// Get the first (most recent) customer with this email
	if iter.Next() {
		stripeCustomer := iter.Customer()
		return &Customer{
			ID:        stripeCustomer.ID,
			Email:     stripeCustomer.Email,
			Name:      stripeCustomer.Name,
			CreatedAt: time.Unix(stripeCustomer.Created, 0),
		}, nil
	}

	if err := iter.Err(); err != nil {
		return nil, wrapStripeError(err)
	}

	// No customer found with this email
	return nil, nil
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
// Subscription flow:
//  1. Customer selects subscription product and frequency
//  2. Creates Stripe customer (via CreateCustomer)
//  3. Collects payment method and saves for future use
//  4. Creates subscription with customer and price ID
//  5. Stripe automatically charges customer each billing period
//  6. Webhook notifies us of successful/failed charges
//  7. We fulfill orders on successful charge
//
// SECURITY: Validates tenant_id is present in metadata.
//
// Metadata should include:
//   - tenant_id (required)
//   - subscription_id (our database ID)
//   - product_sku_id
//   - billing_interval
func (s *StripeProvider) CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*Subscription, error) {
	// Validate required params
	if params.CustomerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}
	if params.PriceID == "" {
		return nil, fmt.Errorf("price ID is required")
	}
	if params.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	// CRITICAL: Validate tenant_id is present for multi-tenant isolation
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}
	if params.Metadata == nil {
		params.Metadata = make(map[string]string)
	}
	params.Metadata["tenant_id"] = params.TenantID

	// Build Stripe subscription parameters
	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(params.CustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(params.PriceID),
				Quantity: stripe.Int64(int64(params.Quantity)),
			},
		},
		Metadata: params.Metadata,
	}

	// Set default payment method if provided
	if params.DefaultPaymentMethodID != "" {
		subParams.DefaultPaymentMethod = stripe.String(params.DefaultPaymentMethodID)
	}

	// Set collection method (default: charge_automatically)
	if params.CollectionMethod != "" {
		subParams.CollectionMethod = stripe.String(params.CollectionMethod)
	}

	// Set idempotency key if provided
	if params.IdempotencyKey != "" {
		subParams.IdempotencyKey = stripe.String(params.IdempotencyKey)
	}

	// Create subscription in Stripe
	stripeSubscription, err := subscription.New(subParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return buildSubscription(stripeSubscription), nil
}

// CreateRecurringPrice creates a Stripe Price for recurring subscriptions.
//
// Each subscription item needs its own recurring price in Stripe.
// Prices are linked to existing Stripe Products (synced from product catalog).
//
// SECURITY: Validates tenant_id is present in metadata.
//
// Returns Price with Stripe price ID (price_...) for use in CreateSubscription.
func (s *StripeProvider) CreateRecurringPrice(ctx context.Context, params CreateRecurringPriceParams) (*Price, error) {
	// Validate required params
	if params.Currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if params.UnitAmountCents <= 0 {
		return nil, fmt.Errorf("unit amount must be greater than 0")
	}
	if params.BillingInterval == "" {
		return nil, fmt.Errorf("billing interval is required")
	}
	if params.ProductID == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// CRITICAL: Validate tenant_id is present for multi-tenant isolation
	if params.Metadata == nil || params.Metadata["tenant_id"] == "" {
		return nil, fmt.Errorf("tenant_id is required in metadata for multi-tenant isolation")
	}

	// Build Stripe price parameters
	priceParams := &stripe.PriceParams{
		Currency:   stripe.String(strings.ToLower(params.Currency)),
		UnitAmount: stripe.Int64(int64(params.UnitAmountCents)),
		Product:    stripe.String(params.ProductID),
		Recurring: &stripe.PriceRecurringParams{
			Interval:      stripe.String(params.BillingInterval),
			IntervalCount: stripe.Int64(int64(params.IntervalCount)),
		},
	}

	if params.Nickname != "" {
		priceParams.Nickname = stripe.String(params.Nickname)
	}

	if params.Metadata != nil {
		priceParams.Metadata = params.Metadata
	}

	// Create price in Stripe
	stripePrice, err := price.New(priceParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return buildPrice(stripePrice), nil
}

// GetSubscription retrieves an existing subscription.
//
// SECURITY: Validates tenant_id in subscription metadata before returning.
// Returns ErrSubscriptionNotFound if subscription doesn't exist or tenant mismatch.
func (s *StripeProvider) GetSubscription(ctx context.Context, params GetSubscriptionParams) (*Subscription, error) {
	// Validate required params
	if params.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// Build Stripe subscription parameters
	subParams := &stripe.SubscriptionParams{}

	// Add expand parameters if specified
	if len(params.Expand) > 0 {
		for _, expand := range params.Expand {
			subParams.AddExpand(expand)
		}
	}

	// Get subscription from Stripe
	stripeSubscription, err := subscription.Get(params.SubscriptionID, subParams)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return nil, ErrSubscriptionNotFound
		}
		return nil, wrapStripeError(err)
	}

	// CRITICAL: Verify the subscription belongs to the requesting tenant
	if stripeSubscription.Metadata == nil || stripeSubscription.Metadata["tenant_id"] != params.TenantID {
		return nil, ErrSubscriptionNotFound // Don't leak existence to other tenants
	}

	return buildSubscription(stripeSubscription), nil
}

// PauseSubscription pauses a subscription until explicitly resumed.
//
// Paused subscriptions:
//   - Stop creating invoices
//   - Retain all settings (payment method, pricing, items)
//   - Can be resumed at any time
//
// SECURITY: Validates tenant_id ownership before pausing.
func (s *StripeProvider) PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*Subscription, error) {
	// Validate required params
	if params.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// Verify tenant ownership
	_, err := s.GetSubscription(ctx, GetSubscriptionParams{
		SubscriptionID: params.SubscriptionID,
		TenantID:       params.TenantID,
	})
	if err != nil {
		return nil, err // Returns ErrSubscriptionNotFound if tenant mismatch
	}

	// Build Stripe subscription update parameters
	subParams := &stripe.SubscriptionParams{
		PauseCollection: &stripe.SubscriptionPauseCollectionParams{
			Behavior: stripe.String(params.Behavior),
		},
	}

	// Set ResumesAt if provided
	if params.ResumesAt != nil {
		subParams.PauseCollection.ResumesAt = stripe.Int64(params.ResumesAt.Unix())
	}

	// Update subscription in Stripe
	stripeSubscription, err := subscription.Update(params.SubscriptionID, subParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return buildSubscription(stripeSubscription), nil
}

// ResumeSubscription resumes a paused subscription immediately.
//
// Resumed subscriptions:
//   - Immediately create invoice for current period
//   - Resume regular billing cycle
//
// SECURITY: Validates tenant_id ownership before resuming.
func (s *StripeProvider) ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*Subscription, error) {
	// Validate required params
	if params.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// Verify tenant ownership
	_, err := s.GetSubscription(ctx, GetSubscriptionParams{
		SubscriptionID: params.SubscriptionID,
		TenantID:       params.TenantID,
	})
	if err != nil {
		return nil, err // Returns ErrSubscriptionNotFound if tenant mismatch
	}

	// Build Stripe subscription update parameters
	// Setting PauseCollection to empty string resumes the subscription
	subParams := &stripe.SubscriptionParams{
		PauseCollection: &stripe.SubscriptionPauseCollectionParams{},
	}

	// Update subscription in Stripe
	stripeSubscription, err := subscription.Update(params.SubscriptionID, subParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return buildSubscription(stripeSubscription), nil
}

// CancelSubscription cancels a subscription.
//
// If CancelAtPeriodEnd is true:
//   - Subscription remains active until end of current billing period
//   - Customer can still access benefits until period ends
//
// If CancelAtPeriodEnd is false:
//   - Subscription canceled immediately
//   - Customer loses access immediately
//
// SECURITY: Validates tenant_id ownership before canceling.
func (s *StripeProvider) CancelSubscription(ctx context.Context, params CancelSubscriptionParams) error {
	// Validate required params
	if params.SubscriptionID == "" {
		return fmt.Errorf("subscription ID is required")
	}
	if params.TenantID == "" {
		return fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}

	// Verify tenant ownership
	_, err := s.GetSubscription(ctx, GetSubscriptionParams{
		SubscriptionID: params.SubscriptionID,
		TenantID:       params.TenantID,
	})
	if err != nil {
		return err // Returns ErrSubscriptionNotFound if tenant mismatch
	}

	if params.CancelAtPeriodEnd {
		// Schedule cancellation at end of period
		subParams := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}

		// Add cancellation reason to metadata if provided
		if params.CancellationReason != "" {
			subParams.AddMetadata("cancellation_reason", params.CancellationReason)
		}

		_, err = subscription.Update(params.SubscriptionID, subParams)
		if err != nil {
			return wrapStripeError(err)
		}
	} else {
		// Cancel immediately
		cancelParams := &stripe.SubscriptionCancelParams{}

		_, err = subscription.Cancel(params.SubscriptionID, cancelParams)
		if err != nil {
			stripeErr, ok := err.(*stripe.Error)
			if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
				return ErrSubscriptionNotFound
			}
			return wrapStripeError(err)
		}
	}

	return nil
}

// CreateCustomerPortalSession creates a Stripe Customer Portal session.
//
// Returns session URL where customer can:
//   - View subscription details
//   - Update payment method
//   - Pause/resume subscription
//   - Cancel subscription
//   - View invoice history
//
// Session expires after 60 minutes.
//
// SECURITY: Validates customer belongs to tenant before creating session.
func (s *StripeProvider) CreateCustomerPortalSession(ctx context.Context, params CreatePortalSessionParams) (*PortalSession, error) {
	// Validate required params
	if params.CustomerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for multi-tenant isolation")
	}
	if params.ReturnURL == "" {
		return nil, fmt.Errorf("return URL is required")
	}

	// Get customer and verify tenant ownership
	stripeCustomer, err := customer.Get(params.CustomerID, nil)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, wrapStripeError(err)
	}

	// Verify customer belongs to tenant
	if stripeCustomer.Metadata == nil || stripeCustomer.Metadata["tenant_id"] != params.TenantID {
		return nil, fmt.Errorf("customer does not belong to tenant")
	}

	// Build portal session parameters
	sessionParams := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(params.CustomerID),
		ReturnURL: stripe.String(params.ReturnURL),
	}

	// Create portal session
	portalSession, err := session.New(sessionParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	// Map to PortalSession
	return &PortalSession{
		ID:        portalSession.ID,
		URL:       portalSession.URL,
		CreatedAt: time.Unix(portalSession.Created, 0),
		ExpiresAt: time.Unix(portalSession.Created+3600, 0), // Sessions expire after 1 hour
	}, nil
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
		ReceiptEmail: stripePI.ReceiptEmail,
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

// buildPrice maps Stripe Price to our Price type.
// Centralizes mapping logic used by CreateRecurringPrice method.
func buildPrice(stripePrice *stripe.Price) *Price {
	if stripePrice == nil {
		return nil
	}

	price := &Price{
		ID:              stripePrice.ID,
		ProductID:       stripePrice.Product.ID,
		Currency:        string(stripePrice.Currency),
		UnitAmountCents: int32(stripePrice.UnitAmount),
		Type:            string(stripePrice.Type),
		Active:          stripePrice.Active,
		Metadata:        stripePrice.Metadata,
		CreatedAt:       time.Unix(stripePrice.Created, 0),
	}

	if stripePrice.Recurring != nil {
		price.Recurring = &PriceRecurring{
			Interval:      string(stripePrice.Recurring.Interval),
			IntervalCount: int32(stripePrice.Recurring.IntervalCount),
		}
	}

	return price
}

// buildSubscription maps Stripe Subscription to our Subscription type.
// Centralizes mapping logic used by subscription methods.
func buildSubscription(stripeSub *stripe.Subscription) *Subscription {
	if stripeSub == nil {
		return nil
	}

	subscription := &Subscription{
		ID:                     stripeSub.ID,
		CustomerID:             stripeSub.Customer.ID,
		Status:                 string(stripeSub.Status),
		DefaultPaymentMethodID: "",
		CurrentPeriodStart:     time.Unix(stripeSub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:       time.Unix(stripeSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:      stripeSub.CancelAtPeriodEnd,
		Metadata:               stripeSub.Metadata,
		CreatedAt:              time.Unix(stripeSub.Created, 0),
	}

	// Set default payment method if present
	if stripeSub.DefaultPaymentMethod != nil {
		subscription.DefaultPaymentMethodID = stripeSub.DefaultPaymentMethod.ID
	}

	// Set canceled at timestamp if present
	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		subscription.CanceledAt = &canceledAt
	}

	// Map subscription items
	subscription.Items = make([]SubscriptionItem, len(stripeSub.Items.Data))
	for i, item := range stripeSub.Items.Data {
		subscription.Items[i] = SubscriptionItem{
			ID:       item.ID,
			PriceID:  item.Price.ID,
			Quantity: int32(item.Quantity),
			Metadata: item.Metadata,
		}
	}

	// Map pause collection if present
	if stripeSub.PauseCollection != nil {
		subscription.PauseCollection = &SubscriptionPauseCollection{
			Behavior: string(stripeSub.PauseCollection.Behavior),
		}
		if stripeSub.PauseCollection.ResumesAt > 0 {
			resumesAt := time.Unix(stripeSub.PauseCollection.ResumesAt, 0)
			subscription.PauseCollection.ResumesAt = &resumesAt
		}
	}

	return subscription
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
