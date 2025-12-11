# SaaS Deployment Configuration

This document covers the environment variables and Stripe setup required to deploy the SaaS tenant signup flow in production.

## Environment Variables

### Stripe SaaS Subscription

| Variable | Required | Description |
|----------|----------|-------------|
| `STRIPE_SAAS_PRICE_ID` | Yes (production) | Stripe Price ID for the $149/month subscription |
| `STRIPE_SAAS_WEBHOOK_SECRET` | Yes (production) | Webhook signing secret for `/webhooks/stripe/saas` |

### Domain Configuration (Multi-Tenant Routing)

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `HOST_ROUTING_ENABLED` | Yes (production) | Enable subdomain-based tenant routing | `true` |
| `BASE_DOMAIN` | Yes (production) | Root domain for subdomains | `hiri.coffee` |
| `APP_DOMAIN` | Yes (production) | Domain for admin dashboard and operator auth | `app.hiri.coffee` |

### Example Production Configuration

```bash
# Domain routing
HOST_ROUTING_ENABLED=true
BASE_DOMAIN=hiri.coffee
APP_DOMAIN=app.hiri.coffee

# Stripe SaaS
STRIPE_SAAS_PRICE_ID=price_1ABC123XYZ
STRIPE_SAAS_WEBHOOK_SECRET=whsec_abc123xyz
```

### Example Development Configuration

```bash
# Domain routing (disabled - single tenant mode)
HOST_ROUTING_ENABLED=false
BASE_DOMAIN=
APP_DOMAIN=

# Stripe SaaS (not needed if using /dev/signup bypass)
STRIPE_SAAS_PRICE_ID=
STRIPE_SAAS_WEBHOOK_SECRET=
```

---

## Stripe Setup

### 1. Create a Product

1. Go to [Stripe Dashboard → Products](https://dashboard.stripe.com/products)
2. Click **Add Product**
3. Fill in:
   - **Name:** Hiri Platform
   - **Description:** E-commerce platform for coffee roasters
4. Save the product

### 2. Create a Price

1. On the product page, click **Add Price**
2. Configure:
   - **Pricing model:** Standard pricing
   - **Price:** $149.00
   - **Billing period:** Monthly
   - **Usage type:** Licensed
3. Save the price
4. **Copy the Price ID** (starts with `price_`)
5. Set `STRIPE_SAAS_PRICE_ID=price_xxx` in your environment

### 3. Create a Webhook Endpoint

1. Go to [Stripe Dashboard → Webhooks](https://dashboard.stripe.com/webhooks)
2. Click **Add endpoint**
3. Configure:
   - **Endpoint URL:** `https://app.hiri.coffee/webhooks/stripe/saas`
   - **Description:** Hiri SaaS subscription events
4. Select events to listen for:
   - `checkout.session.completed`
   - `invoice.paid`
   - `invoice.payment_failed`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
5. Click **Add endpoint**
6. **Copy the Signing secret** (starts with `whsec_`)
7. Set `STRIPE_SAAS_WEBHOOK_SECRET=whsec_xxx` in your environment

### 4. (Optional) Create a Promotion Code

For the launch promotion ($5/month for 3 months):

1. Go to [Stripe Dashboard → Coupons](https://dashboard.stripe.com/coupons)
2. Click **Create coupon**
3. Configure:
   - **Type:** Fixed amount discount
   - **Amount off:** $144.00 (reduces $149 to $5)
   - **Duration:** Repeating (3 months)
4. Create a promotion code for the coupon
5. Users can enter this code during checkout

---

## Domain Architecture

### Production URLs

| Domain | Purpose | Port (dev) |
|--------|---------|------------|
| `hiri.coffee` | Marketing site (landing, pricing, about) | 3001 |
| `app.hiri.coffee` | Admin dashboard, operator auth, API, webhooks | 3000 |
| `{slug}.hiri.coffee` | Tenant storefronts | 3000 |
| Custom domains | Tenant storefronts (e.g., `shop.acme.com`) | 3000 |

### Cookie Domain Scoping

When `BASE_DOMAIN` is set, session cookies are scoped to `.hiri.coffee` to allow sharing across:
- `hiri.coffee` (marketing)
- `app.hiri.coffee` (admin)
- `{tenant}.hiri.coffee` (storefronts)

This enables single sign-on across subdomains.

---

## Development vs Production

### Development (Local)

```
ENV=dev
HOST_ROUTING_ENABLED=false
```

- Use `/dev/signup` to bypass Stripe and create tenants instantly
- No Stripe configuration required
- Cookies are host-only (no domain scoping)
- Both servers run on localhost (ports 3000 and 3001)

### Production

```
ENV=prod
HOST_ROUTING_ENABLED=true
BASE_DOMAIN=hiri.coffee
APP_DOMAIN=app.hiri.coffee
```

- Full Stripe checkout flow
- Webhook-driven tenant creation
- Email verification for account setup
- Cookies scoped to `.hiri.coffee`
- Caddy or similar for TLS termination and routing

---

## Webhook Event Handling

| Event | Handler Action |
|-------|----------------|
| `checkout.session.completed` | Create tenant (pending), create operator (pending), generate setup token, queue welcome email |
| `invoice.paid` | Clear grace period if tenant was past_due, activate tenant |
| `invoice.payment_failed` | Start 7-day grace period, queue payment failed email |
| `customer.subscription.updated` | Sync subscription status to tenant |
| `customer.subscription.deleted` | Set tenant status to cancelled, queue cancellation email |

---

## Checklist

### Before Deploying SaaS Signup

- [ ] Create Stripe Product and Price
- [ ] Set `STRIPE_SAAS_PRICE_ID` environment variable
- [ ] Create Stripe Webhook endpoint for `/webhooks/stripe/saas`
- [ ] Set `STRIPE_SAAS_WEBHOOK_SECRET` environment variable
- [ ] Configure domain routing (`HOST_ROUTING_ENABLED`, `BASE_DOMAIN`, `APP_DOMAIN`)
- [ ] Set up DNS records for `app.hiri.coffee`
- [ ] Configure TLS certificates
- [ ] Test webhook delivery in Stripe Dashboard
- [ ] Create email templates for:
  - [ ] Welcome email (with setup link)
  - [ ] Payment failed email
  - [ ] Subscription cancelled email
  - [ ] Account suspended email

### Testing Webhook Locally

Use Stripe CLI to forward webhooks to your local server:

```bash
# Install Stripe CLI
brew install stripe/stripe-cli/stripe

# Login to Stripe
stripe login

# Forward webhooks to local server
stripe listen --forward-to localhost:3000/webhooks/stripe/saas

# The CLI will display a webhook signing secret - use this for local testing
# STRIPE_SAAS_WEBHOOK_SECRET=whsec_xxx
```

In another terminal, trigger test events:

```bash
stripe trigger checkout.session.completed
stripe trigger invoice.paid
stripe trigger invoice.payment_failed
```
