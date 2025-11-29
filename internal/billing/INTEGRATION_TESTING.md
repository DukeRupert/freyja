# Stripe Integration Testing Guide

This guide explains how to run integration tests against the real Stripe API using test mode credentials.

## Prerequisites

1. **Stripe Test Account**: Create a free account at https://dashboard.stripe.com/register
2. **Test API Keys**: Get your test keys from https://dashboard.stripe.com/test/apikeys
3. **Stripe CLI** (for webhook testing): Install from https://stripe.com/docs/stripe-cli

## Setup

### 1. Configure Test Credentials

Add your Stripe test credentials to `.env.test`:

```bash
# Stripe (get from https://dashboard.stripe.com/test/apikeys)
STRIPE_SECRET_KEY=sk_test_51xxxxx  # Your actual test secret key
STRIPE_WEBHOOK_SECRET=whsec_xxxxx  # Your webhook signing secret
```

**IMPORTANT**: Never use live mode keys (sk_live_...) for testing! The integration tests will refuse to run with live keys to prevent accidental charges.

### 2. Install Dependencies

The integration tests require the `godotenv` package to load environment variables:

```bash
go get github.com/joho/godotenv
```

## Running Integration Tests

Integration tests are tagged with `//go:build integration` and are **skipped by default** during normal test runs.

### Run All Integration Tests

```bash
go test -tags=integration -v ./internal/billing/...
```

### Run Specific Integration Test

```bash
go test -tags=integration -v ./internal/billing/... -run TestStripeIntegration_CreatePaymentIntent
```

### Run Without Integration Tests (Default)

```bash
go test ./internal/billing/...
```

This runs only the fast unit tests with MockProvider (no real Stripe API calls).

## What the Integration Tests Cover

### 1. CreatePaymentIntent
- ✅ Creates real payment intents via Stripe API
- ✅ Validates minimum amount enforcement
- ✅ Tests customer email field
- ✅ Verifies idempotency key behavior
- ✅ Checks metadata preservation

### 2. GetPaymentIntent
- ✅ Retrieves payment intents
- ✅ Verifies all fields match
- ✅ Tests tenant isolation enforcement

### 3. UpdatePaymentIntent
- ✅ Updates payment amount (cart changes)
- ✅ Updates description and metadata
- ✅ Verifies tenant validation

### 4. CancelPaymentIntent
- ✅ Cancels unconfirmed payment intents
- ✅ Tests idempotency (canceling already-canceled)
- ✅ Verifies tenant validation

### 5. Tenant Isolation
- ✅ Creates payment intent for tenant A
- ✅ Verifies tenant A can retrieve it
- ✅ Verifies tenant B **cannot** retrieve it (security test)

### 6. Error Handling
- ✅ Non-existent payment intent returns ErrPaymentIntentNotFound
- ✅ Missing tenant_id validation
- ✅ Invalid amounts rejected

### 7. Idempotency
- ✅ Same idempotency key returns same payment intent
- ✅ Prevents duplicate payment intents

## Viewing Test Results in Stripe Dashboard

After running integration tests, you can view the created payment intents in your Stripe Dashboard:

1. Go to https://dashboard.stripe.com/test/payments
2. Look for payment intents with descriptions like "Integration test payment"
3. Check metadata for `tenant_id: tenant_integration_test`

Each test logs the payment intent ID:
```
Created payment intent: pi_xxxxx (view at https://dashboard.stripe.com/test/payments/pi_xxxxx)
```

Click the link to view details in Stripe Dashboard.

## Webhook Testing

### Setup Stripe CLI

1. **Install Stripe CLI**:
   ```bash
   # macOS
   brew install stripe/stripe-cli/stripe

   # Linux
   wget https://github.com/stripe/stripe-cli/releases/download/v1.19.0/stripe_1.19.0_linux_x86_64.tar.gz
   tar -xvf stripe_1.19.0_linux_x86_64.tar.gz
   sudo mv stripe /usr/local/bin/
   ```

2. **Login to Stripe**:
   ```bash
   stripe login
   ```
   This opens your browser to authenticate with your Stripe account.

### Local Webhook Testing

#### 1. Start Your Application
```bash
go run main.go
# Application should be listening on port 3000 (or your configured port)
```

#### 2. Start Stripe CLI Listener

In a separate terminal, forward Stripe webhook events to your local server:

```bash
# Forward to your webhook endpoint
stripe listen --forward-to localhost:3000/webhooks/stripe

# The CLI will output a webhook signing secret like:
# > Ready! Your webhook signing secret is whsec_xxxxx (^C to quit)
```

**Copy the `whsec_xxxxx` secret** and update `.env.test`:
```bash
STRIPE_WEBHOOK_SECRET=whsec_xxxxx  # Secret from Stripe CLI
```

#### 3. Trigger Test Events

In another terminal, trigger Stripe events:

