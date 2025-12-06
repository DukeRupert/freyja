# Consolidated Billing

Combine multiple orders into single invoices.

## What is Consolidated Billing?

Instead of invoicing each order separately, consolidated billing:

- Accumulates orders over a billing period
- Generates one invoice for all orders
- Sends a single bill to the customer

## Benefits

### For You
- Less invoice administration
- Fewer payment transactions to track
- Cleaner bookkeeping

### For Customers
- Single payment per billing cycle
- Easier to manage
- Consolidated record of purchases

## How It Works

```
Week 1: Customer places Order A
Week 2: Customer places Order B
Week 3: Customer places Order C
End of month: All orders invoiced together
              → One invoice for Orders A + B + C
```

## Setting Up Consolidated Billing

### Per Customer
1. Go to **Customers**
2. Open wholesale customer
3. Click **Edit**
4. Enable **Consolidated Billing**
5. Set billing cycle (e.g., monthly)
6. Save changes

### Billing Cycles

| Cycle | Invoice Generated |
|-------|------------------|
| Weekly | Every Sunday |
| Bi-weekly | Every other Sunday |
| Monthly | First of month |
| Custom | Specific date |

## Order Flow

With consolidated billing:

1. Customer places order
2. Order fulfilled normally
3. Order marked as "Awaiting Invoice"
4. At billing cycle end:
   - All "Awaiting Invoice" orders collected
   - Single invoice generated
   - Invoice sent to customer

## Invoice Contents

Consolidated invoices show:

- All orders in the period
- Line items per order
- Order dates and numbers
- Subtotal per order
- Grand total
- Due date

## Manual Consolidation

If you want to consolidate manually:

1. Go to **Invoices**
2. Click **Create Invoice**
3. Select customer
4. Choose **Select Orders**
5. Check orders to include
6. Generate invoice

## When to Use Consolidated Billing

### Good Candidates
- High-frequency orderers
- Established relationships
- Customers who request it
- Standard billing cycle businesses

### Not Recommended
- Infrequent orderers
- New accounts
- Customers with payment issues
- One-time orders

## Managing the Cycle

### Before Cycle End
- Review pending orders
- Verify all orders are fulfilled
- Check for any issues

### At Cycle End
- Invoices generate automatically (if configured)
- Or manually trigger consolidation
- Review before sending

### After Sending
- Monitor for payment
- Follow up if overdue
- Update for next cycle

---

Previous: [Creating Invoices](creating-invoices.md) | Next: [Recording Payments](payments.md)
