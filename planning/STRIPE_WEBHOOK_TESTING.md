# Stripe Webhook Testing Guide

This document describes how to test Stripe webhook handling in Freyja using the Stripe CLI and the test mode feature.

## Overview

Freyja processes Stripe webhook events to:
1. **Create orders** from successful checkout payments (`payment_intent.succeeded`)
2. **Create subscription renewal orders** from invoice payments (`invoice.payment_succeeded`)
3. **Update subscription status** on payment failures and cancellations

Testing these webhooks requires either:
- **Test Mode**: Quick validation that webhooks are received and parsed correctly
- **Full Integration**: End-to-end testing with real checkout flows and order creation

## Metadata Requirements

### PaymentIntent (Standard Checkout)

The webhook handler expects these metadata fields on PaymentIntents:

| Field | Required | Purpose |
|-------|----------|---------|
| `tenant_id` | **Yes** | Multi-tenant isolation - must match configured tenant |
| `cart_id` | **Yes** | Retrieves cart items to create order |
| `customer_email` | Yes | Guest checkout email address |
| `shipping_address` | Yes | JSON-serialized shipping address |
| `billing_address` | Yes | JSON-serialized billing address |
| `shipping_rate_id` | Yes | Selected shipping method reference |
| `subtotal_cents` | Yes | Order subtotal for verification |
| `shipping_cents` | Yes | Shipping cost |
| `tax_cents` | Yes | Tax amount |
| `tax_calculation_id` | Yes | Stripe Tax audit trail |
| `customer_notes` | Optional | Customer's order notes |

These are set automatically by `CheckoutService.CreatePaymentIntent()` in `internal/service/checkout.go`.

### Subscription (Renewal Orders)

Subscription metadata is set when creating subscriptions via `SubscriptionService.CreateSubscription()`:

| Field | Required | Purpose |
|-------|----------|---------|
| `tenant_id` | **Yes** | Multi-tenant isolation |
| `subscription_id` | Yes | Links to local subscription record |
| `user_id` | Yes | Customer identification |
| `product_sku_id` | Yes | Product being delivered |
| `billing_interval` | Yes | Subscription frequency |

## Test Mode

### What Test Mode Does

When `STRIPE_WEBHOOK_TEST_MODE=true`:

1. **Signature verification still runs** - webhooks must be properly signed
2. **Events are parsed and logged** - validates JSON structure and event handling
3. **Tenant validation is bypassed** - allows CLI-triggered events without metadata
4. **Business logic is skipped** - no orders created, no subscriptions updated
5. **Clear logging** - all test mode actions prefixed with `[TEST MODE]`

### When to Use Test Mode

- Verifying webhook endpoint is reachable
- Testing signature verification works
- Validating event parsing without side effects
- Quick smoke tests during development

### When NOT to Use Test Mode

- Production environments (obviously)
- Testing actual order creation
- Validating subscription renewal flows
- Pre-launch integration testing

## Setup

### 1. Install Stripe CLI

```bash
# macOS
brew install stripe/stripe-cli/stripe

# Linux (Debian/Ubuntu)
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list
sudo apt update && sudo apt install stripe

# Verify installation
stripe version
```

### 2. Login to Stripe

```bash
stripe login
# Follow the browser prompt to authenticate
```

### 3. Start Webhook Listener

```bash
stripe listen --forward-to localhost:3000/webhooks/stripe
```

This outputs a webhook signing secret like:
```
Ready! Your webhook signing secret is whsec_xxxxxxxxxxxxxxxxxxxxx
```

### 4. Configure Environment

Add the webhook secret to your `.env`:
```bash
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxxxxxxxxxx
```

For test mode, also add:
```bash
STRIPE_WEBHOOK_TEST_MODE=true
```

## Testing Scenarios

### Scenario 1: Quick Webhook Validation (Test Mode)

Verify webhooks are received and parsed without creating orders.

**Terminal 1 - Start server with test mode:**
```bash
STRIPE_WEBHOOK_TEST_MODE=true go run cmd/server/main.go
```

**Terminal 2 - Start Stripe CLI:**
```bash
stripe listen --forward-to localhost:3000/webhooks/stripe
```

**Terminal 3 - Trigger events:**
```bash
# Standard checkout events
stripe trigger payment_intent.succeeded
stripe trigger payment_intent.payment_failed
stripe trigger payment_intent.canceled

# Subscription events
stripe trigger invoice.payment_succeeded
stripe trigger invoice.payment_failed
stripe trigger customer.subscription.created
stripe trigger customer.subscription.updated
stripe trigger customer.subscription.deleted
```

**Expected Log Output:**
```
[WEBHOOK] Received request: POST /webhooks/stripe
[WEBHOOK] Signature verification SUCCESS
Received Stripe webhook event: payment_intent.succeeded (ID: evt_xxx)
Payment succeeded for payment intent: pi_xxx (amount: 2000 usd)
Creating order - tenant: , cart: , type:
[TEST MODE] Tenant mismatch ignored - expected: <your-tenant-id>, got:
[TEST MODE] Skipping order creation (missing required metadata)
[TEST MODE] ✓ Webhook received and parsed successfully
```

### Scenario 2: Full Checkout Integration

Test actual order creation through the storefront.

**Terminal 1 - Start server (test mode OFF):**
```bash
go run cmd/server/main.go
```

**Terminal 2 - Start Stripe CLI:**
```bash
stripe listen --forward-to localhost:3000/webhooks/stripe
```

