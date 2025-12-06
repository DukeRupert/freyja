# Troubleshooting

Common issues and how to resolve them.

## In This Section

- [Payment Issues](payments.md) - Problems with payments and checkout
- [Order Problems](orders.md) - Order and fulfillment issues
- [Subscription Issues](subscriptions.md) - Subscription-related problems

## Quick Diagnostics

### Customer Can't Complete Checkout
1. Check Stripe connection status
2. Verify test vs live mode
3. Check for address validation errors
4. Review browser console for JavaScript errors

### Orders Not Appearing
1. Verify payment succeeded in Stripe
2. Check webhook logs
3. Confirm order creation in database

### Emails Not Sending
1. Check email provider configuration
2. Verify API key/credentials
3. Check email logs for errors
4. Verify from address is correct

### Wrong Prices Showing
1. Check customer's assigned price list
2. Verify product has price in that list
3. Check visibility settings

## Getting Help

If you can't resolve an issue:

1. Check these troubleshooting guides
2. Review relevant settings
3. Contact support with:
   - Description of the problem
   - Steps to reproduce
   - Any error messages
   - Screenshots if helpful

---

Next: [Payment Issues](payments.md)
