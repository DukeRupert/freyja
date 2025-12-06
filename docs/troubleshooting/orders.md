# Order Problems

Troubleshooting order and fulfillment issues.

## Order Not Created

### Symptoms
- Payment succeeded in Stripe
- No order in Freyja

### Diagnosis
1. Check Stripe Dashboard for payment
2. Check webhook delivery logs
3. Check application logs

### Common Causes

**Webhook failure**
- Webhook didn't reach Freyja
- Check webhook endpoint configuration

**Processing error**
- Order creation failed internally
- Check application logs for errors

### Resolution
- Fix webhook configuration
- Manually create order if needed
- Contact support if persists

## Wrong Items in Order

### Symptoms
- Order shows incorrect products
- Quantities don't match cart

### Investigation
1. Review cart at time of order
2. Check for cart manipulation
3. Verify product configuration

### Resolution
- Cancel/refund incorrect order
- Create correct order manually
- Investigate root cause

## Order Stuck in Status

### Pending Forever
- Check if payment actually succeeded
- Payment may have failed
- May need manual status update

### Processing Forever
- Fulfillment wasn't completed
- Update to shipped when label created

### Shipped But No Tracking
- Tracking may not have synced
- Add tracking manually
- Check carrier for updates

## Address Issues

### Invalid Address
- Customer entered incorrectly
- Contact customer for correction
- Update before shipping

### Address Won't Validate
- Check address components
- May be new construction
- Manual override if verified correct

### Wrong Address Shipped
- Contact carrier to intercept if possible
- Reship if intercepted
- Refund and reship if needed

## Inventory Problems

### Order Created Without Stock
- Inventory not decremented fast enough
- Concurrent order issue
- Contact customer about delay

### Negative Inventory
- More sold than available
- Update inventory count
- May need to backorder

## Shipping Label Issues

### Can't Create Label
- Verify address is valid
- Check carrier availability
- Try different carrier

### Label Created But Wrong
- Void the label
- Create correct label
- Use voided label number for records

### Tracking Not Updating
- Carrier may not have scanned
- Allow 24 hours for updates
- Contact carrier if persists

## Fulfillment Workflow

### Orders Not Showing
- Check status filter
- Paid orders ready for fulfillment
- May be filtered out

### Can't Find Order
- Search by order number
- Search by customer
- Check date range

---

Previous: [Payment Issues](payments.md) | Next: [Subscription Issues](subscriptions.md)
