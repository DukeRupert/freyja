# Freyja Billing Package

Production-ready Stripe payment integration for the Freyja e-commerce platform.

## Overview

This package provides a complete Stripe payment implementation with:
- ✅ Multi-tenant isolation (critical security feature)
- ✅ Payment intent lifecycle management
- ✅ Idempotency key support
- ✅ Webhook signature verification
- ✅ Comprehensive error handling
- ✅ Mock provider for testing
- ✅ Integration tests with real Stripe API

## Files

### Core Implementation
- **`billing.go`** - Provider interface and domain types
- **`stripe.go`** - Stripe SDK implementation
- **`stripe_config.go`** - Configuration with validation
- **`stripe_tax.go`** - Stripe Tax calculator adapter
- **`errors.go`** - Billing-specific error types
- **`mock.go`** - Mock provider for unit testing

### Testing
- **`stripe_test.go`** - Unit tests (35 test cases, 70% coverage)
- **`stripe_integration_test.go`** - Integration tests with real Stripe API
- **`INTEGRATION_TESTING.md`** - Guide for running integration tests

### Webhook Support
- **`../handler/webhook/stripe.go`** - Webhook event handler

## Quick Start

### 1. Configuration

```go
import "github.com/dukerupert/freyja/internal/billing"

config := billing.StripeConfig{
    APIKey:          os.Getenv("STRIPE_SECRET_KEY"),
    WebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
    EnableStripeTax: false,
    MaxRetries:      3,
    TimeoutSeconds:  30,
}

provider, err := billing.NewStripeProvider(config)
if err != nil {
    log.Fatal(err)
}
```

### 2. Create Payment Intent (Checkout)

```go
params := billing.CreatePaymentIntentParams{
    AmountCents: 5000, // $50.00
    Currency:    "usd",
    Description: "Order #1234",
    CustomerEmail: "customer@example.com",
    Metadata: map[string]string{
        "tenant_id":  "tenant_abc",  // REQUIRED for multi-tenant isolation
        "cart_id":    "cart_123",
        "order_type": "retail",
    },
    IdempotencyKey: "cart_123", // Prevents duplicate payment intents
}

pi, err := provider.CreatePaymentIntent(ctx, params)
if err != nil {
    return err
}

// Send pi.ClientSecret to frontend for Stripe.js
```

### 3. Verify Payment (Order Creation)

```go
// After frontend confirms payment, verify before creating order
pi, err := provider.GetPaymentIntent(ctx, billing.GetPaymentIntentParams{
    PaymentIntentID: "pi_...",
    TenantID:        "tenant_abc", // REQUIRED for security
})
if err != nil {
    return err
}

if pi.Status != "succeeded" {
    return billing.ErrPaymentFailed
}

// Safe to create order now
```

### 4. Handle Cart Changes

```go
// Customer adds items during checkout
updated, err := provider.UpdatePaymentIntent(ctx, billing.UpdatePaymentIntentParams{
    PaymentIntentID: "pi_...",
    TenantID:        "tenant_abc", // REQUIRED
    AmountCents:     7500, // $75.00 (new total)
    Metadata: map[string]string{
        "updated_at": time.Now().Format(time.RFC3339),
    },
})
```

### 5. Cancel Abandoned Checkouts

```go
// Background job to clean up old carts
err := provider.CancelPaymentIntent(ctx, "pi_...", "tenant_abc")
if err != nil {
    log.Printf("Failed to cancel payment intent: %v", err)
}
```

## Multi-Tenant Security

**CRITICAL**: All payment intent operations validate tenant isolation.

### ✅ Secure Usage

```go
// Always provide tenant_id in metadata when creating
params := billing.CreatePaymentIntentParams{
    Metadata: map[string]string{
        "tenant_id": currentTenant.ID, // Required!
    },
}

// Always provide TenantID when retrieving/updating/canceling
pi, err := provider.GetPaymentIntent(ctx, billing.GetPaymentIntentParams{
    PaymentIntentID: "pi_...",
    TenantID:        currentTenant.ID, // Required!
})
```

### ❌ Insecure Usage (Will Fail)

```go
// Missing tenant_id in metadata
params := billing.CreatePaymentIntentParams{
    Metadata: map[string]string{
        "cart_id": "cart_123",
        // ERROR: tenant_id missing!
    },
}
// Returns error: "tenant_id is required in metadata"

// Wrong tenant_id when retrieving
pi, err := provider.GetPaymentIntent(ctx, billing.GetPaymentIntentParams{
    PaymentIntentID: "pi_from_tenant_a",
    TenantID:        "tenant_b", // Wrong tenant!
})
// Returns: ErrPaymentIntentNotFound (doesn't leak existence)
```

## Testing

### Unit Tests (Fast)

```bash
go test ./internal/billing/...
```

Uses MockProvider - no real Stripe API calls.

### Integration Tests (Requires Stripe Keys)

1. Add test keys to `.env.test`:
```bash
STRIPE_SECRET_KEY=sk_test_51xxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxx
```

2. Run integration tests:
```bash
go test -tags=integration -v ./internal/billing/...
```

See [INTEGRATION_TESTING.md](INTEGRATION_TESTING.md) for detailed guide.

## Webhook Handling

### 1. Setup Webhook Endpoint

