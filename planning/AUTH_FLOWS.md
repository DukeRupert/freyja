# Authentication Flows Overview

This document provides a strategic survey of the three distinct signup/authentication flows in Freyja, their relationships, and potential conflicts.

**Last updated:** December 6, 2024

---

## The Three User Types

Freyja has three distinct user types, each with different authentication needs:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FREYJA PLATFORM                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. TENANT OPERATORS (Coffee Roasters who PAY for Freyja)                   │
│     ├── Pay $149/month for platform access                                  │
│     ├── Log into admin dashboard to manage their store                      │
│     ├── Create products, process orders, manage customers                   │
│     └── One owner per tenant for MVP                                        │
│                                                                             │
│  2. ADMIN USERS (Current Implementation - Transitional)                     │
│     ├── Bootstrap user created via environment variables                    │
│     ├── Uses same `users` table as customers (account_type='admin')         │
│     └── Will be migrated to tenant_operators in SaaS rollout                │
│                                                                             │
│  3. STOREFRONT CUSTOMERS (People who BUY coffee)                            │
│     ├── Sign up on a roaster's storefront                                   │
│     ├── Can be retail (immediate payment) or wholesale (invoiced)           │
│     ├── View orders, manage addresses, handle subscriptions                 │
│     └── Many per tenant                                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Flow Comparison Matrix

| Aspect | Tenant Operator | Admin (Current) | Storefront Customer |
|--------|-----------------|-----------------|---------------------|
| **Status** | NOT IMPLEMENTED | ✅ Implemented | ✅ Implemented |
| **Entry Point** | Stripe Checkout | Bootstrap on startup | `/signup` form |
| **Database Table** | `tenant_operators` (planned) | `users` (temp) | `users` |
| **Session Table** | `operator_sessions` (planned) | `sessions` (shared) | `sessions` |
| **Cookie Name** | `freyja_operator` (planned) | `freyja_session` | `freyja_session` |
| **Cookie Path** | `/admin` (planned) | `/` | `/` |
| **Email Verification** | Via Stripe payment | Auto-verified | Required |
| **Password Setup** | Email link after purchase | Env var at startup | During signup |
| **Protected Routes** | `/admin/*` | `/admin/*` | `/account/*`, `/checkout` |
| **Account Type Field** | N/A (separate table) | `account_type='admin'` | `retail` or `wholesale` |

---

## Flow 1: Tenant Signup (SaaS Onboarding)

**Status:** NOT IMPLEMENTED (Planned in SAAS_ONBOARDING.md)

### Sequence Diagram

```
                         TENANT SIGNUP FLOW

    Landing Page                Stripe                    Freyja                    Email
         │                        │                         │                         │
         │  "Get Started"         │                         │                         │
         ├───────────────────────►│                         │                         │
         │                        │                         │                         │
         │                   Checkout Page                  │                         │
         │             (email, business name,               │                         │
         │              payment method, $5)                 │                         │
         │                        │                         │                         │
         │                        │  checkout.session       │                         │
         │                        │     .completed          │                         │
         │                        ├────────────────────────►│                         │
         │                        │                         │                         │
         │                        │              Create tenant (pending)              │
         │                        │              Create operator (no password)        │
         │                        │              Generate setup token                 │
         │                        │                         │                         │
         │                        │                         │   Welcome Email         │
         │                        │                         ├────────────────────────►│
         │                        │                         │   (setup link)          │
         │                        │                         │                         │
         │◄──────────────────────────────────────────────────────────────────────────┤
         │                        │                         │    Click link           │
         │                        │                         │                         │
         │  GET /setup?token=xxx  │                         │                         │
         ├─────────────────────────────────────────────────►│                         │
         │                        │                         │                         │
         │                  Password form                   │                         │
         │◄─────────────────────────────────────────────────┤                         │
         │                        │                         │                         │
         │  POST /setup           │                         │                         │
         │  (password)            │                         │                         │
         ├─────────────────────────────────────────────────►│                         │
         │                        │              Set password_hash                    │
         │                        │              Activate tenant                      │
         │                        │              Create operator session              │
         │                        │              Set freyja_operator cookie           │
         │                        │                         │                         │
         │          Redirect to /admin                      │                         │
         │◄─────────────────────────────────────────────────┤                         │
         │                        │                         │                         │
         ▼                        │                         │                         │
    Admin Dashboard               │                         │                         │
```

