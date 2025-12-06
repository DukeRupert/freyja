# Email Settings

Configure email delivery for transactional messages.

## Email Providers

### SMTP
Standard email protocol:
- Works with any SMTP server
- Good for development/testing
- Can use Gmail, Mailgun, etc.

### Postmark
Dedicated transactional email service:
- High deliverability
- Fast delivery
- Designed for transactional email

## Setting Up Email

1. Go to **Settings > Email**
2. Select your email provider
3. Enter configuration details
4. Test email delivery
5. Save changes

## Provider: SMTP

### Configuration

| Setting | Description |
|---------|-------------|
| SMTP Host | Server address (e.g., smtp.gmail.com) |
| SMTP Port | Port number (usually 587 or 465) |
| Username | SMTP username |
| Password | SMTP password or app password |
| From Address | Sender email address |
| From Name | Sender name (e.g., "Your Roastery") |

### Common SMTP Servers

| Provider | Host | Port |
|----------|------|------|
| Gmail | smtp.gmail.com | 587 |
| Mailgun | smtp.mailgun.org | 587 |
| SendGrid | smtp.sendgrid.net | 587 |

## Provider: Postmark

### Requirements
- Postmark account ([postmarkapp.com](https://postmarkapp.com))
- Server API token
- Verified sender domain

### Configuration

| Setting | Description |
|---------|-------------|
| API Token | Postmark Server API Token |
| From Address | Verified sender address |
| From Name | Sender name |

### Setting Up Postmark

1. Create Postmark account
2. Add and verify sender signature (domain)
3. Get Server API Token
4. Enter in Freyja
5. Test delivery

## Automated Emails

Freyja sends these transactional emails:

| Email | When Sent |
|-------|-----------|
| Email Verification | Account signup |
| Password Reset | Password reset requested |
| Order Confirmation | Order placed/paid |
| Shipping Confirmation | Order shipped |
| Subscription Welcome | New subscription created |
| Subscription Payment Failed | Payment issue |
| Subscription Cancelled | Subscription ended |

## From Address

Your "from" address should:
- Be a real address you can receive mail at
- Match your domain if possible
- Look professional (e.g., orders@yourroastery.com)

## Testing Email

After configuration:

1. Click **Send Test Email**
2. Check that email arrives
3. Verify it's not in spam
4. Review how it renders

## Email Deliverability

### Best Practices
- Use a dedicated transactional email service (Postmark recommended)
- Set up SPF, DKIM, DMARC records
- Use consistent from address
- Monitor bounce rates

### If Emails Go to Spam
- Check DNS records (SPF, DKIM)
- Use dedicated email service
- Avoid spam trigger words
- Maintain list hygiene

## Development vs Production

### Development
- Use SMTP with Mailhog or similar
- Catch all emails locally
- Test without sending real mail

### Production
- Use Postmark or reliable SMTP
- Monitor delivery rates
- Handle bounces appropriately

---

Previous: [Shipping Setup](shipping.md) | Next: [Payment Settings](payments.md)
