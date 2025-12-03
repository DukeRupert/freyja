# SaaS Customer Onboarding

This document defines the onboarding flow for new Freyja SaaS customers (coffee roasters signing up for the platform).

**Last updated:** December 3, 2024

---

## Business Decisions

### Pricing Strategy

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Base price | $149/month | Undercuts Shopify + plugins by $50-200/month |
| Introductory offer | $5/month for first 3 months | Low barrier to entry, validates commitment without free trial complexity |
| Billing cycle | Monthly only (MVP) | Reduces complexity; annual option deferred |
| Transaction fees | None (Stripe fees only) | Clear value proposition, predictable costs |

**Stripe Configuration:**
- Product: "Freyja Platform"
- Price: $149/month recurring
- Coupon: $144 off for 3 months (applied automatically at checkout)
- Result: $5 → $5 → $5 → $149 → $149...

### Payment Failure Handling

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Grace period | 7 days | Balances customer experience with revenue protection |
| User notification | Banner + email | Clear communication without being aggressive |
| Access during grace | Full access | Maintains goodwill, most failures are accidental |
| After grace period | Suspended (blocked) | Must protect revenue |
| Recovery | Automatic on payment | No manual intervention needed |

### Authentication

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary method | Email + password | Expected by B2B users, familiar pattern |
| Initial access | Password setup via email link | Secure, no temporary passwords |
| Token expiry | 48 hours | Reasonable window without security risk |
| Session duration | 7 days | Balance convenience and security |
| OAuth | Deferred | Not critical for coffee roaster audience |
| MFA | Deferred | Low-risk data, can add later if requested |