**Browser - Complete checkout:**
1. Add items to cart at `http://localhost:3000/`
2. Proceed to checkout
3. Enter shipping/billing information
4. Use test card: `4242 4242 4242 4242` (any future expiry, any CVC)
5. Complete payment

**Expected Results:**
- Order created in database
- Order confirmation page displays order number
- Webhook logs show successful order creation:
  ```
  Order created successfully: ORD-XXXXX (payment: pi_xxx, total: 5000 usd)
  ```

### Scenario 3: Subscription Renewal

Test subscription invoice processing.

**Setup - Create a subscription:**
1. Log in as a customer
2. Subscribe to a product through the storefront
3. Note the subscription ID created

**Test - Trigger renewal (Stripe Dashboard):**
1. Go to Stripe Dashboard > Subscriptions
2. Find the test subscription
3. Click "Actions" > "Create invoice" or wait for billing cycle

**Expected Results:**
- `invoice.payment_succeeded` webhook received
- Renewal order created linked to subscription
- Log output:
  ```
  Subscription renewal order created successfully: ORD-XXXXX (invoice: in_xxx, subscription: sub_xxx)
  ```

### Scenario 4: Failed Payment Handling

Test subscription payment failure flow.

**Stripe CLI:**
```bash
stripe trigger invoice.payment_failed
```

**Or use a declining test card during checkout:**
- Card number: `4000 0000 0000 0002` (generic decline)
- Card number: `4000 0000 0000 9995` (insufficient funds)

**Expected Results:**
- Subscription status updated to `past_due`
- Log output indicates status sync

## Automated Test Script

Use the provided test script for automated validation:

```bash
./scripts/test-stripe-webhooks.sh
```

See `scripts/test-stripe-webhooks.sh` for implementation.

## Stripe CLI Reference

### Forwarding Commands

```bash
# Forward all events
stripe listen --forward-to localhost:3000/webhooks/stripe

# Forward specific events only
stripe listen \
  --events payment_intent.succeeded,invoice.payment_succeeded \
  --forward-to localhost:3000/webhooks/stripe

# Skip HTTPS verification (for self-signed certs)
stripe listen --forward-to localhost:3000/webhooks/stripe --skip-verify
```

### Trigger Commands

```bash
# Basic trigger
stripe trigger <event_type>

# With metadata overrides (advanced)
stripe trigger payment_intent.succeeded \
  --override payment_intent:metadata.tenant_id=your-tenant-uuid \
  --override payment_intent:metadata.cart_id=your-cart-uuid
```

### Available Trigger Events

**Payment Events:**
- `payment_intent.created`
- `payment_intent.succeeded`
- `payment_intent.payment_failed`
- `payment_intent.canceled`
- `charge.succeeded`
- `charge.failed`
- `charge.refunded`

**Checkout Events:**
- `checkout.session.completed`
- `checkout.session.async_payment_succeeded`
- `checkout.session.async_payment_failed`

**Subscription Events:**
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`

**Invoice Events:**
- `invoice.created`
- `invoice.finalized`
- `invoice.paid`
- `invoice.payment_succeeded`
- `invoice.payment_failed`

## Webhook Handler Implementation

### File Locations

- **Handler**: `internal/handler/webhook/stripe.go`
- **Route Registration**: `internal/routes/webhook.go`
- **Configuration**: `cmd/server/main.go` (lines 376-390)

### Handled Events

| Event | Handler | Action |
|-------|---------|--------|
| `payment_intent.succeeded` | `handlePaymentIntentSucceeded` | Creates order from cart |
| `payment_intent.payment_failed` | `handlePaymentIntentFailed` | Logs failure |
| `payment_intent.canceled` | `handlePaymentIntentCanceled` | Logs cancellation |
| `invoice.payment_succeeded` | `handleInvoicePaymentSucceeded` | Creates subscription renewal order |
| `invoice.payment_failed` | `handleInvoicePaymentFailed` | Updates subscription to `past_due` |
| `customer.subscription.updated` | `handleSubscriptionUpdated` | Syncs subscription status |
| `customer.subscription.deleted` | `handleSubscriptionDeleted` | Marks subscription expired |

### Adding New Event Handlers

1. Add case to switch statement in `HandleWebhook()`
2. Create handler method following existing patterns
3. Include test mode support if handler modifies data
4. Update this documentation

## Troubleshooting

### "Invalid signature" errors

- Ensure `STRIPE_WEBHOOK_SECRET` matches the CLI output
- The CLI generates a new secret each time it starts
- Check for trailing whitespace in environment variable

### Events not received

- Verify server is running on correct port
- Check firewall/network settings
- Ensure CLI is pointing to correct URL

### Tenant mismatch warnings (not in test mode)

- Payment was created in different Stripe account
- Metadata was not set during PaymentIntent creation
- Check `CheckoutService.CreatePaymentIntent()` is being used

### Order not created after successful payment

- Check for `CRITICAL: Failed to create order` in logs
- Verify cart still exists and has items
- Check database connection
- Look for idempotency message (order may already exist)

## Security Considerations

1. **Never enable test mode in production** - it bypasses tenant isolation
2. **Webhook secrets are environment-specific** - use different secrets per environment
3. **Always verify signatures** - test mode still validates signatures
4. **Monitor webhook logs** - failed webhooks may indicate attacks or misconfigurations
