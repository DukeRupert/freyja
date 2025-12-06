# Connecting Stripe

Freyja uses Stripe for all payment processing, including one-time purchases, subscriptions, and invoicing. This guide walks through connecting your Stripe account.

## Prerequisites

- A Stripe account (create one at [stripe.com](https://stripe.com) if needed)
- Business verification completed in Stripe (for receiving payouts)

## Connecting Your Account

1. In your Freyja dashboard, go to **Settings > Payments**
2. Click **Connect Stripe Account**
3. You'll be redirected to Stripe to authorize the connection
4. Log in to your Stripe account (or create one)
5. Review and approve the connection
6. You'll be redirected back to Freyja

Once connected, you'll see your Stripe account status in the Payments settings.

## Test Mode vs Live Mode

Stripe provides two modes:

- **Test Mode** - Use fake card numbers for testing. No real charges.
- **Live Mode** - Real payments and payouts.

During setup, keep Stripe in test mode until you're ready to launch. Test card numbers:

- **Successful payment:** 4242 4242 4242 4242
- **Declined card:** 4000 0000 0000 0002
- **Requires authentication:** 4000 0025 0000 3155

Use any future expiration date and any 3-digit CVC.

## Webhook Configuration

Freyja automatically configures webhooks when you connect Stripe. These webhooks notify Freyja when:

- Payments succeed or fail
- Subscriptions are created, updated, or cancelled
- Invoices are paid or become overdue
- Disputes are opened

You don't need to manually configure webhooks.

## What Stripe Handles

With Stripe connected, Freyja uses:

- **Stripe Payments** - One-time checkout purchases
- **Stripe Billing** - Subscription management and recurring payments
- **Stripe Invoicing** - Wholesale invoice creation and payment
- **Stripe Tax** (optional) - Automatic tax calculation

## Stripe Fees

Stripe charges standard processing fees (typically 2.9% + $0.30 per transaction in the US). Freyja does not add any additional transaction fees.

## Troubleshooting

### Connection Failed
- Ensure you're logged into the correct Stripe account
- Check that your Stripe account has completed business verification
- Try disconnecting and reconnecting

### Webhooks Not Working
- Check your Stripe webhook settings in the Stripe Dashboard
- Verify the webhook endpoint URL is correct
- Check webhook signing secret is properly configured

### Test Payments Not Working
- Confirm Stripe is in test mode
- Use valid test card numbers (listed above)
- Check that the expiration date is in the future

---

Previous: [Quick Start Guide](quick-start.md) | Next: [Your First Product](first-product.md)
