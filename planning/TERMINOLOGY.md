# Terminology Reference

This document defines the key terms used throughout the Hiri codebase to ensure consistent understanding.

---

## Core Entities

### Tenant

A coffee roaster business that subscribes to Hiri.

| Attribute | Description |
|-----------|-------------|
| Table | `tenants` |
| Example | "Acme Coffee Roasters" |
| Subscription | $149/month SaaS fee |
| Storefront | `{slug}.hiri.coffee` or custom domain |
| Status | `pending`, `active`, `past_due`, `suspended`, `cancelled` |

A tenant represents the business entity. Each tenant has:
- One or more **operators** who manage the store
- Zero or more **customers** who buy from the store
- Their own products, orders, invoices, etc.

---

### Operator

A person who manages a tenant's store (the roaster's staff).

| Attribute | Description |
|-----------|-------------|
| Table | `tenant_operators` |
| Foreign Key | `tenant_id` → `tenants.id` |
| Auth Service | `OperatorService` |
| Session Cookie | `hiri_operator_session` |
| Roles | `owner`, `staff` |
| Status | `pending`, `active`, `suspended` |

Operators:
- Log in to the **admin dashboard** (`/admin`)
- Manage products, orders, customers, settings
- The `owner` is created during tenant signup
- Additional `staff` operators can be invited later

**Login location:** `/admin/login` on the main app (port 3000 in dev)

---

### Customer

A person who buys coffee from a tenant's storefront.

| Attribute | Description |
|-----------|-------------|
| Table | `customers` |
| Foreign Key | `tenant_id` → `tenants.id` |
| Auth Service | `UserService` (storefront) |
| Session Cookie | `hiri_session` |
| Types | Retail, Wholesale |
| Status | `pending`, `active`, `suspended` |

Two customer types:

**Retail Customer**
- Regular consumer
- Pays immediately at checkout
- Can have subscriptions (recurring coffee deliveries)

**Wholesale Customer**
- B2B account (café, restaurant, office)
- Applies for approval via `/wholesale/apply`
- Has payment terms (Net 15/30)
- Assigned to a price list/tier
- Receives consolidated invoices

**Login location:** `/login` on the tenant's storefront

---

### User (Legacy/Deprecated)

A legacy entity from before multi-tenancy was implemented.

| Attribute | Description |
|-----------|-------------|
| Table | `users` |
| Auth Service | `UserService` (admin) |
| Status | **Being phased out** |

The `users` table was originally used for:
- Single-tenant admin users
- Storefront customers (before `customers` table existed)

**Migration path:** Replace with `tenant_operators` for admin access.

---

## Authentication Contexts

| Context | Entity | Service | Login URL | Cookie |
|---------|--------|---------|-----------|--------|
| Admin Dashboard | Operator | `OperatorService` | `/admin/login` | `hiri_operator_session` |
| Storefront | Customer | `UserService` | `/login` | `hiri_session` |
| SaaS Marketing | (none) | - | - | - |

---

## Development Ports

| Port | Domain (Production) | Purpose |
|------|---------------------|---------|
| 3000 | `app.hiri.coffee` / `{slug}.hiri.coffee` | Main app: admin dashboard, storefronts, API, webhooks |
| 3001 | `hiri.coffee` | Marketing site: landing, pricing, signup flow |

---

## Signup Flows

### Tenant Signup (Operator Onboarding)

1. Visit `/pricing` on marketing site (port 3001)
2. Complete Stripe Checkout (or `/dev/signup` in development)
3. Tenant created with status `pending`
4. Operator created with status `pending`
5. Welcome email sent with setup link
6. Operator sets password at `/setup?token=xxx`
7. Operator logs in at `/admin/login` (port 3000)

### Customer Signup

1. Visit `/signup` on tenant storefront
2. Enter email, password, name
3. Customer created with status `pending`
4. Verification email sent
5. Customer verifies email
6. Customer logs in at `/login`

### Wholesale Application

1. Visit `/wholesale/apply` on tenant storefront
2. Submit application (business name, tax ID, etc.)
3. Application created with status `pending_approval`
4. Tenant operator reviews in admin dashboard
5. Operator approves and sets tier/terms
6. Applicant notified and completes account setup