### Key Decisions

- **No free trial**: $5/month intro offer validates commitment
- **Email verified by payment**: Stripe ensures valid email during checkout
- **48-hour setup window**: Reasonable time to complete password setup
- **7-day session duration**: Balance convenience and security

### Database Tables (Planned)

```sql
-- tenant_operators: People who PAY for and MANAGE a Freyja store
CREATE TABLE tenant_operators (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),        -- NULL until setup complete
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'owner',  -- owner, admin, staff (future)
    setup_token VARCHAR(255),
    setup_token_expires_at TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'pending', -- pending, active, suspended
    UNIQUE(tenant_id, email)
);

-- operator_sessions: Separate from customer sessions
CREATE TABLE operator_sessions (
    id UUID PRIMARY KEY,
    operator_id UUID REFERENCES tenant_operators(id),
    token_hash VARCHAR(255) NOT NULL,  -- SHA-256 of session token
    expires_at TIMESTAMPTZ NOT NULL
);
```

---

## Flow 2: Admin Login (Current Implementation)

**Status:** ✅ IMPLEMENTED (Transitional - will be replaced by Tenant Operator flow)

### Sequence Diagram

```
                        ADMIN LOGIN FLOW (CURRENT)

    Application Start                Freyja                       Database
           │                           │                             │
           │  Check FREYJA_ADMIN_*     │                             │
           │  env vars                 │                             │
           ├──────────────────────────►│                             │
           │                           │                             │
           │                           │  Check if admin exists      │
           │                           ├────────────────────────────►│
           │                           │                             │
           │                           │  If not, create with        │
           │                           │  account_type='admin'       │
           │                           │  email_verified=true        │
           │                           ├────────────────────────────►│
           │                           │                             │


    Admin User                      Freyja                       Database
         │                            │                             │
         │  GET /admin/login          │                             │
         ├───────────────────────────►│                             │
         │                            │                             │
         │    Login form              │                             │
         │◄───────────────────────────┤                             │
         │                            │                             │
         │  POST /admin/login         │                             │
         │  (email, password)         │                             │
         ├───────────────────────────►│                             │
         │                            │                             │
         │                            │  GetUserByEmail             │
         │                            ├────────────────────────────►│
         │                            │                             │
         │                            │  Verify password            │
         │                            │  Check account_type='admin' │
         │                            │                             │
         │                            │  CreateSession              │
         │                            ├────────────────────────────►│
         │                            │                             │
         │   Set freyja_session       │                             │
         │   Redirect to /admin       │                             │
         │◄───────────────────────────┤                             │
         │                            │                             │
         ▼                            │                             │
    Admin Dashboard                   │                             │
```

### Current Implementation Details

| Component | Location |
|-----------|----------|
| Login Handler | `internal/handler/admin/auth.go` |
| Bootstrap | `internal/bootstrap/admin.go` |
| Middleware | `middleware.RequireAdmin()` |
| Routes | `internal/routes/admin.go` |

### Known Issues

1. **Shared users table**: Admins use same table as customers (semantic confusion)
2. **Shared sessions table**: Cannot distinguish admin vs customer sessions at DB level
3. **No email verification**: Admin created with `email_verified=true` automatically
4. **Single admin per tenant**: No staff/team support yet

---

## Flow 3: Storefront Customer Signup

**Status:** ✅ IMPLEMENTED

### Sequence Diagram

