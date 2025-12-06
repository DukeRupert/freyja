# Order Lifecycle

Understanding how orders move through the system.

## Order States

| Status | Description |
|--------|-------------|
| **Pending** | Order placed, awaiting payment |
| **Paid** | Payment received, ready to fulfill |
| **Processing** | Being prepared for shipment |
| **Shipped** | Handed off to carrier |
| **Delivered** | Confirmed delivery |
| **Cancelled** | Order cancelled |
| **Refunded** | Payment refunded |

## Standard Flow

### Retail Orders

```
Customer checkout
       ↓
   [Pending]
       ↓
Payment processed (automatic)
       ↓
    [Paid]
       ↓
You start fulfillment
       ↓
 [Processing]
       ↓
Label created, package shipped
       ↓
   [Shipped]
       ↓
Carrier confirms delivery
       ↓
  [Delivered]
```

### Subscription Orders

```
Subscription renewal triggered
       ↓
Payment processed (automatic)
       ↓
    [Paid]
       ↓
(continues same as retail)
```

### Wholesale Orders (Net Terms)

```
Wholesale customer places order
       ↓
   [Pending]
       ↓
Invoice sent, payment due per terms
       ↓
Customer pays invoice
       ↓
    [Paid]
       ↓
(continues same as retail)
```

## Status Transitions

### Automatic Transitions
- **Pending → Paid**: When payment succeeds (Stripe webhook)
- **Shipped → Delivered**: When carrier confirms (if tracking enabled)

### Manual Transitions
- **Paid → Processing**: When you start preparing the order
- **Processing → Shipped**: When you create a shipping label
- **Any → Cancelled**: When cancelling an order
- **Any → Refunded**: When processing a refund

## Viewing Order History

Each order tracks its status history:

1. Open the order
2. View the **History** section
3. See all status changes with timestamps

## Tips

- Move orders to **Processing** when you start packing
- This helps track what's in progress vs. waiting
- Use status filters to manage your fulfillment queue

---

Previous: [Orders Overview](index.md) | Next: [Fulfillment Workflow](fulfillment.md)