### Tenant Identification

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Slug generation | Auto from business name | Simple, no decision fatigue |
| Slug format | lowercase-hyphenated | URL-safe, readable |
| Duplicates | Append number (-2, -3) | Automatic resolution |
| Routing (MVP) | Path-based (/app/*) | Simpler than subdomain routing |
| Subdomains | Deferred | Can add {slug}.freyja.app later |

### Data Retention

| Decision | Choice | Rationale |
|----------|--------|-----------|
| After cancellation | 90 days soft-delete | Generous window for reactivation |
| Reactivation method | New checkout flow | Simpler than resurrection; data restored automatically |
| After 90 days | Hard delete or archive | Compliance; reduces storage costs |
| Backups | Standard retention | Follow normal backup policy |

### Multi-User Access

| Decision | Choice | Rationale |
|----------|--------|-----------|
| MVP scope | Single user per tenant | Most small roasters are 1-2 person operations |
| User type | Owner (full access) | No role complexity for MVP |
| Team invites | Deferred | Add when customers request it |

---

## Architecture

### User Type Distinction

The platform has two distinct user types that must not be confused:

```
┌─────────────────────────────────────────────────────────────────┐
│                         FREYJA PLATFORM                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  OPERATORS (tenant_operators table)                             │
│  ├── People who PAY for Freyja ($149/month)                    │
│  ├── Log into admin dashboard                                   │
│  ├── Manage products, orders, customers                         │
│  └── One per tenant for MVP                                     │
│                                                                 │
│  CUSTOMERS (users table - existing)                             │
│  ├── People who BUY coffee from a roaster                       │
│  ├── Log into storefront                                        │
│  ├── Place orders, manage subscriptions                         │
│  └── Many per tenant                                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Why separate tables?**
- Different authentication flows (operators via SaaS onboarding, customers via storefront signup)
- Different session scopes (operators access admin, customers access storefront)
- Different lifecycle (operator tied to subscription, customers independent)
- Cleaner security model (no risk of privilege confusion)

### Tenant States

```
                    checkout.session.completed
                              │
                              ▼
┌─────────┐            ┌──────────┐
│ (none)  │───────────►│ pending  │ User hasn't set password yet
└─────────┘            └────┬─────┘
                            │ password set
                            ▼
                       ┌──────────┐
              ┌───────►│  active  │◄────────────────┐
              │        └────┬─────┘                 │
              │             │                       │
              │   invoice.payment_failed            │ invoice.paid
              │             │                       │
              │             ▼                       │
              │        ┌──────────┐                 │
              │        │ past_due │─────────────────┘
              │        └────┬─────┘
              │             │
              │       7 days elapsed
              │             │
              │             ▼
              │       ┌───────────┐
              │       │ suspended │
              │       └─────┬─────┘
              │             │
              │   invoice.paid (manual)
              │             │
              └─────────────┘
                            │
        customer.subscription.deleted
                            │
                            ▼
                      ┌───────────┐
                      │ cancelled │ (terminal, data retained)
                      └───────────┘
```

### Onboarding Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     ONBOARDING SEQUENCE                          │
└─────────────────────────────────────────────────────────────────┘

Step 1: Landing Page
├── User clicks "Get Started" or "Start Your Store"
└── Redirect to Stripe Checkout

Step 2: Stripe Checkout (hosted by Stripe)
├── Collect email
├── Collect business name (custom field)
├── Collect payment method
├── Apply $144-off coupon automatically
└── Process $5 payment

Step 3: Webhook Processing (checkout.session.completed)
├── Extract email, business name from session
├── Generate unique slug from business name
├── Create tenant record (status: pending)
├── Create operator record (password_hash: NULL)
├── Generate password setup token (48h expiry)
└── Send welcome email with setup link

Step 4: Welcome Email
├── Subject: "Welcome to Freyja - Set up your account"
├── Contains: Setup link with token
└── CTA: "Set Your Password"

Step 5: Account Setup Page (/setup?token=xxx)
├── Validate token (exists, not expired, not used)
├── Show password form
├── On submit: hash password, clear token, set tenant active
└── Create session, redirect to dashboard

Step 6: Dashboard
├── User lands on /admin/dashboard
├── Guided setup prompts (add products, configure shipping, etc.)
└── Full platform access
```

---

## Database Schema Changes

### New Table: tenant_operators

```sql
-- Operators: people who manage a tenant (roaster staff)
CREATE TABLE tenant_operators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Authentication
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),  -- NULL until setup complete

    -- Profile
    name VARCHAR(255),

    -- Role (for future multi-user support)
    role VARCHAR(50) NOT NULL DEFAULT 'owner',
    -- owner: full access, billing management
    -- admin: full access except billing (future)
    -- staff: limited access (future)

    -- Setup/reset tokens
    setup_token VARCHAR(255) UNIQUE,
    setup_token_expires_at TIMESTAMPTZ,
    reset_token VARCHAR(255) UNIQUE,
    reset_token_expires_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending: invited, hasn't set password
    -- active: can log in
    -- suspended: access revoked

    last_login_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_tenant_operators_tenant_id ON tenant_operators(tenant_id);
CREATE INDEX idx_tenant_operators_email ON tenant_operators(email);
CREATE INDEX idx_tenant_operators_setup_token ON tenant_operators(setup_token)
    WHERE setup_token IS NOT NULL;
CREATE INDEX idx_tenant_operators_reset_token ON tenant_operators(reset_token)
    WHERE reset_token IS NOT NULL;
```

### New Table: operator_sessions

```sql
-- Sessions for tenant operators (separate from customer sessions)
CREATE TABLE operator_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id UUID NOT NULL REFERENCES tenant_operators(id) ON DELETE CASCADE,

    token_hash VARCHAR(255) NOT NULL,  -- SHA-256 of session token

    -- Session metadata
    user_agent TEXT,
    ip_address INET,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operator_sessions_token_hash ON operator_sessions(token_hash);
CREATE INDEX idx_operator_sessions_operator_id ON operator_sessions(operator_id);
CREATE INDEX idx_operator_sessions_expires_at ON operator_sessions(expires_at);
```

### Tenants Table Additions

```sql
-- Add Stripe integration and grace period columns to tenants
ALTER TABLE tenants
    ADD COLUMN stripe_customer_id VARCHAR(255) UNIQUE,
    ADD COLUMN stripe_subscription_id VARCHAR(255) UNIQUE,
    ADD COLUMN grace_period_started_at TIMESTAMPTZ;

-- Update status check constraint to include new states
ALTER TABLE tenants
    DROP CONSTRAINT tenants_status_check,
    ADD CONSTRAINT tenants_status_check
        CHECK (status IN ('pending', 'active', 'past_due', 'suspended', 'cancelled'));

CREATE INDEX idx_tenants_stripe_customer_id ON tenants(stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX idx_tenants_stripe_subscription_id ON tenants(stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;
CREATE INDEX idx_tenants_grace_period ON tenants(grace_period_started_at)
    WHERE status = 'past_due';
```

---

## Stripe Configuration

### One-Time Setup (Manual in Stripe Dashboard or CLI)

```bash
# 1. Create the product
stripe products create \
  --name="Freyja Platform" \
  --description="E-commerce platform for coffee roasters"
# Returns: prod_xxx

# 2. Create the monthly price
stripe prices create \
  --product=prod_xxx \
  --unit-amount=14900 \
  --currency=usd \
  --recurring[interval]=month \
  --lookup-key=freyja_monthly
# Returns: price_xxx

# 3. Create the introductory coupon
stripe coupons create \
  --id=freyja-launch-special \
  --amount-off=14400 \
  --currency=usd \
  --duration=repeating \
  --duration-in-months=3 \
  --name="Launch Special - $5/month for 3 months"
# Returns: freyja-launch-special
```

### Webhook Endpoints

Configure these webhooks in Stripe Dashboard:

| Event | Handler | Action |
|-------|---------|--------|
| `checkout.session.completed` | `/webhooks/stripe/checkout` | Create tenant + operator, send welcome email |
| `invoice.paid` | `/webhooks/stripe/invoice` | Clear grace period if past_due |
| `invoice.payment_failed` | `/webhooks/stripe/invoice` | Start grace period, send email |
| `customer.subscription.deleted` | `/webhooks/stripe/subscription` | Set tenant cancelled |
| `customer.subscription.updated` | `/webhooks/stripe/subscription` | Sync status changes |

---

## Email Templates

### 1. Welcome Email (account setup)

**Subject:** Welcome to Freyja - Set up your account

**Trigger:** checkout.session.completed webhook

```
Hi {{.BusinessName}},

Your Freyja account is ready! Click below to set your password and
start building your online store.

[Set Your Password]
{{.SetupURL}}

This link expires in 48 hours. If you need a new link, visit
freyja.app/resend-setup.

Your subscription:
- $5/month for your first 3 months
- Then $149/month
- Cancel anytime

Questions? Reply to this email.

— The Freyja Team
```

### 2. Password Reset Email

**Subject:** Reset your Freyja password

**Trigger:** POST /forgot-password

```
Hi {{.Name}},

Click below to reset your password:

[Reset Password]
{{.ResetURL}}

This link expires in 1 hour.

If you didn't request this, you can safely ignore this email.

— The Freyja Team
```

### 3. Payment Failed Email

**Subject:** Action needed: Payment failed for your Freyja account

**Trigger:** invoice.payment_failed webhook

```
Hi {{.Name}},

We couldn't process your subscription payment of {{.Amount}}.

Please update your payment method to keep your store running:

[Update Payment Method]
{{.BillingPortalURL}}

Your account will remain active for 7 days while you resolve this.

— The Freyja Team
```

### 4. Account Suspended Email

**Subject:** Your Freyja account has been suspended

**Trigger:** Grace period expiration job

```
Hi {{.Name}},

Your Freyja account has been suspended due to an unpaid balance.

Your store is currently offline. Your data is safe and will remain
available for 90 days.

To reactivate immediately:

[Update Payment & Reactivate]
{{.BillingPortalURL}}

Questions? Reply to this email.

— The Freyja Team
```

---

## Implementation Plan

### Phase 1: Schema & Core Infrastructure

**Migration: Add SaaS onboarding tables**
1. Add Stripe columns to tenants table
2. Create tenant_operators table
3. Create operator_sessions table (with token_hash)
4. Update tenants status constraint
5. Migrate existing `sessions` table: rename `token` → `token_hash` for consistent security

**Slug generation utility**
1. Create `internal/tenant/slug.go`
2. Implement GenerateSlug (name → slug)
3. Implement GenerateUniqueSlug (checks database)
4. Add tests for edge cases

### Phase 2: Stripe Checkout Integration

**Landing page checkout redirect**
1. Create GET `/signup` page (or use landing page CTA)
2. Create POST `/api/create-checkout-session`
3. Configure Stripe Checkout with:
   - Price ID for $149/month
   - Coupon auto-applied
   - Custom field for business name
   - Success/cancel URLs

**Webhook handler for checkout.session.completed**
1. Create `/webhooks/stripe/saas` endpoint (separate from existing)
2. Extract email, business name from session metadata
3. Generate unique slug
4. Create tenant (status: pending)
5. Create operator (password_hash: NULL)
6. Generate setup token
7. Queue welcome email

### Phase 3: Account Setup Flow

**Setup page**
1. Create GET `/setup` - validate token, show form
2. Create POST `/setup` - set password, activate account
3. Password requirements: minimum 8 characters
4. On success: create session, redirect to dashboard

**Resend setup email**
1. Create GET `/resend-setup` - email form
2. Create POST `/resend-setup` - lookup operator by email, regenerate token, send email
3. Rate limit: max 3 requests per email per hour
4. Always show success message (don't leak whether email exists)

**Operator authentication**
1. Create GET `/login` - login form
2. Create POST `/login` - validate credentials, create session
3. Create POST `/logout` - destroy session
4. Session middleware for admin routes

**Password reset**
1. Create GET `/forgot-password` - email form
2. Create POST `/forgot-password` - generate token, send email
3. Create GET `/reset-password` - validate token, show form
4. Create POST `/reset-password` - update password

### Phase 4: Payment Failure Handling

**Webhook handlers**
1. `invoice.payment_failed` - set past_due, record grace start, send email
2. `invoice.paid` - clear past_due if applicable
3. `customer.subscription.deleted` - set cancelled

**Grace period job**
1. Create background job: check for expired grace periods
2. Run hourly (or on-demand in middleware)
3. Set status to suspended
4. Send suspension email

**UI banner**
1. Add past_due check to admin layout
2. Show warning banner with billing portal link
3. Style: amber/warning, not aggressive

**Billing portal access**
1. Add "Billing" link to admin header/nav menu
2. Create settings page with billing section
3. Both link to Stripe Customer Portal via redirect endpoint

### Phase 5: Email Integration

**Templates**
1. Welcome email (setup link)
2. Password reset email
3. Payment failed email
4. Account suspended email

**Integration**
1. Use existing email service infrastructure
2. Add new job types for SaaS emails
3. Ensure operator emails go to operator.email, not tenant.email

---

## File Structure

```
internal/
├── onboarding/
│   ├── service.go          # Tenant + operator creation logic
│   ├── slug.go             # Slug generation utilities
│   └── slug_test.go
├── operator/
│   ├── service.go          # Operator CRUD, password management
│   ├── auth.go             # Login/logout, session management
│   └── middleware.go       # Session validation for admin routes
├── handler/
│   ├── saas/
│   │   ├── checkout.go     # Create checkout session
│   │   ├── setup.go        # Account setup + resend setup flow
│   │   ├── auth.go         # Login/logout/forgot password
│   │   ├── billing.go      # Billing portal redirect
│   │   ├── settings.go     # Account settings page
│   │   └── webhook.go      # SaaS-specific Stripe webhooks
├── jobs/
│   ├── grace_period.go     # Expire grace periods job
│   └── ... (existing)
└── email/
    └── templates/
        ├── saas_welcome.html
        ├── saas_password_reset.html
        ├── saas_payment_failed.html
        └── saas_suspended.html

web/
└── templates/
    └── saas/
        ├── login.html
        ├── setup.html
        ├── resend_setup.html
        ├── forgot_password.html
        ├── reset_password.html
        └── settings.html
```

---

## Testing Strategy

### Unit Tests

- Slug generation (various inputs, unicode, duplicates)
- Token generation and validation
- Password hashing and verification
- Grace period date calculations

### Integration Tests

- Checkout webhook → tenant + operator created
- Setup flow → password set, session created
- Login → session created, redirected
- Payment failed webhook → status updated, email queued
- Grace period expiration → status suspended

### Manual Testing Checklist

- [ ] Complete Stripe Checkout, receive welcome email
- [ ] Click setup link, set password successfully
- [ ] Log in with new credentials
- [ ] Log out and log back in
- [ ] Use resend setup email flow (before setting password)
- [ ] Trigger password reset, complete reset
- [ ] Access billing portal from header menu
- [ ] Access billing portal from settings page
- [ ] Use Stripe CLI to simulate invoice.payment_failed
- [ ] Verify banner appears in dashboard
- [ ] Wait for (simulated) grace period expiration
- [ ] Verify suspension and email
- [ ] Pay invoice, verify reactivation
- [ ] Cancel subscription, verify cancelled state
- [ ] Re-signup after cancellation (within 90 days), verify data restoration

---

## Security Considerations

### Token Security

- Setup/reset tokens: cryptographically random, 32+ bytes
- Tokens stored hashed (SHA-256) in database
- Single use: cleared on successful use
- Time-limited: 48h for setup, 1h for reset

### Session Security

- Session tokens: cryptographically random, 32+ bytes
- Stored hashed in database
- HttpOnly, Secure, SameSite=Lax cookies
- 7-day expiration, sliding window

### Password Security

- Minimum 8 characters (consider zxcvbn for strength)
- bcrypt hashing with cost factor 12
- No password in logs or error messages

### Webhook Security

- Verify Stripe signature on all webhooks
- Idempotent processing (webhook_events table)
- Rate limiting on public endpoints

---

## Deferred Features

These are explicitly out of scope for MVP:

| Feature | Deferral Reason | Add When |
|---------|-----------------|----------|
| Annual billing | Complexity | Customer demand |
| Multiple operators | Single-person shops | Customer request |
| Role-based permissions | Overkill for MVP | Multiple operators |
| OAuth (Google login) | Not expected by audience | If requested |
| Subdomain routing | Path routing sufficient | Scale/branding needs |
| Free trial | Discounted intro serves same purpose | Marketing strategy change |
| Email verification | Stripe verifies email via payment | Add free tier |
| Account deletion | Manual for now | Self-service demand |

---

## Success Metrics

After launch, track:

1. **Conversion rate**: Landing page visits → Checkout started → Checkout completed
2. **Setup completion**: Checkout completed → Password set (within 48h)
3. **Time to first product**: Account setup → First product created
4. **Churn in discount period**: Cancellations in first 3 months
5. **Payment failure recovery**: Past due → Recovered vs. Suspended

---

## Resolved Decisions

### Business Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Resend setup link | **Build it** | Self-service from day one, reduces support burden |
| Billing portal access | **Both places** | Header menu for quick access + settings page for discoverability |
| Data retention | **90 days** | Longer window for reactivation; soft-delete after |
| Reactivation flow | **New checkout** | Simpler than resurrection flow; data restored if within 90-day window |
| Billing portal return | **Dashboard** | Natural "home" after managing billing |

### Architecture Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Cookie scoping | **Path-based** | Operator cookie `path=/admin`, customer cookie `path=/`; prevents conflicts |
| Session token storage | **Hash both** | SHA-256 for operator AND customer sessions; consistent security model |
| Tenant pending state | **Block both** | Block storefront and admin until setup complete; no confusing empty stores |
| SaaS webhook idempotency | **Master tenant** | Use existing `webhook_events` table with master tenant ID for platform events |
| Slug collisions | **Append numbers** | Auto-generate `-2`, `-3` suffixes; reduces friction |
| Grace period calculation | **168 hours** | Exactly 7 days from `grace_period_started_at` timestamp |
| Email rate limiting | **IP + email** | Rate limit by both IP (3/hr) and email address (3/hr); prevents abuse |
| Role checks | **Defer** | MVP is single user; add role middleware when multi-user is implemented |