```
                     CUSTOMER SIGNUP FLOW

    Customer                       Freyja                     Database              Email
        │                            │                           │                    │
        │  GET /signup               │                           │                    │
        ├───────────────────────────►│                           │                    │
        │                            │                           │                    │
        │    Signup form             │                           │                    │
        │◄───────────────────────────┤                           │                    │
        │                            │                           │                    │
        │  POST /signup              │                           │                    │
        │  (email, password,         │                           │                    │
        │   name)                    │                           │                    │
        ├───────────────────────────►│                           │                    │
        │                            │                           │                    │
        │                            │  Create user              │                    │
        │                            │  account_type='retail'    │                    │
        │                            │  email_verified=false     │                    │
        │                            ├──────────────────────────►│                    │
        │                            │                           │                    │
        │                            │  Generate verification    │                    │
        │                            │  token (SHA-256 hashed)   │                    │
        │                            ├──────────────────────────►│                    │
        │                            │                           │                    │
        │                            │         Verification Email                     │
        │                            ├───────────────────────────────────────────────►│
        │                            │                           │                    │
        │   Redirect to              │                           │                    │
        │   /signup-success          │                           │                    │
        │◄───────────────────────────┤                           │                    │
        │                            │                           │                    │
        │◄──────────────────────────────────────────────────────────────────────────┤
        │   Click verification link  │                           │                    │
        │                            │                           │                    │
        │  GET /verify-email?        │                           │                    │
        │      token=xxx             │                           │                    │
        ├───────────────────────────►│                           │                    │
        │                            │                           │                    │
        │                            │  Validate token           │                    │
        │                            │  Mark email_verified=true │                    │
        │                            ├──────────────────────────►│                    │
        │                            │                           │                    │
        │   Success page             │                           │                    │
        │◄───────────────────────────┤                           │                    │
        │                            │                           │                    │
        │  GET /login                │                           │                    │
        ├───────────────────────────►│                           │                    │
        │                            │                           │                    │
        │  POST /login               │                           │                    │
        │  (email, password)         │                           │                    │
        ├───────────────────────────►│                           │                    │
        │                            │                           │                    │
        │                            │  Check email_verified     │                    │
        │                            │  Create session           │                    │
        │                            ├──────────────────────────►│                    │
        │                            │                           │                    │
        │   Set freyja_session       │                           │                    │
        │   Redirect to /account     │                           │                    │
        │◄───────────────────────────┤                           │                    │
        ▼                            │                           │                    │
    Account Dashboard                │                           │                    │
```

### Wholesale Application Sub-Flow

```
                    WHOLESALE APPLICATION FLOW

    Retail Customer              Freyja                     Database              Admin
         │                          │                           │                    │
         │  (Already logged in      │                           │                    │
         │   as retail customer)    │                           │                    │
         │                          │                           │                    │
         │  GET /wholesale/apply    │                           │                    │
         ├─────────────────────────►│                           │                    │
         │                          │                           │                    │
         │    Application form      │                           │                    │
         │◄─────────────────────────┤                           │                    │
         │                          │                           │                    │
         │  POST /wholesale/apply   │                           │                    │
         │  (company_name,          │                           │                    │
         │   business_type,         │                           │                    │
         │   tax_id, etc.)          │                           │                    │
         ├─────────────────────────►│                           │                    │
         │                          │                           │                    │
         │                          │  Set wholesale_           │                    │
         │                          │  application_status=      │                    │
         │                          │  'pending'                │                    │
         │                          ├──────────────────────────►│                    │
         │                          │                           │                    │
         │   Redirect to            │                           │                    │
         │   /wholesale/status      │                           │                    │
         │◄─────────────────────────┤                           │                    │
         │                          │                           │                    │
         │                          │                           │ Reviews in         │
         │                          │                           │ admin dashboard    │
         │                          │                           │◄───────────────────┤
         │                          │                           │                    │
         │                          │                           │ POST /admin/       │
         │                          │                           │ customers/{id}/    │
         │                          │                           │ wholesale/approve  │
         │                          │                           │◄───────────────────┤
         │                          │                           │                    │
         │                          │  Set account_type=        │                    │
         │                          │  'wholesale'              │                    │
         │                          │  Assign payment_terms     │                    │
         │                          │  Assign price_list        │                    │
         │                          ├──────────────────────────►│                    │
         │                          │                           │                    │
         │                          │                    [EMAIL NOTIFICATION MISSING]│
         │                          │                           │                    │
         │  (Customer logs in       │                           │                    │
         │   and checks status)     │                           │                    │
         │                          │                           │                    │
         │  GET /wholesale/order    │                           │                    │
         ├─────────────────────────►│                           │                    │
         │                          │                           │                    │
         │    Matrix ordering UI    │                           │                    │
         │◄─────────────────────────┤                           │                    │
         ▼                          │                           │                    │
```

