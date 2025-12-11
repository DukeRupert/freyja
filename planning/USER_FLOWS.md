# User Signup & Authentication Flows

This document describes the intended onboarding and authentication flows for all user types in the Hiri platform.

## User Types

| User Type | Description | Entry Point |
|-----------|-------------|-------------|
| **Operator** | Coffee roaster who runs a store on Hiri | `hiri.coffee/pricing` |
| **Retail Customer** | Consumer buying from a roaster's store | `{tenant}.hiri.coffee/signup` |
| **Wholesale Customer** | Café/restaurant with B2B account | `{tenant}.hiri.coffee/wholesale/apply` |
| **Guest** | One-time purchaser (no account) | `{tenant}.hiri.coffee/checkout` |

---

## 1. Tenant Signup Flow (Operator Onboarding)

A coffee roaster signs up to use Hiri as their e-commerce platform.

### Production Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TENANT ONBOARDING                                 │
│                     (Coffee roaster signs up for Hiri)                      │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐
    │  Marketing   │
    │    Site      │
    │ (port 3001)  │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │   /pricing   │ ───► │    Stripe    │ ───► │   /setup/    │
    │              │      │   Checkout   │      │   success    │
    │ "Start Your  │      │              │      │              │
    │   Store"     │      │ • Payment    │      │ "Check your  │
    │   button     │      │ • Biz name   │      │   email"     │
    └──────────────┘      └──────┬───────┘      └──────────────┘
                                 │
                                 │ Webhook: checkout.session.completed
                                 ▼
                          ┌──────────────┐
                          │   Backend    │
                          │              │
                          │ • Create     │
                          │   tenant     │
                          │   (pending)  │
                          │ • Create     │
                          │   operator   │
                          │   (pending)  │
                          │ • Generate   │
                          │   setup      │
                          │   token      │
                          │ • Queue      │
                          │   welcome    │
                          │   email      │
                          └──────┬───────┘
                                 │
                                 ▼
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │   Email      │      │   /setup     │      │    /admin    │
    │   Inbox      │ ───► │  ?token=xxx  │ ───► │  Dashboard   │
    │              │      │              │      │              │
    │ "Complete    │      │ • Set        │      │ • Products   │
    │  your setup" │      │   password   │      │ • Orders     │
    │   link       │      │ • Activate   │      │ • Customers  │
    │              │      │   account    │      │ • Settings   │
    └──────────────┘      └──────────────┘      └──────────────┘
```

### Steps

1. **Visit pricing page** (`/pricing`)
2. **Click "Start Your Store"** → POST to `/api/saas/checkout`
3. **Stripe Checkout** → Enter payment info and business name
4. **Redirect to success page** (`/setup/success`)
5. **Webhook fires** (`checkout.session.completed`)
   - Creates tenant (status: `pending`)
   - Creates operator (status: `pending`)
   - Generates setup token (48h expiry)
   - Queues welcome email
6. **Receive email** with setup link
7. **Complete setup** (`/setup?token=xxx`) → Set password
8. **Redirected to admin** (`/admin`) → Logged in

### Returning Operator

```
    ┌──────────────┐      ┌──────────────┐
    │   /login     │ ───► │    /admin    │
    │              │      │  Dashboard   │
    │ Email +      │      │              │
    │ Password     │      │              │
    └──────────────┘      └──────────────┘
```

### Development Bypass

For local development without Stripe:

```
    ┌──────────────┐      ┌──────────────┐
    │ /dev/signup  │ ───► │    /admin    │
    │              │      │  Dashboard   │
    │ • Biz name   │      │              │
    │ • Email      │      │ (logged in   │
    │ • Password   │      │  immediately)│
    └──────────────┘      └──────────────┘
```

**Skips:** Stripe checkout, email verification, setup token
**Creates:** Active tenant + active operator + session cookie

See [docs/dev-tenant-setup.md](/docs/dev-tenant-setup.md) for usage instructions.

---

## 2. Retail Customer Signup Flow

A consumer creates an account on a roaster's storefront.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CUSTOMER ONBOARDING                                │
│              (Consumer signs up on a roaster's storefront)                  │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐
    │  Storefront  │
    │  (port 3000) │
    │              │
    │ acme.hiri.co │
    │     or       │
    │ shop.acme.com│
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │   /signup    │ ───► │   Backend    │ ───► │   Email      │
    │              │      │              │      │   Inbox      │
    │ • Email      │      │ • Create     │      │              │
    │ • Password   │      │   customer   │      │ "Verify your │
    │ • Name       │      │   (pending)  │      │   email"     │
    │              │      │ • Generate   │      │   link       │
    │              │      │   verify     │      │              │
    │              │      │   token      │      │              │
    └──────────────┘      └──────────────┘      └──────┬───────┘
                                                       │
                                                       ▼
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │   /account   │ ◄─── │   /login     │ ◄─── │   /verify    │
    │              │      │              │      │  ?token=xxx  │
    │ • Orders     │      │ Email +      │      │              │
    │ • Addresses  │      │ Password     │      │ • Activate   │
    │ • Subscrip-  │      │              │      │   account    │
    │   tions      │      │              │      │ • Redirect   │
    │ • Payment    │      │              │      │   to login   │
    │   methods    │      │              │      │              │
    └──────────────┘      └──────────────┘      └──────────────┘
```

### Steps

1. **Visit signup page** (`/signup`)
2. **Enter details** → Email, password, name
3. **Account created** (status: `pending`)
4. **Receive verification email**
5. **Click verification link** (`/verify?token=xxx`)
6. **Account activated** → Redirect to login
7. **Login** (`/login`)
8. **Access account dashboard** (`/account`)

### Account Dashboard Features

- Order history
- Saved addresses
- Subscription management (pause/skip/cancel)
- Payment methods
- Profile settings

