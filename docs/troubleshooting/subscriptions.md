# Subscription Issues

Troubleshooting subscription-related problems.

## Subscription Not Created

### Symptoms
- Customer completed subscription checkout
- No subscription in Freyja

### Diagnosis
1. Check Stripe Dashboard for subscription
2. Check webhook logs
3. Check application logs

### Common Causes

**Webhook failure**
- Subscription events not received
- Check webhook configuration

**Plan mismatch**
- Product/plan configuration issue
- Verify subscription plan setup

## Orders Not Generated

### Symptoms
- Subscription active in Stripe
- No orders being created

### Diagnosis
1. Check subscription status in Freyja
2. Check webhook delivery for `invoice.payment_succeeded`
3. Review application logs

### Solutions

**Webhook not received**
- Check Stripe webhook configuration
- Verify endpoint is correct

**Order creation failed**
- Check logs for error
- May need manual order creation

## Payment Failed

### Symptoms
- Subscription shows "Past Due"
- Customer not charged

### Customer Actions
1. Customer updates payment method
2. Through customer portal link
3. Or through Stripe-hosted page

### Your Actions
- Monitor past due subscriptions
- Reach out to customer if needed
- Stripe will retry automatically

## Customer Can't Cancel

### Symptoms
- Customer wants to cancel
- Can't access portal

### Solutions

**Portal link not working**
- Verify customer portal is enabled in Stripe
- Check portal configuration

**Manual cancellation**
- Cancel in Stripe Dashboard
- Status syncs to Freyja via webhook

## Wrong Frequency

### Symptoms
- Customer receiving too often/infrequently
- Billing not matching expectation

### Solutions

**Customer self-service**
- Customer can change frequency in portal
- Changes apply to next billing

**Admin change**
- Modify in Stripe Dashboard
- Subscription updates sync to Freyja

## Subscription Syncing Issues

### Symptoms
- Stripe shows one status
- Freyja shows different status

### Diagnosis
1. Check webhook delivery logs
2. Look for failed events
3. Compare timestamps

### Resolution

**Resync**
- Manually update status to match Stripe
- Or trigger webhook replay from Stripe

**Fix webhooks**
- Ensure webhooks are configured
- Check for errors in delivery

## Duplicate Subscriptions

### Symptoms
- Customer has multiple subscriptions
- Charged multiple times

### Investigation
1. Review subscriptions in Stripe
2. Check order history
3. Determine which is correct

### Resolution
1. Cancel incorrect subscription(s)
2. Refund duplicate charges
3. Communicate with customer

## Paused Subscription Issues

### Won't Pause
- Check portal configuration
- May need to pause in Stripe Dashboard

### Won't Resume
- Check payment method is valid
- Customer may need to update card
- Resume in Stripe Dashboard if needed

## Product Changes

### Customer Wants Different Product
- Current: Cancel and create new subscription
- Customer manages through portal or you help

### Product Discontinued
- Communicate with subscribers
- Offer alternative or cancel
- Don't leave orphaned subscriptions

---

Previous: [Order Problems](orders.md) | Back to [Troubleshooting](index.md)