```go
// main.go
import "github.com/dukerupert/freyja/internal/handler/webhook"

webhookHandler := webhook.NewStripeHandler(billingProvider, webhook.StripeWebhookConfig{
    WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
    TenantID:      os.Getenv("TENANT_ID"),
})

http.HandleFunc("/webhooks/stripe", webhookHandler.HandleWebhook)
```

### 2. Test with Stripe CLI

```bash
# Terminal 1: Start your app
go run main.go

# Terminal 2: Forward webhooks
stripe listen --forward-to localhost:3000/webhooks/stripe

# Terminal 3: Trigger test events
stripe trigger payment_intent.succeeded
stripe trigger payment_intent.payment_failed
```

### 3. Handle Events

The webhook handler processes these events:
- `payment_intent.succeeded` - Create order, send confirmation email
- `payment_intent.payment_failed` - Notify customer, log error
- `payment_intent.canceled` - Clean up abandoned cart

## Error Handling

### Standard Errors

```go
import "github.com/dukerupert/freyja/internal/billing"

// Check for specific errors
if errors.Is(err, billing.ErrPaymentIntentNotFound) {
    return http.StatusNotFound
}

if errors.Is(err, billing.ErrAmountTooSmall) {
    return http.StatusBadRequest
}
```

### Stripe-Specific Errors

```go
// Type assert to get Stripe error details
var stripeErr *billing.StripeError
if errors.As(err, &stripeErr) {
    if stripeErr.IsDeclined() {
        // Card was declined
        log.Printf("Card declined: %s", stripeErr.DeclineCode)
    }

    if stripeErr.IsTemporary() {
        // Retry-able error (rate limit, network issue)
        log.Printf("Temporary error, will retry")
    }
}
```

## MVP vs Post-MVP Methods

### MVP Methods (Implemented)
- ✅ `CreatePaymentIntent` - Payment creation
- ✅ `GetPaymentIntent` - Payment verification
- ✅ `UpdatePaymentIntent` - Cart changes
- ✅ `CancelPaymentIntent` - Cleanup
- ✅ `VerifyWebhookSignature` - Security

### Post-MVP Methods (Stubs)
- ⏳ `CreateCustomer` - Returns `ErrNotImplemented`
- ⏳ `GetCustomer` - Returns `ErrNotImplemented`
- ⏳ `UpdateCustomer` - Returns `ErrNotImplemented`
- ⏳ `CreateSubscription` - Returns `ErrNotImplemented`
- ⏳ `CancelSubscription` - Returns `ErrNotImplemented`
- ⏳ `RefundPayment` - Returns `ErrNotImplemented`

Post-MVP methods will be implemented when subscription and refund features are added.

## Stripe Tax (Optional)

To enable Stripe Tax:

1. **Enable in Stripe Dashboard**:
   - Go to https://dashboard.stripe.com/settings/tax
   - Enable Stripe Tax
   - Configure your business address

2. **Update Configuration**:
```go
config := billing.StripeConfig{
    EnableStripeTax: true,
}
```

3. **Provide Shipping Address**:
```go
params := billing.CreatePaymentIntentParams{
    EnableStripeTax: true,
    ShippingAddress: &billing.PaymentAddress{
        Line1:      "123 Main St",
        City:       "San Francisco",
        State:      "CA",
        PostalCode: "94102",
        Country:    "US",
    },
    LineItems: []billing.PaymentLineItem{
        {
            ProductID:   "prod_123",
            Description: "Coffee Beans",
            Quantity:    2,
            AmountCents: 5000,
            TaxCode:     "txcd_30011000", // Food & beverages
        },
    },
}
```

4. **Tax Included in Response**:
```go
pi, _ := provider.CreatePaymentIntent(ctx, params)
fmt.Printf("Subtotal: $%.2f\n", float64(pi.AmountCents-pi.TaxCents)/100)
fmt.Printf("Tax: $%.2f\n", float64(pi.TaxCents)/100)
fmt.Printf("Total: $%.2f\n", float64(pi.AmountCents)/100)
```

## Production Checklist

Before going live:

- [ ] Replace test keys with live keys
- [ ] Enable webhook endpoint in Stripe dashboard
- [ ] Configure webhook endpoint URL (https://yourapp.com/webhooks/stripe)
- [ ] Test with real credit cards in test mode first
- [ ] Review Stripe Tax configuration if enabled
- [ ] Set up monitoring for webhook failures
- [ ] Configure retry logic for webhook processing
- [ ] Review and test error handling flows
- [ ] Verify tenant isolation in production
- [ ] Enable Stripe Radar for fraud detection
- [ ] Configure receipt email settings in Stripe

## Resources

- **Stripe Go SDK**: https://github.com/stripe/stripe-go
- **Stripe API Docs**: https://stripe.com/docs/api
- **Payment Intents Guide**: https://stripe.com/docs/payments/payment-intents
- **Webhook Guide**: https://stripe.com/docs/webhooks
- **Testing Guide**: https://stripe.com/docs/testing
- **Stripe CLI**: https://stripe.com/docs/stripe-cli

## Support

For Stripe integration questions:
- Check integration tests for examples
- Review INTEGRATION_TESTING.md for testing guide
- See webhook handler example in `internal/handler/webhook/stripe.go`
- Consult Stripe documentation: https://stripe.com/docs
