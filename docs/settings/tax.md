# Tax Configuration

Set up tax calculation for your store.

## Tax Providers

Freyja supports three tax calculation options:

### None
No tax calculated. Use when:
- You don't collect sales tax
- You include tax in prices
- Your situation doesn't require tax collection

### Stripe Tax
Automatic tax calculation based on address. Use when:
- You sell to multiple states
- You want automatic rate updates
- You need compliance support

### Percentage
Manual state-by-state rates. Use when:
- You only sell in a few states
- You want direct control over rates
- You have simple tax requirements

## Setting Up Tax

1. Go to **Settings > Tax**
2. Select your tax provider
3. Configure provider settings
4. Save changes

## Provider: None

No configuration needed. Tax will be $0 on all orders.

## Provider: Stripe Tax

Stripe Tax automatically:
- Determines if tax is due based on address
- Calculates correct rate
- Handles rate changes

### Requirements
- Stripe account connected
- Stripe Tax enabled in Stripe Dashboard
- Tax registration information entered in Stripe

### Configuration
1. Select **Stripe Tax** as provider
2. Tax calculation happens automatically
3. Rates determined by Stripe

### Tax Registration
You must configure tax registration in Stripe Dashboard:
1. Log into Stripe
2. Go to **Tax > Registrations**
3. Add states where you're registered to collect
4. Stripe only calculates tax for registered states

## Provider: Percentage

Set specific tax rates by state.

### Configuration
1. Select **Percentage** as provider
2. Click **Manage Tax Rates**
3. Add rates for each state
4. Save changes

### Adding Rates

| Field | Description |
|-------|-------------|
| State | Two-letter state code (e.g., WA) |
| Rate | Percentage (e.g., 6.5 for 6.5%) |
| Name | Display name (e.g., "WA Sales Tax") |

### Example Setup

| State | Rate | Name |
|-------|------|------|
| WA | 6.5% | Washington Sales Tax |
| OR | 0% | Oregon (no sales tax) |
| CA | 7.25% | California Sales Tax |

## How Tax is Calculated

At checkout:
1. Customer enters shipping address
2. State is extracted from address
3. Tax rate looked up (based on provider)
4. Tax calculated on taxable items
5. Added to order total

## Tax on Shipping

Whether shipping is taxed depends on state laws:
- Stripe Tax handles this automatically
- For Percentage provider, shipping tax follows product tax

## Changing Tax Providers

You can switch tax providers:
1. Go to Settings > Tax
2. Select new provider
3. Configure as needed
4. Save changes

Changes apply to future orders. Past orders retain their original tax calculations.

## Compliance Note

Tax compliance is your responsibility. Freyja provides tools for calculation, but you should:
- Understand your tax obligations
- Register where required
- File and remit taxes on schedule
- Consult a tax professional if needed

---

Previous: [Settings Overview](index.md) | Next: [Shipping Setup](shipping.md)
