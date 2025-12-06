# Subscriptions

Recurring orders and subscription management for your customers.

## In This Section

- [How Subscriptions Work](overview.md) - Understanding the subscription system
- [Subscription Plans](plans.md) - Setting up subscription offerings
- [Managing Subscriptions](management.md) - View and manage active subscriptions
- [Failed Payments](failed-payments.md) - Handling payment issues

## Overview

Subscriptions let customers receive regular coffee deliveries automatically:

- Customer chooses product and frequency
- Payment is charged automatically each cycle
- Orders are created and fulfilled like regular orders
- Customers manage their subscription via portal

## Benefits for Your Business

- **Recurring revenue** - Predictable monthly income
- **Customer retention** - Subscribers stay longer
- **Inventory planning** - Know upcoming demand
- **Reduced friction** - Customers don't need to reorder

## How It Works

```
Customer subscribes
        ↓
Stripe creates subscription
        ↓
Each billing cycle:
    → Stripe charges customer
    → Order created in Freyja
    → You fulfill the order
    → Customer receives coffee
        ↓
Repeat until cancelled
```

## Frequency Options

Customers can choose delivery frequency:

| Frequency | Billing Cycle |
|-----------|---------------|
| Weekly | Every 7 days |
| Every 2 weeks | Every 14 days |
| Monthly | Every month |
| Every 6 weeks | Every 42 days |
| Every 2 months | Every 60 days |

## Customer Portal

Customers manage their subscriptions through a self-service portal:

- View subscription details
- Change frequency
- Update payment method
- Pause or cancel subscription

---

Next: [How Subscriptions Work](overview.md)