### Current Implementation Details

| Component | Location |
|-----------|----------|
| Signup/Login Handler | `internal/handler/storefront/auth.go` |
| Wholesale Application | `internal/handler/storefront/wholesale.go` |
| Wholesale Ordering | `internal/handler/storefront/wholesale_ordering.go` |
| Email Verification Service | `internal/service/email_verification.go` |
| Password Reset Service | `internal/service/password_reset.go` |
| Middleware | `middleware.RequireAuth()` |
| Routes | `internal/routes/storefront.go` |

### Security Features

- **Email verification required**: Cannot login until verified
- **Token hashing**: Verification tokens stored as SHA-256 hashes
- **Rate limiting**: 5 requests/user/hour, 10 requests/IP/hour
- **24-hour token expiry**: Security without inconvenience
- **bcrypt password hashing**: Industry standard

---

## Potential Conflicts & Resolutions

### 1. Shared `users` Table for Admins and Customers

**Current State:**
```sql
users (
    account_type VARCHAR -- 'retail', 'wholesale', 'admin'
)
```

**Problem:** Admin users and storefront customers share the same table, creating semantic confusion.

**Resolution (Planned):**
- Create `tenant_operators` table for platform operators
- Keep `users` table for storefront customers only
- Remove `account_type='admin'` usage
- Migrate existing bootstrap admins to tenant_operators

**Migration Path:**
1. Create `tenant_operators` table
2. Create `operator_sessions` table
3. Migrate existing admin users to tenant_operators
4. Remove `account_type='admin'` code paths
5. Drop `admin` from account_type enum

---

### 2. Shared Sessions Table

**Current State:**
```sql
sessions (
    token VARCHAR,
    data JSONB  -- { "user_id": "xxx" }
)
```

**Problem:** Cannot distinguish admin vs customer sessions at database level.

**Resolution (Planned):**
- Separate `operator_sessions` table for platform operators
- Different cookie names: `freyja_operator` (path=/admin) vs `freyja_session` (path=/)
- Session tokens hashed in both tables (SHA-256)

**Cookie Scoping:**
```
freyja_operator cookie:
  - Path: /admin
  - Used by: tenant_operators
  - Session table: operator_sessions

freyja_session cookie:
  - Path: /
  - Used by: storefront customers
  - Session table: sessions
```

---

### 3. Email Uniqueness Constraints

**Current State:**
```sql
UNIQUE(tenant_id, email)  -- Per-tenant uniqueness
```

**Potential Issues:**
- Same email can exist in multiple tenants (correct for customers)
- Admin email should be globally unique? (No - one admin per tenant is fine)
- Tenant operator email: unique per tenant (same constraint, different table)

**Resolution:** Current constraint is correct. Each table has its own uniqueness:
- `users`: `UNIQUE(tenant_id, email)` - Customer can have accounts at multiple roasters
- `tenant_operators`: `UNIQUE(tenant_id, email)` - One operator account per tenant

---

### 4. Missing Wholesale Approval Notification

**Current State:** Admin approves wholesale application, but customer is not notified.

**Resolution (Needed):**
```go
// In admin/customers.go WholesaleApproval handler:
if action == "approve" {
    // After updating account_type...
    jobData := map[string]interface{}{
        "user_id":   user.ID.String(),
        "tenant_id": tenantID.String(),
    }
    h.jobService.EnqueueJob(ctx, tenantID, "email:wholesale_approved", jobData)
}
```

Add email template: `web/templates/email/wholesale_approved.html`

---

### 5. Password Terms Column Duplication

**Current State:**
```sql
users (
    payment_terms VARCHAR,       -- Old string column (deprecated)
    payment_terms_id UUID        -- New FK to payment_terms table
)
```

**Resolution (Needed):**
1. Ensure all code uses `payment_terms_id`
2. Backfill any existing data
3. Add migration to drop `payment_terms` column (or rename to `payment_terms_legacy`)

---

## Route Structure Summary

### Public Routes (No Authentication)