```bash
# Trigger successful payment
stripe trigger payment_intent.succeeded

# Trigger failed payment
stripe trigger payment_intent.payment_failed

# Trigger payment intent creation
stripe trigger payment_intent.created
```

#### 4. Watch Webhook Handler Logs

Your application should receive the webhook events and process them. Check your application logs to verify:
- Webhook signature verification passes
- Events are processed correctly
- Tenant isolation is maintained

### Webhook Handler Example

Here's a basic webhook handler structure (to be implemented):

```go
// internal/handler/webhook/stripe.go
func HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
    provider := billing.NewStripeProvider(config)

    // Read webhook payload
    payload, _ := ioutil.ReadAll(r.Body)
    signature := r.Header.Get("Stripe-Signature")

    // Verify signature
    err := provider.VerifyWebhookSignature(payload, signature, webhookSecret)
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Parse event
    event := stripe.Event{}
    json.Unmarshal(payload, &event)

    // Handle event type
    switch event.Type {
    case "payment_intent.succeeded":
        // Extract payment intent ID from event
        // Call GetPaymentIntent to verify tenant
        // Create order in database

    case "payment_intent.payment_failed":
        // Notify customer of failed payment
        // Log for debugging

    default:
        log.Printf("Unhandled event type: %s", event.Type)
    }

    w.WriteHeader(http.StatusOK)
}
```

## Testing Stripe Tax (Optional)

If you enable Stripe Tax in your test account:

1. **Enable Stripe Tax**:
   - Go to https://dashboard.stripe.com/test/settings/tax
   - Enable Stripe Tax
   - Configure your business address

2. **Update Configuration**:
   ```go
   config := billing.StripeConfig{
       APIKey:          "sk_test_...",
       WebhookSecret:   "whsec_...",
       EnableStripeTax: true,  // Enable tax calculation
   }
   ```

3. **Test Tax Calculation**:
   ```go
   params := billing.CreatePaymentIntentParams{
       AmountCents: 10000,
       Currency:    "usd",
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

4. **Verify Tax**:
   - Check that `PaymentIntent.TaxCents` is calculated
   - View tax breakdown in Stripe Dashboard

## CI/CD Integration

To run integration tests in GitHub Actions or other CI/CD:

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Run unit tests
        run: go test ./...

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Create .env.test
        run: |
          echo "STRIPE_SECRET_KEY=${{ secrets.STRIPE_TEST_SECRET_KEY }}" >> .env.test
          echo "STRIPE_WEBHOOK_SECRET=${{ secrets.STRIPE_TEST_WEBHOOK_SECRET }}" >> .env.test
      - name: Run integration tests
        run: go test -tags=integration -v ./...
```

**Store secrets in GitHub**:
- Go to Settings → Secrets → Actions
- Add `STRIPE_TEST_SECRET_KEY` and `STRIPE_TEST_WEBHOOK_SECRET`

## Troubleshooting

### "Skipping integration test: .env.test not found"

**Solution**: Make sure `.env.test` exists in the project root (two directories up from the test file):
```
freyja/
├── .env.test                    ← Should be here
├── internal/
│   └── billing/
│       └── stripe_integration_test.go
```

### "DANGER: Live Stripe key detected!"

**Solution**: You're using a live key (sk_live_...) instead of a test key. Replace with your test key (sk_test_...) from https://dashboard.stripe.com/test/apikeys

### "Invalid signature" in webhook tests

**Solution**:
1. Make sure you're using the webhook secret from `stripe listen` output (starts with `whsec_`)
2. Update `.env.test` with the correct secret
3. Restart the Stripe CLI listener

### Tests fail with "resource_missing" error

**Solution**: The payment intent may have expired or been deleted. Integration tests create new payment intents each run, so this shouldn't happen unless you're testing with old IDs.

## Cleanup

Integration tests create real payment intents in your Stripe test account. They remain in "requires_payment_method" status and don't incur charges, but you can clean them up:

1. **Manual Cleanup**:
   - Go to https://dashboard.stripe.com/test/payments
   - Filter by metadata: `tenant_id:tenant_integration_test`
   - Delete old test payment intents

2. **Automatic Cleanup** (not implemented yet):
   Could add teardown functions to cancel all created payment intents after each test.

## Best Practices

1. **Run integration tests before deploying** to catch Stripe API changes
2. **Use unique idempotency keys** with timestamps to avoid conflicts
3. **Monitor Stripe Dashboard** during test development to verify behavior
4. **Never commit .env.test** with real keys to version control
5. **Use Stripe CLI** for webhook testing instead of exposing localhost to internet
6. **Test tenant isolation** thoroughly - this is critical for security

## Next Steps

Once integration tests pass:
1. Implement webhook handler in `internal/handler/webhook/stripe.go`
2. Add route for `/webhooks/stripe` in main.go
3. Test end-to-end checkout flow with Stripe.js on frontend
4. Set up production Stripe account with live keys (separate from test)
