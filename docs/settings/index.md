# Settings

Platform configuration and integrations.

## In This Section

- [Tax Configuration](tax.md) - Set up tax calculation
- [Shipping Setup](shipping.md) - Configure shipping providers
- [Email Settings](email.md) - Configure email delivery
- [Payment Settings](payments.md) - Stripe and payment configuration

## Overview

Settings control how Freyja handles:

- **Tax** - How taxes are calculated at checkout
- **Shipping** - How shipping rates are determined
- **Email** - How transactional emails are sent
- **Payments** - Stripe integration and payment processing

## Accessing Settings

1. Go to **Settings** in the admin sidebar
2. Select the section to configure
3. Make changes
4. Save

## Provider-Based Configuration

Freyja uses a provider model for external services:

| Service | Available Providers |
|---------|---------------------|
| Tax | None, Stripe Tax, Percentage |
| Shipping | Flat Rate, EasyPost |
| Email | SMTP, Postmark |
| Payments | Stripe (required) |

Each provider has specific configuration requirements.

## Initial Setup

When setting up Freyja, configure in this order:

1. **Payments** - Connect Stripe first
2. **Email** - Set up email delivery
3. **Tax** - Configure tax calculation
4. **Shipping** - Set up shipping rates

## Testing Configuration

After configuring:

1. Place a test order
2. Verify tax calculates correctly
3. Check shipping rates
4. Confirm emails are delivered
5. Verify payment processes

---

Next: [Tax Configuration](tax.md)