```
Storefront:
  GET  /                         # Homepage
  GET  /products                 # Product listing
  GET  /products/{slug}          # Product detail
  GET  /cart                     # View cart
  POST /cart/add                 # Add to cart
  GET  /signup                   # Customer signup form
  POST /signup                   # Submit signup
  GET  /login                    # Customer login form
  POST /login                    # Submit login
  GET  /verify-email             # Email verification
  GET  /forgot-password          # Password reset request
  POST /forgot-password          # Submit reset request
  GET  /reset-password           # Password reset form
  POST /reset-password           # Submit new password

Admin:
  GET  /admin/login              # Admin login form
  POST /admin/login              # Submit admin login

SaaS (Planned):
  GET  /                         # Landing page
  POST /api/create-checkout      # Create Stripe checkout session
  GET  /setup                    # Password setup after purchase
  POST /setup                    # Submit password
```

### Protected Routes (Customer Authentication Required)

```
  GET  /account                  # Account dashboard
  GET  /account/orders           # Order history
  GET  /account/addresses        # Address book
  GET  /account/subscriptions    # Subscription management
  GET  /account/settings         # Profile settings
  GET  /checkout                 # Checkout flow
  POST /checkout                 # Submit order
  GET  /wholesale/apply          # Wholesale application form
  POST /wholesale/apply          # Submit application
  GET  /wholesale/status         # Application status
  GET  /wholesale/order          # Matrix ordering (wholesale only)
  POST /wholesale/cart/batch     # Batch add to cart (wholesale only)
```

### Protected Routes (Admin/Operator Authentication Required)

```
  GET  /admin                    # Admin dashboard
  GET  /admin/products           # Product management
  GET  /admin/orders             # Order management
  GET  /admin/customers          # Customer management
  GET  /admin/invoices           # Invoice management
  GET  /admin/subscriptions      # Subscription overview
  GET  /admin/settings           # Store settings
  POST /admin/logout             # Admin logout
```

---

## Implementation Priority

### Phase 1: Fix Immediate Issues (Before Multi-Tenant)

1. **Add wholesale approval email notification**
   - Create email template
   - Add job enqueueing in approval handler
   - Priority: HIGH (improves customer experience)

2. **Clean up payment_terms columns**
   - Ensure consistent usage of `payment_terms_id`
   - Migration to drop/rename old column
   - Priority: MEDIUM (technical debt)

### Phase 2: SaaS Onboarding (Multi-Tenant Launch)

1. **Create tenant_operators table and service**
2. **Create operator_sessions table**
3. **Implement Stripe Checkout webhook handler**
4. **Build setup flow UI**
5. **Separate cookie scoping**
6. **Migrate bootstrap admin to tenant_operators**

### Phase 3: Polish

1. **Add subscription payment failure handling**
2. **Implement grace period jobs**
3. **Build billing portal integration**
4. **Add past_due UI banner**

---

## Security Checklist

### Current Implementation

- [x] bcrypt password hashing
- [x] HttpOnly session cookies
- [x] CSRF protection (except webhooks)
- [x] Email verification for customers
- [x] Rate limiting on verification/reset requests
- [x] Token hashing (SHA-256) for verification/reset tokens
- [x] Webhook signature verification
- [x] Idempotent webhook processing

### Planned for SaaS Launch

- [ ] Separate session tables for operators/customers
- [ ] Path-scoped cookies (/admin vs /)
- [ ] Operator session token hashing
- [ ] Tenant status checks on all routes (pending blocks access)
- [ ] Grace period enforcement

---

## Testing Strategy

### Unit Tests

- Token generation and validation
- Password hashing and verification
- Slug generation (unicode, duplicates)
- Session creation and validation

### Integration Tests

- Complete signup → verify → login flow
- Wholesale application → approval flow
- Password reset flow
- Admin login flow

### Manual Testing Checklist

- [ ] Customer signup with email verification
- [ ] Customer login after verification
- [ ] Customer password reset
- [ ] Wholesale application submission
- [ ] Admin approval of wholesale application
- [ ] Admin login and logout
- [ ] Session expiration handling
- [ ] Rate limiting on verification requests

---

## Related Documents

- [SAAS_ONBOARDING.md](./SAAS_ONBOARDING.md) - Detailed SaaS onboarding plan
- [TECHNICAL.md](./TECHNICAL.md) - Technical architecture decisions
- [ROADMAP.md](./ROADMAP.md) - Feature roadmap and status
