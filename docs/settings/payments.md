# Payment Settings

Stripe integration and payment configuration.

## Overview

Freyja uses Stripe for all payment processing:

- One-time checkout payments
- Subscription billing
- Invoice payments
- Refunds

## Connecting Stripe

### Initial Setup
1. Go to **Settings > Payments**
2. Click **Connect Stripe Account**
3. Log into Stripe (or create account)
4. Authorize the connection
5. Return to Freyja

### What Gets Connected
- Payment processing
- Customer management
- Subscription billing
- Invoice generation
- Webhook events

## Stripe Modes

### Test Mode
- Use for development and testing
- Fake card numbers work
- No real charges
- Test data stays separate

### Live Mode
- Real payment processing
- Real charges to real cards
- Payouts to your bank
- Use only when ready for production

## Test Card Numbers

Use these in test mode:

| Card | Number |
|------|--------|
| Successful | 4242 4242 4242 4242 |
| Declined | 4000 0000 0000 0002 |
| Requires Auth | 4000 0025 0000 3155 |

Any future expiration and any CVC work.

## Webhooks

Freyja automatically configures Stripe webhooks for:

| Event | Purpose |
|-------|---------|
| payment_intent.succeeded | Mark orders as paid |
| payment_intent.failed | Handle failed payments |
| invoice.paid | Record invoice payments |
| invoice.payment_failed | Handle failed invoice payments |
| customer.subscription.* | Sync subscription status |

You don't need to manually configure webhooks.

## Stripe Dashboard

Access your Stripe Dashboard for:

- Transaction history
- Payout schedule
- Dispute management
- Tax settings
- Customer details

Link: [dashboard.stripe.com](https://dashboard.stripe.com)

## Stripe Tax

If using Stripe Tax:

1. Enable in Stripe Dashboard
2. Configure tax registrations
3. Select "Stripe Tax" in Freyja tax settings

See [Tax Configuration](tax.md) for details.

## Fees

### Stripe Fees
Standard Stripe processing fees apply (typically 2.9% + $0.30 per transaction in the US).

### Freyja Fees
Freyja does not charge additional transaction fees. Flat monthly subscription only.

## Refunds

Process refunds through:
1. Freyja order detail (if implemented)
2. Or directly in Stripe Dashboard

Refunded orders update status automatically.

## Disputes

Handle chargebacks in Stripe Dashboard:
1. Receive dispute notification
2. Review in Stripe
3. Submit evidence if contesting
4. Outcome determined by card network

## Disconnecting Stripe

If you need to disconnect:
1. Go to Settings > Payments
2. Click **Disconnect**
3. Confirm disconnection

Note: This will prevent payment processing until reconnected.

## Troubleshooting

### Connection Failed
- Verify Stripe account is in good standing
- Complete Stripe onboarding/verification
- Try disconnecting and reconnecting

### Payments Not Processing
- Check Stripe is in correct mode (test vs live)
- Verify webhook configuration
- Check Stripe Dashboard for errors

### Webhooks Not Working
- Verify webhook endpoint in Stripe
- Check webhook signing secret
- Review webhook logs in Stripe Dashboard

---

Previous: [Email Settings](email.md) | Back to [Settings](index.md)
