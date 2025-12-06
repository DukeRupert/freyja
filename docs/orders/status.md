# Order Status

Track and update order status throughout fulfillment.

## Status Overview

| Status | Meaning | Action Required |
|--------|---------|-----------------|
| Pending | Awaiting payment | Wait for payment |
| Paid | Payment complete | Ready to fulfill |
| Processing | Being prepared | Complete packing |
| Shipped | In transit | Monitor delivery |
| Delivered | Customer received | Complete |
| Cancelled | Order cancelled | None |
| Refunded | Payment returned | None |

## Viewing Status

### Order List
The Orders page shows status for all orders with color-coded badges.

### Order Detail
Each order shows:
- Current status prominently displayed
- Status history with timestamps
- Available status actions

## Updating Status

### Manual Update
1. Open the order
2. Click the status dropdown or **Update Status**
3. Select new status
4. Confirm the change

### Automatic Updates
Some status changes happen automatically:
- **Paid**: When Stripe confirms payment
- **Shipped**: When shipping label is created
- **Delivered**: When carrier confirms (if tracking integrated)

## Filtering Orders

Use status filters to manage your workflow:

1. Go to **Orders**
2. Click the status filter
3. Select one or more statuses
4. View filtered list

### Suggested Filters

| Task | Filter |
|------|--------|
| Ready to ship | Paid |
| In progress | Processing |
| Recently shipped | Shipped |
| Needs attention | Pending (old) |

## Status History

Every order maintains a complete history:

- Each status change logged
- Timestamp recorded
- Who made the change (if manual)

View history in the order detail page.

## Notifications

Status changes can trigger notifications:

| Status Change | Notification |
|---------------|-------------|
| Paid | Order confirmation email |
| Shipped | Shipping confirmation with tracking |
| Delivered | (Optional) Delivery confirmation |

## Handling Edge Cases

### Stuck in Pending
If an order stays pending too long:
- Check Stripe for payment status
- Customer may have abandoned checkout
- Payment may have failed

### Partial Fulfillment
If you can only ship part of an order:
- Contact customer about partial shipment
- Consider splitting into multiple shipments
- Refund unfulfillable items

### Returns and Refunds
When processing returns:
- Update status to Refunded
- Process refund through Stripe
- Update inventory if restocking

---

Previous: [Shipping Labels](shipping-labels.md) | Back to [Orders](index.md)
