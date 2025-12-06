# How Subscriptions Work

Understanding Freyja's subscription system.

## Subscription Flow

### Customer Signs Up

1. Customer browses your store
2. Selects a product with subscription option
3. Chooses their preferences:
   - Weight/size
   - Grind option
   - Delivery frequency
4. Completes subscription checkout
5. First payment is charged
6. First order is created

### Ongoing Subscriptions

Each billing cycle:

1. Stripe charges the customer
2. Freyja receives webhook notification
3. New order is automatically created
4. Order appears in your fulfillment queue
5. You ship the order
6. Customer receives their coffee

### Customer Management

Customers can self-manage through the portal:

- View upcoming deliveries
- Change frequency
- Update payment method
- Pause subscription
- Resume subscription
- Cancel subscription

## Subscription vs One-Time

| Aspect | Subscription | One-Time |
|--------|-------------|----------|
| Payment | Automatic recurring | Single charge |
| Orders | Created each cycle | One order |
| Pricing | Can offer discount | Regular price |
| Management | Customer portal | N/A |

## Stripe Billing

Freyja uses Stripe Billing for subscriptions:

- **Stripe manages**: Payment scheduling, retries, customer portal
- **Freyja manages**: Order creation, fulfillment, product details

This separation ensures reliable payment handling while keeping your order workflow in Freyja.

## Subscription States

| State | Meaning |
|-------|---------|
| Active | Subscription is running, payments processing |
| Paused | Temporarily stopped, no charges |
| Past Due | Payment failed, awaiting retry |
| Cancelled | Customer cancelled, no more charges |
| Expired | Payment failures exhausted retries |

## Order Creation

When Stripe charges a subscription:

1. Stripe sends `invoice.payment_succeeded` webhook
2. Freyja validates the event
3. Creates order with subscription details
4. Links order to subscription
5. Order ready for fulfillment

## Fulfillment

Subscription orders appear in your regular order list:

- Same fulfillment process as one-time orders
- Can filter by subscription orders
- Ship same as any other order

---

Previous: [Subscriptions Overview](index.md) | Next: [Subscription Plans](plans.md)
