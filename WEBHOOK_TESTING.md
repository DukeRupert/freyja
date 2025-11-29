# Webhook Testing Guide

Your webhook endpoint is now set up! Here's how to test it with Stripe CLI.

## Prerequisites

✅ Stripe CLI is installed at `/bin/stripe`
✅ Webhook handler is wired up to `/webhooks/stripe`
✅ Application is ready to run

## Step-by-Step Testing

### Terminal 1: Start Your Application

```bash
go run cmd/server/main.go
```

You should see:
```
INFO Connecting to database...
INFO Database connection established
INFO Running database migrations...
INFO Database migrations completed successfully
INFO Loading templates...
INFO Templates loaded successfully
INFO Initializing Stripe billing provider...
INFO Stripe billing provider initialized test_mode=true
INFO Starting server address=:3000
```

### Terminal 2: Start Stripe CLI Listener

```bash
stripe listen --forward-to localhost:3000/webhooks/stripe
```

**IMPORTANT**: Copy the webhook signing secret from the output:
```
> Ready! Your webhook signing secret is whsec_xxxxxxxxxxxxx (^C to quit)
```

Update `.env.test` with this secret:
```bash
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxx  # Paste the secret here
```

Then **restart your application** (Ctrl+C in Terminal 1, then `go run cmd/server/main.go` again)

### Terminal 3: Trigger Test Events

Now you can trigger webhook events:

```bash
# Trigger successful payment
stripe trigger payment_intent.succeeded
```

### What You'll See

**In Terminal 1 (Application logs):**
```
INFO POST /webhooks/stripe 200
Received Stripe webhook event: payment_intent.succeeded (ID: evt_xxxxx)
Payment succeeded for payment intent: pi_xxxxx (amount: 1099 usd)
Creating order - tenant: tenant_abc, cart: cart_123, type: retail
```

**In Terminal 2 (Stripe CLI):**
```
2025-01-29 10:15:42  --> payment_intent.succeeded [evt_xxxxx]
2025-01-29 10:15:42  <--  [200] POST http://localhost:3000/webhooks/stripe [evt_xxxxx]
```

## Testing Different Event Types

### Successful Payment
```bash
stripe trigger payment_intent.succeeded
```
- Application logs: "Payment succeeded..."
- Shows amount and currency
- Logs tenant/cart/order_type from metadata

### Failed Payment
```bash
stripe trigger payment_intent.payment_failed
```
- Application logs: "Payment failed..."
- Shows failure reason and error codes

### Canceled Payment
```bash
stripe trigger payment_intent.canceled
```
- Application logs: "Payment intent canceled..."

## Verifying Webhook Signature

The application automatically verifies webhook signatures. Try sending an invalid webhook:

```bash
curl -X POST http://localhost:3000/webhooks/stripe \
  -H "Content-Type: application/json" \
  -d '{"type": "fake.event"}'
```

You should see:
```
Invalid signature
```

This proves your webhook is protected against forged events!

## What the Webhook Handler Currently Does

Right now, the webhook handler:
- ✅ Receives webhook events from Stripe
- ✅ Verifies webhook signatures (security!)
- ✅ Parses event data
- ✅ Logs payment details
- ✅ Extracts metadata (tenant_id, cart_id, order_type)
- ⏳ **Doesn't create orders yet** (TODOs in place)

## Common Issues

### "Missing Stripe-Signature header"
- You're calling the endpoint directly without Stripe CLI
- Use `stripe trigger` commands instead

### "Invalid signature"
- Webhook secret in `.env.test` doesn't match Stripe CLI output
- Copy the `whsec_xxx` from Stripe CLI and update `.env.test`
- Restart the application

### "Connection refused"
- Application isn't running on port 3000
- Check Terminal 1 to make sure server started

### "No route found"
- Webhook route isn't registered
- Verify main.go has: `r.Post("/webhooks/stripe", ...)`

## Next Steps

Once webhooks are working:

1. **Test with real checkout flow:**
   - Create payment intent via API
   - Use Stripe.js to confirm payment
   - Webhook fires automatically

2. **Implement order creation:**
   - Complete the TODOs in `/internal/handler/webhook/stripe.go`
   - Create orders from `payment_intent.succeeded`
   - Send confirmation emails

3. **Add to production:**
   - Configure webhook endpoint in Stripe Dashboard
   - Use production webhook secret
   - Monitor webhook delivery in Stripe Dashboard

## Viewing Webhooks in Stripe Dashboard

After testing, you can view webhook events:
- Go to https://dashboard.stripe.com/test/webhooks
- See all triggered events
- View request/response details
- Check delivery status

## Stopping the Test

Press `Ctrl+C` in all three terminals to stop:
1. Stripe CLI (Terminal 2)
2. Application (Terminal 1)
3. Back to normal shell (Terminal 3)
