# Payment Issues

Troubleshooting payment and checkout problems.

## Payment Declined

### Symptoms
- Customer sees "Payment declined" error
- Card rejected at checkout

### Causes & Solutions

**Insufficient funds**
- Customer needs to use different card
- Check with bank

**Card expired**
- Customer needs to use current card
- Check expiration date

**Incorrect details**
- Verify card number
- Check CVV and expiration
- Ensure billing address matches

**Bank declined**
- Customer should contact bank
- May be fraud prevention
- Try different card

**Test card in live mode**
- Switch to live mode for real cards
- Use test cards only in test mode

## Payment Stuck Processing

### Symptoms
- Checkout spinning
- Payment doesn't complete

### Solutions

1. **Refresh the page** - May complete on refresh
2. **Check Stripe Dashboard** - Payment may have succeeded
3. **Clear browser cache** - Try incognito/private mode
4. **Check network** - Ensure stable connection

## Double Charges

### Symptoms
- Customer charged twice
- Duplicate orders

### Investigation

1. Check Stripe Dashboard for duplicate payments
2. Check orders in Freyja
3. Review webhook logs

### Resolution

- If duplicate orders exist, refund one
- If payment but no order, check webhooks

## Webhooks Not Working

### Symptoms
- Payments succeed but orders not created
- Stripe events not received

### Diagnosis

1. Go to Stripe Dashboard > Webhooks
2. Check webhook endpoint status
3. Review recent deliveries
4. Look for failed events

### Solutions

**Webhook endpoint wrong**
- Verify endpoint URL in Stripe
- Ensure matches Freyja configuration

**Signature verification failed**
- Check webhook signing secret
- Regenerate if needed

**Server errors**
- Check application logs
- Verify Freyja is running

## Refund Issues

### Can't Process Refund
- Refund through Stripe Dashboard directly
- Check payment is refundable (not disputed)

### Partial Refund
- Calculate amount manually
- Process through Stripe Dashboard

## Stripe Not Connected

### Symptoms
- "Connect Stripe" still showing
- Payments not processing

### Solution
1. Go to Settings > Payments
2. Click Connect Stripe
3. Complete authorization
4. Verify connection status

## Test Mode vs Live Mode

### Symptoms
- Test cards work but real cards don't
- Or vice versa

### Solution
- Match Stripe mode to intended environment
- Use test mode for testing
- Use live mode for real customers
- Check mode in Stripe Dashboard header

---

Previous: [Troubleshooting](index.md) | Next: [Order Problems](orders.md)
