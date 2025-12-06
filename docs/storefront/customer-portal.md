# Customer Portal

Customer self-service features.

## Account Dashboard

After logging in, customers can access:

- Order history
- Saved addresses
- Subscription management
- Payment methods
- Account settings

## Order History

Customers can view:

### Order List
- All past orders
- Order dates and totals
- Order status

### Order Details
- Line items
- Shipping address
- Tracking information
- Order total breakdown

## Address Management

### Saved Addresses
- View all saved addresses
- Add new addresses
- Edit existing addresses
- Set default shipping address

### Using Saved Addresses
- Quick selection at checkout
- Pre-filled shipping forms
- One-click address selection

## Subscription Management

Customers manage subscriptions through Stripe Customer Portal:

### Available Actions
- View subscription details
- Change delivery frequency
- Update payment method
- Pause subscription
- Resume subscription
- Cancel subscription

### Accessing Portal
1. Go to account
2. Click "Manage Subscription"
3. Redirected to Stripe portal
4. Make changes
5. Return to store

## Payment Methods

### Viewing Methods
- See saved payment methods
- View last 4 digits
- See expiration dates

### Managing Methods
- Done through Stripe Customer Portal
- Add new cards
- Remove old cards
- Set default method

## Account Settings

### Profile
- Update name
- Change email
- Update phone number

### Password
- Change password
- Requires current password

### Email Preferences
- Order notifications
- Shipping updates
- (If marketing implemented) Marketing preferences

## Security

### Password Requirements
- Minimum length enforced
- Secure storage (bcrypt hashed)
- Password reset via email

### Session Management
- Automatic logout on inactivity
- Secure session handling

## Self-Service Benefits

Customer self-service reduces your support load:

| Task | Customer Can Do |
|------|-----------------|
| Check order status | Yes |
| View tracking | Yes |
| Update address | Yes |
| Change subscription | Yes |
| Update payment | Yes |
| Reset password | Yes |

---

Previous: [Cart & Checkout](checkout.md) | Back to [Storefront](index.md)
