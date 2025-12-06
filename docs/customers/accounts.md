# Customer Accounts

View and manage customer information.

## Account Creation

Customers create accounts by:

1. Signing up on your storefront
2. Providing email and password
3. Verifying their email address

Email verification is required before customers can log in and place orders.

## Viewing Customer Details

1. Go to **Customers**
2. Click on a customer name
3. View their full profile

### Profile Information

| Field | Description |
|-------|-------------|
| Name | Customer's full name |
| Email | Account email (login) |
| Phone | Contact phone number |
| Created | Account creation date |
| Account Type | Retail or Wholesale |
| Price List | Assigned pricing tier |

### Related Information

- **Addresses**: Saved shipping addresses
- **Orders**: Order history with status
- **Subscriptions**: Active subscriptions
- **Payment Methods**: Saved payment info (managed via Stripe)

## Editing Customer Information

1. Open customer detail
2. Click **Edit**
3. Update fields as needed
4. Save changes

### What You Can Edit

- Name
- Phone number
- Account type (retail ↔ wholesale)
- Assigned price list
- Net terms (for wholesale)

### What You Cannot Edit

- Email address (customer manages this)
- Password (customer manages this)
- Payment methods (managed via Stripe portal)

## Account Types

### Retail Account
- Default for new signups
- Pays at checkout
- Sees retail pricing
- No net terms

### Wholesale Account
- Requires approval
- Can use net terms
- Sees assigned price list pricing
- May have minimum order requirements

## Changing Account Type

To convert a customer to wholesale:

1. Open customer detail
2. Click **Edit**
3. Change **Account Type** to Wholesale
4. Assign a wholesale price list
5. Set net terms if applicable
6. Save changes

Customer will immediately see their new pricing.

## Deactivating Accounts

If you need to prevent a customer from ordering:

1. Open customer detail
2. Click **Edit**
3. Set status to **Inactive**
4. Save changes

Inactive customers cannot place new orders but can still view past orders.

## Customer Communication

Customer email is their primary contact. Use it for:
- Order issues
- Wholesale account discussions
- Service inquiries

Freyja sends automated emails for:
- Email verification
- Password reset
- Order confirmation
- Shipping confirmation
- Subscription notifications

---

Previous: [Customers Overview](index.md) | Next: [Wholesale Applications](wholesale-applications.md)
