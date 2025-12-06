# Cart & Checkout

The customer purchase flow.

## Shopping Cart

### Adding Items
1. Customer selects product
2. Chooses size and grind
3. Clicks "Add to Cart"
4. Cart updates with new item

### Cart Features
- View all cart items
- Update quantities
- Remove items
- See subtotal
- Proceed to checkout

### Cart Persistence
- Guest carts saved in session
- Logged-in carts saved to account
- Cart restored on return visit

## Checkout Flow

Freyja uses a step-by-step checkout:

### Step 1: Contact
- Email address
- Phone number (optional)
- Creates account or logs in

### Step 2: Shipping Address
- Full shipping address
- Address validation (if EasyPost configured)
- Save to address book option

### Step 3: Shipping Method
- View available shipping options
- See rates and delivery estimates
- Select preferred method

### Step 4: Billing
- Same as shipping, or
- Enter separate billing address

### Step 5: Payment
- Enter card details (Stripe Elements)
- Review order summary
- Place order

## Order Confirmation

After successful payment:
- Confirmation page displayed
- Order number shown
- Confirmation email sent
- Order created in system

## Guest vs Account Checkout

### Guest Checkout
- No account required
- Email captured for order updates
- Can create account after checkout

### Account Checkout
- Saved addresses available
- Saved payment methods (via Stripe)
- Order history accessible

## Subscription Checkout

For subscription products:
- Separate subscription checkout flow
- Frequency selection
- Recurring payment setup
- Subscription confirmation

## Address Validation

If EasyPost is configured:
- Addresses validated in real-time
- Suggestions offered for corrections
- Invalid addresses flagged

## Payment Security

- Card details never touch your server
- Stripe handles PCI compliance
- Secure payment processing
- 3D Secure support for authentication

## Common Issues

### Cart Empty After Login
- Guest cart merges with account cart
- Check if items were already in account cart

### Address Rejected
- Check formatting
- Verify apartment/unit number
- Ensure postal code matches city/state

### Payment Declined
- Verify card details
- Check with bank
- Try alternative payment method

---

Previous: [Storefront Overview](overview.md) | Next: [Customer Portal](customer-portal.md)
