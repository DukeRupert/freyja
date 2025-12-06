# Creating Invoices

Generate and send invoices for wholesale orders.

## Invoice Workflow

```
Orders placed (net terms)
         ↓
Create invoice
         ↓
Review and finalize
         ↓
Send to customer
         ↓
Customer pays
         ↓
Record payment
```

## Creating an Invoice

### From Order
1. Go to **Orders**
2. Find the wholesale order
3. Click **Create Invoice**
4. Review invoice details
5. Send to customer

### Manual Creation
1. Go to **Invoices**
2. Click **Create Invoice**
3. Select customer
4. Add line items
5. Set due date
6. Save and send

## Invoice Details

Each invoice includes:

| Field | Description |
|-------|-------------|
| Invoice number | Unique identifier |
| Customer | Who's being billed |
| Invoice date | When created |
| Due date | Payment deadline |
| Line items | Products and quantities |
| Subtotal | Before tax |
| Tax | If applicable |
| Total | Amount due |
| Status | Draft, Sent, Paid, etc. |

## Invoice States

| Status | Meaning |
|--------|---------|
| Draft | Being prepared, not sent |
| Sent | Delivered to customer |
| Viewed | Customer opened invoice |
| Paid | Payment received |
| Overdue | Past due date, unpaid |
| Void | Cancelled invoice |

## Sending Invoices

### Automatic Send
When creating invoice:
1. Review details
2. Click **Send Invoice**
3. Customer receives email with link
4. Status changes to "Sent"

### Email Contents
Invoice email includes:
- Invoice summary
- Link to view/pay online
- Due date reminder
- Your business contact info

## Viewing Invoices

### All Invoices
1. Go to **Invoices**
2. See all invoices with status
3. Filter by status, date, customer
4. Click for details

### Invoice Detail
Shows:
- Full invoice information
- Payment history
- Status timeline
- Actions available

## Editing Invoices

### Draft Invoices
Can be fully edited:
- Add/remove line items
- Change amounts
- Update due date

### Sent Invoices
Limited editing:
- Can void and create new
- Cannot change amounts

## Voiding Invoices

If invoice was created in error:
1. Open the invoice
2. Click **Void Invoice**
3. Confirm void
4. Invoice marked as void

Voided invoices remain for records but show $0 due.

## Best Practices

### Timing
- Send invoices promptly after order/delivery
- Don't let orders stack up without invoicing

### Accuracy
- Verify line items and amounts
- Double-check customer details
- Review before sending

### Records
- Keep all invoices (including voided)
- Track payment dates
- Note any special arrangements

---

Previous: [Net Terms](net-terms.md) | Next: [Consolidated Billing](consolidated-billing.md)