---

## 3. Guest Checkout Flow

One-time purchase without creating an account.

```
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │    /cart     │ ───► │  /checkout   │ ───► │ Order email  │
    │              │      │              │      │              │
    │              │      │ • Email      │      │ Confirmation │
    │              │      │ • Shipping   │      │ + tracking   │
    │              │      │ • Payment    │      │              │
    └──────────────┘      └──────────────┘      └──────────────┘
```

### Steps

1. **Add items to cart** (`/cart`)
2. **Proceed to checkout** (`/checkout`)
3. **Enter shipping & payment** (no account required)
4. **Complete purchase**
5. **Receive order confirmation email**

Guest orders are associated with email only. Customer can create an account later to view order history.

---

## 4. Wholesale Customer Flow (B2B)

A café or restaurant applies for a wholesale account.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        WHOLESALE ONBOARDING                                 │
│                (Café/restaurant applies for wholesale)                      │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │  /wholesale  │ ───► │   Backend    │ ───► │   Email      │
    │   /apply     │      │              │      │  (to tenant) │
    │              │      │ • Create     │      │              │
    │ • Biz name   │      │   wholesale  │      │ "New         │
    │ • Tax ID     │      │   account    │      │  wholesale   │
    │ • Contact    │      │   (pending   │      │  application"│
    │ • Est. vol.  │      │   approval)  │      │              │
    └──────────────┘      └──────────────┘      └──────┬───────┘
                                                       │
                                                       ▼
    ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
    │  Wholesale   │ ◄─── │   Email      │ ◄─── │    /admin    │
    │   Portal     │      │ (to café)    │      │  /customers  │
    │              │      │              │      │              │
    │ • Net terms  │      │ "You've been │      │ Tenant       │
    │   (Net 30)   │      │  approved!"  │      │ reviews &    │
    │ • Tier       │      │              │      │ approves     │
    │   pricing    │      │ Setup link   │      │              │
    │ • Minimums   │      │              │      │ • Set tier   │
    │ • Invoices   │      │              │      │ • Set terms  │
    └──────────────┘      └──────────────┘      └──────────────┘
```

### Steps

1. **Submit application** (`/wholesale/apply`)
   - Business name
   - Tax ID
   - Contact information
   - Estimated monthly volume
2. **Application created** (status: `pending_approval`)
3. **Tenant notified** via email
4. **Tenant reviews** in admin (`/admin/customers`)
   - Approve or reject
   - Set pricing tier
   - Set payment terms (Net 15/30)
   - Set minimum order quantity
5. **Applicant notified** of approval
6. **Complete account setup** via email link
7. **Access wholesale portal**

### Wholesale Portal Features

- Tier-specific pricing
- Net terms ordering (pay later)
- Minimum order enforcement
- Consolidated monthly invoices
- Order history
- Account statements

---

## Authentication URLs

### SaaS Marketing Site (port 3001 / hiri.coffee)

| URL | Purpose |
|-----|---------|
| `/login` | Operator sign in |
| `/forgot-password` | Request password reset |
| `/reset-password?token=xxx` | Set new password |
| `/setup?token=xxx` | Complete account setup |
| `/resend-setup` | Resend setup email |
| `/dev/signup` | Dev bypass (development only) |

### Tenant Storefront (port 3000 / {tenant}.hiri.coffee)

| URL | Purpose |
|-----|---------|
| `/signup` | Customer registration |
| `/login` | Customer sign in |
| `/logout` | Sign out |
| `/forgot-password` | Request password reset |
| `/reset-password?token=xxx` | Set new password |
| `/verify?token=xxx` | Email verification |
| `/account` | Account dashboard |

### Admin Dashboard (port 3000 / app.hiri.coffee)

| URL | Purpose |
|-----|---------|
| `/admin/login` | Admin sign in |
| `/admin/logout` | Admin sign out |
| `/admin` | Dashboard |

---

## Port Reference (Development)

| Port | Production Domain | Purpose |
|------|-------------------|---------|
| 3001 | `hiri.coffee` | SaaS marketing site + operator auth |
| 3000 | `app.hiri.coffee` | Main app (admin, API, webhooks) |
| 3000 | `{slug}.hiri.coffee` | Tenant storefronts |

---

## Database Entities

### Tenant Onboarding

| Entity | Table | Key Fields |
|--------|-------|------------|
| Tenant | `tenants` | id, name, slug, status, stripe_customer_id |
| Operator | `tenant_operators` | id, tenant_id, email, role, status |
| Session | `operator_sessions` | id, operator_id, token_hash, expires_at |

### Customer Onboarding

| Entity | Table | Key Fields |
|--------|-------|------------|
| Customer | `customers` | id, tenant_id, email, status |
| Session | `customer_sessions` | id, customer_id, token_hash |
| Wholesale Account | `wholesale_accounts` | id, customer_id, tier, payment_terms |

---

## Status Values

### Tenant Status

| Status | Description |
|--------|-------------|
| `pending` | Payment received, awaiting subscription confirmation |
| `active` | Subscription active, full access |
| `past_due` | Payment failed, in grace period |
| `suspended` | Grace period expired, access restricted |
| `cancelled` | Subscription cancelled |

### Operator Status

| Status | Description |
|--------|-------------|
| `pending` | Created, awaiting password setup |
| `active` | Password set, can log in |
| `suspended` | Access revoked |

### Customer Status

| Status | Description |
|--------|-------------|
| `pending` | Registered, awaiting email verification |
| `active` | Email verified, can log in |
| `suspended` | Access revoked |

### Wholesale Account Status

| Status | Description |
|--------|-------------|
| `pending_approval` | Application submitted |
| `approved` | Approved, can place orders |
| `rejected` | Application denied |
| `suspended` | Account suspended |
