# Managing Subscriptions

View and manage active subscriptions.

## Viewing Subscriptions

1. Go to **Subscriptions** in the sidebar
2. See all subscriptions with status
3. Filter by status (active, paused, cancelled)
4. Click a subscription for details

## Subscription Details

Each subscription shows:

- **Customer**: Who subscribed
- **Product**: What they're receiving
- **Frequency**: How often they receive it
- **Status**: Current subscription state
- **Next billing**: When they'll be charged next
- **Created**: When subscription started
- **Orders**: History of subscription orders

## Subscription States

| Status | Description | Orders Created? |
|--------|-------------|-----------------|
| Active | Running normally | Yes |
| Paused | Temporarily stopped | No |
| Past Due | Payment failed | No (until resolved) |
| Cancelled | Customer cancelled | No |
| Expired | Failed payments exhausted | No |

## Admin Actions

### View in Stripe
Click **View in Stripe** to see full subscription details in Stripe Dashboard.

### View Customer
Click through to the customer record for account details.

### View Orders
See all orders created from this subscription.

## Customer Self-Service

Customers manage their own subscriptions through the Stripe Customer Portal:

- Change delivery frequency
- Update payment method
- Pause subscription
- Resume subscription
- Cancel subscription

This reduces admin work and gives customers immediate control.

## Paused Subscriptions

When a customer pauses:
- No charges occur
- No orders created
- Subscription remains in system
- Customer can resume anytime

Pausing is useful for:
- Vacation / travel
- Too much coffee built up
- Temporary budget concerns

## Cancelled Subscriptions

When cancelled:
- No future charges
- Subscription marked as cancelled
- Historical orders remain
- Customer can start new subscription later

## Monitoring Subscriptions

### Key Metrics
- Active subscription count
- New subscriptions this month
- Churned subscriptions this month
- Subscription revenue

### Warning Signs
- Increasing pause rate
- High cancellation rate
- Many past due subscriptions

### Healthy Signs
- Growing active count
- Low churn rate
- Few past due issues

## Tips for Success

### Reduce Churn
- Offer pause option (vs. cancel)
- Send pre-billing reminders
- Maintain product quality

### Handle Issues Promptly
- Monitor past due subscriptions
- Reach out to customers with payment issues
- Resolve problems before cancellation

---

Previous: [Subscription Plans](plans.md) | Next: [Failed Payments](failed-payments.md)
