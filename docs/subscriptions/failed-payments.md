# Failed Payments

Handling subscription payment issues.

## Why Payments Fail

Common reasons for failed subscription payments:

- **Expired card** - Card reached expiration date
- **Insufficient funds** - Not enough balance
- **Card declined** - Bank rejected the charge
- **Lost/stolen card** - Card was cancelled
- **Outdated info** - Card number changed

## What Happens When Payment Fails

1. Stripe attempts to charge the card
2. Payment fails
3. Subscription moves to **Past Due** status
4. Stripe begins retry attempts
5. Customer receives failed payment notification

## Automatic Retries

Stripe automatically retries failed payments:

- First retry: 3 days after failure
- Second retry: 5 days after first retry
- Third retry: 7 days after second retry

If all retries fail, subscription moves to **Expired**.

## Customer Notifications

Customers receive emails when:

- Payment fails initially
- Each retry attempt fails
- Subscription is about to expire
- Subscription expires

These emails include links to update payment information.

## Past Due Subscriptions

Subscriptions in past due status:

- No orders created
- Customer notified
- Retries scheduled
- Can be recovered

### Viewing Past Due
1. Go to **Subscriptions**
2. Filter by **Past Due**
3. See all subscriptions needing attention

## Recovering Failed Payments

### Customer Self-Service
Customers can update their payment method:
1. Click link in failed payment email
2. Access Stripe Customer Portal
3. Update payment information
4. Payment retried automatically

### Admin Options
You can:
- Contact customer directly
- Point them to customer portal
- View details in Stripe Dashboard

## Preventing Payment Failures

### Card Updater
Stripe's card updater automatically:
- Updates expired cards
- Refreshes changed card numbers
- Works with participating banks

### Pre-Billing Reminders
Consider reminding customers before billing:
- "Your subscription renews in 3 days"
- Gives them time to update payment if needed

## Expired Subscriptions

If all retries fail:
- Subscription status becomes **Expired**
- No more charges attempted
- Customer must start new subscription

## Best Practices

### Monitor Actively
- Check past due subscriptions regularly
- Reach out before expiration
- Personal touch can save subscriptions

### Make Recovery Easy
- Ensure customer portal access works
- Respond quickly to payment questions
- Consider flexibility for good customers

### Track Patterns
- High failure rate may indicate pricing issues
- Consistent failures from same cards need attention
- Seasonal patterns may exist

---

Previous: [Managing Subscriptions](management.md) | Back to [Subscriptions](index.md)
