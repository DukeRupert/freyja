# Subscription Feature Design

**Created:** December 2024
**Status:** ✅ Feature Complete
**Last Updated:** December 2, 2024

---

## Implementation Progress

### Completed ✅

**Phase 1: Billing Provider Extensions**
- [x] Add subscription types to `internal/billing/billing.go`
- [x] Implement Stripe subscription methods in `internal/billing/stripe.go`
  - CreateRecurringPrice, CreateSubscription, GetSubscription
  - PauseSubscription, ResumeSubscription, CancelSubscription
  - CreateCustomerPortalSession, CreateProduct
- [x] Fix Stripe v83 API compatibility (CurrentPeriodStart/End on SubscriptionItem)
- [x] Add `GetInvoice` method to billing provider

**Phase 2: SQLc Queries**
- [x] Create `sqlc/queries/subscriptions.sql` with comprehensive queries
- [x] Generate repository code via `sqlc generate`
- [x] Includes: CRUD operations, user queries, admin queries, stats
- [x] Add `subscription_id` to CreateOrder query
- [x] Add `ListOrdersBySubscription` query for order history
- [x] Add `sqlc/queries/addresses.sql` - ListAddressesForUser, GetAddressByIDForUser
- [x] Add `sqlc/queries/billing.sql` - ListPaymentMethodsForUser, GetPaymentMethodByID

**Phase 3: Subscription Service**
- [x] Create `internal/service/subscription.go` (interface + types)
- [x] Create `internal/service/subscription_impl.go` (implementation)
- [x] Implement CreateSubscription, GetSubscription, ListSubscriptionsForUser
- [x] Implement PauseSubscription, ResumeSubscription, CancelSubscription
- [x] Implement CreateCustomerPortalSession, SyncSubscriptionFromWebhook
- [x] Implement `CreateOrderFromSubscriptionInvoice` - creates orders from paid invoices
- [x] Initialize service in main.go
- [x] Create `internal/service/account.go` (AccountService for addresses/payment methods)

**Phase 4: Webhook Handlers**
- [x] Extend `internal/handler/webhook/stripe.go`
- [x] Handle `invoice.payment_succeeded` → triggers order creation
- [x] Handle `invoice.payment_failed` → updates status to past_due
- [x] Handle `customer.subscription.updated` → syncs subscription state
- [x] Handle `customer.subscription.deleted` → marks as expired
- [x] Fix Stripe v83 Invoice API (subscription now in Parent.SubscriptionDetails)
- [x] Fix Stripe v83 Invoice.Payments API (PaymentIntent now in Payments.Data[].Payment)

**Phase 5: Authentication & Authorization**
- [x] Add RequireAuth middleware to subscription routes
- [x] Add RequireAdmin middleware to admin routes
- [x] Update subscription handlers to use `middleware.GetUserFromContext(ctx)`
- [x] Add ownership validation to GetSubscription (UserID field)

**Phase 6: HTTP Handlers & UI**
- [x] Create `internal/handler/storefront/subscription.go`
  - SubscriptionListHandler: GET /account/subscriptions
  - SubscriptionDetailHandler: GET /account/subscriptions/{id}
  - SubscriptionPortalHandler: GET /account/subscriptions/portal
  - SubscriptionCheckoutHandler: GET /subscribe/checkout
  - CreateSubscriptionHandler: POST /subscribe
- [x] Create `internal/handler/admin/subscriptions.go`
  - SubscriptionListHandler: GET /admin/subscriptions
  - SubscriptionDetailHandler: GET /admin/subscriptions/{id}
- [x] Create templates:
  - `web/templates/storefront/subscriptions.html`
  - `web/templates/storefront/subscription_detail.html`
  - `web/templates/storefront/subscription_checkout.html`
  - `web/templates/admin/subscriptions.html`
  - `web/templates/admin/subscription_detail.html`
- [x] Register all routes in main.go
- [x] Add `mulf` and `uuidToString` template functions

**Phase 7: Subscription Checkout Flow**
- [x] Add one-time/subscribe toggle to product detail page
- [x] Create dedicated subscription checkout page
- [x] Saved address selection with radio buttons
- [x] Saved payment method selection
- [x] Delivery frequency selector
- [x] Order summary with subscription benefits

### Remaining (Post-MVP) 🔲

**Testing**
- [ ] Update test mocks to include new interface methods (ClearCart, subscription methods)
- [ ] Add integration tests for subscription flow
- [ ] Test pause/resume/cancel with Stripe CLI

**Email Notifications**
- [ ] Subscription created confirmation
- [ ] Renewal reminder (uses Stripe's invoice.upcoming webhook)
- [ ] Payment failed notification

**Future Enhancements**
- [ ] Skip next delivery feature
- [ ] In-app frequency/quantity changes (currently uses Stripe Portal)
- [ ] Product swap mid-subscription

### Git Commits

1. `d1639d5` - feat: implement subscription billing and webhook handlers
2. `73ab28d` - feat: add subscription HTTP handlers and templates
3. `4f678fb` - feat: complete subscription order creation from invoices
4. `e52aa1c` - feat: add subscription auth middleware and product page subscribe option
5. `ccd1372` - feat: add subscription checkout page flow

### Known Issues (Resolved)

1. **Stripe v83 API Changes**: ✅ Fixed - Invoice no longer has direct `Subscription` field. Access via `invoice.Parent.SubscriptionDetails.Subscription`.

2. **pgtype.Numeric**: ✅ Handled - Use `Float64Value()` method, check `.Valid` before access.

3. **Authentication**: ✅ Fixed - All storefront handlers now use `middleware.GetUserFromContext(ctx)` with RequireAuth middleware.

4. **Test Mocks**: Still need updates for new interface methods added for subscriptions.

## Overview

Subscriptions allow customers to receive recurring coffee deliveries at configurable intervals. This document defines the architecture, implementation approach, and key decisions for the subscription feature.

**MVP Scope:**
- Single-item subscriptions only (multi-item deferred)
- No trial periods — standard billing from day one
- No "skip next delivery" — can be added post-MVP
- Stripe Customer Portal for self-service (no custom portal UI)

---

## Architecture Decision: Separate Checkout Flows

**Decision:** Subscriptions have their own checkout flow, completely separate from the cart-based one-time purchase checkout.

**Rationale:**
1. **Conceptual clarity:** Subscriptions are fundamentally different from one-time purchases — different data requirements (billing interval, recurring payment), different user intent, different post-purchase management
2. **Implementation simplicity:** Avoids complexity of mixed carts with both one-time and subscription items
3. **UX cleanliness:** Each flow can be optimized for its purpose without compromise
4. **Stripe alignment:** Stripe treats subscriptions and one-time payments as separate entities

**How it works:**

```
ONE-TIME PURCHASE FLOW:
  Product Page → [Add to Cart] → /cart → /checkout → Stripe Payment Intent → Order

SUBSCRIPTION FLOW:
  Product Page → [Subscribe] → /subscribe/checkout → POST /subscribe → Stripe Subscription → Order (via webhook)
```

**Product Detail Page Behavior:**
- Toggle between "One-time purchase" and "Subscribe & Save"
- One-time: Form POSTs to `/cart/add`, adds item to cart
- Subscribe: Form GETs to `/subscribe/checkout`, redirects to subscription checkout page

**Subscription Checkout Page (`/subscribe/checkout`):**
- Requires authentication (RequireAuth middleware)
- Displays product/SKU summary
- Delivery frequency selector
- Select from saved addresses (radio buttons)
- Select from saved payment methods (radio buttons)
- POSTs to `/subscribe` to create subscription

**Key Files:**
- `internal/handler/storefront/subscription.go` - SubscriptionCheckoutHandler
- `web/templates/storefront/subscription_checkout.html` - Checkout UI
- `web/templates/storefront/product_detail.html` - Toggle and form logic

**Tradeoffs accepted:**
- Users cannot combine subscription items with one-time items in single checkout
- Requires saved addresses/payment methods (users must add these first)
- Two separate checkout experiences to maintain

**Future consideration:**
If multi-item subscriptions are needed, this architecture supports it — the subscription checkout page can display multiple subscription items from a "subscription cart" concept without affecting one-time checkout.

---

## Architecture Decision: Stripe-First

**Decision:** Use Stripe Billing as source of truth for scheduling and payments. Local database is a read model synced via webhooks.

**Rationale:**
1. **Simplicity for solo maintainer:** Stripe handles scheduling, dunning, retry logic, proration — all complex problems that would take weeks to build correctly
2. **Aligns with existing pattern:** Order creation already happens via webhook (`OrderService.CreateOrderFromPaymentIntent`)
3. **Customer Portal works:** Stripe's hosted portal handles pause/resume/cancel/update payment method without custom UI
4. **Reliability:** Stripe's battle-tested billing system is more reliable than a first-pass custom implementation

**Tradeoffs accepted:**
- Eventual consistency for subscription status (webhook latency typically < 1 second)
- Dependence on Stripe API for subscription queries
- Less flexibility for exotic billing schedules (acceptable for coffee subscriptions)

**Extensibility plan:**
- Custom skip functionality can be layered on later via `subscription_schedule`
- Product swaps handled by updating Stripe subscription items + local snapshot
- Future wholesale subscriptions could use local scheduling if needed

---

## Database Schema (Already Implemented)

```
subscription_plans       — Templates (optional, for admin-defined plans)
subscriptions           — Customer subscription instances
subscription_items      — Products in subscription (single item for MVP)
subscription_schedule   — Event log for billing/pause/resume events
orders.subscription_id  — Links orders back to originating subscription
```

**Key fields in `subscriptions`:**
- `provider_subscription_id` — Stripe subscription ID
- `status` — active, paused, cancelled, past_due, expired
- `billing_interval` — weekly, biweekly, monthly, every_6_weeks, every_2_months
- `next_billing_date` — When next charge occurs
- `shipping_address_id` — Where to ship

---

## Stripe Entity Mapping

| Freyja Entity | Stripe Entity | Notes |
|---------------|---------------|-------|
| Product SKU | Stripe Product | Created per tenant, per subscribable SKU |
| Subscription pricing | Stripe Price | Created per subscription (not reused) |
| Subscription | Stripe Subscription | Links to Stripe Customer |
| Renewal | Stripe Invoice | Auto-generated by Stripe Billing |

**Why one Stripe Price per subscription:**
- Each subscription may have custom pricing based on customer's price list
- Avoids conflicts when price lists change
- Simplifies logic at cost of more Stripe objects

**Stripe metadata on all objects:**
```json
{
  "tenant_id": "uuid",
  "subscription_id": "uuid",
  "user_id": "uuid"
}
```

---

## Billing Interval Mapping

| Freyja Interval | Stripe Interval | Stripe Interval Count |
|-----------------|-----------------|----------------------|
| weekly | week | 1 |
| biweekly | week | 2 |
| monthly | month | 1 |
| every_6_weeks | week | 6 |
| every_2_months | month | 2 |

---

## Stripe API Reference

**Subscription Statuses:** `trialing`, `active`, `incomplete`, `incomplete_expired`, `past_due`, `canceled`, `unpaid`, `paused`

**Key Webhook Events:**
| Event | When | Action |
|-------|------|--------|
| `invoice.payment_succeeded` | Subscription renewal paid | Create order |
| `invoice.payment_failed` | Payment failed | Set status to `past_due` |
| `customer.subscription.updated` | Status/settings changed | Sync local state |
| `customer.subscription.deleted` | Subscription ended | Set status to `expired` |

**Pause Subscription:** Update with `pause_collection.behavior` = "void" (voids open invoices)

**Resume Subscription:** Update with `pause_collection` = empty/null

**Cancel Subscription:**
- `cancel_at_period_end=true` → Cancel at end of period
- DELETE `/v1/subscriptions/{id}` → Cancel immediately

**Customer Portal:** `POST /v1/billing_portal/sessions` with `customer` and `return_url`

---

## Billing Provider Interface Extensions

Add to `internal/billing/billing.go`:

```go
// New methods to add to Provider interface
type Provider interface {
    // ... existing methods ...

    // CreateRecurringPrice creates a Stripe Price for recurring subscriptions.
    CreateRecurringPrice(ctx context.Context, params CreateRecurringPriceParams) (*Price, error)

    // CreateSubscription creates a recurring subscription.
    CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*Subscription, error)

    // GetSubscription retrieves an existing subscription.
    GetSubscription(ctx context.Context, params GetSubscriptionParams) (*Subscription, error)

    // PauseSubscription pauses a subscription until explicitly resumed.
    PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*Subscription, error)

    // ResumeSubscription resumes a paused subscription immediately.
    ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*Subscription, error)

    // CancelSubscription cancels a subscription (updated signature with tenant validation).
    CancelSubscription(ctx context.Context, params CancelSubscriptionParams) error

    // CreateCustomerPortalSession creates a Stripe Customer Portal session.
    CreateCustomerPortalSession(ctx context.Context, params CreatePortalSessionParams) (*PortalSession, error)
}
```

### Billing Package Types

```go
// CreateRecurringPriceParams for creating recurring prices
type CreateRecurringPriceParams struct {
    Currency        string            // "usd" (ISO 4217 lowercase)
    UnitAmountCents int32             // Price per billing period in cents
    BillingInterval string            // "week" or "month"
    IntervalCount   int32             // Multiplier (2 for biweekly)
    ProductID       string            // Stripe product ID (prod_...)
    Metadata        map[string]string // Must include tenant_id
    Nickname        string            // Display name
}

// Price represents a Stripe price
type Price struct {
    ID              string
    ProductID       string
    Currency        string
    UnitAmountCents int32
    Type            string // "one_time" or "recurring"
    Recurring       *PriceRecurring
    Active          bool
    Metadata        map[string]string
    CreatedAt       time.Time
}

type PriceRecurring struct {
    Interval      string // "day", "week", "month", "year"
    IntervalCount int32
}

// CreateSubscriptionParams for creating subscriptions
type CreateSubscriptionParams struct {
    TenantID               string
    CustomerID             string // Stripe customer ID (cus_...)
    PriceID                string // Stripe price ID (price_...)
    Quantity               int32
    DefaultPaymentMethodID string // pm_...
    CollectionMethod       string // "charge_automatically" (default)
    Metadata               map[string]string
    IdempotencyKey         string
}

// Subscription represents a Stripe subscription (extended)
type Subscription struct {
    ID                     string
    CustomerID             string
    Status                 string // "active", "past_due", "canceled", etc.
    Items                  []SubscriptionItem
    DefaultPaymentMethodID string
    CurrentPeriodStart     time.Time
    CurrentPeriodEnd       time.Time
    CancelAtPeriodEnd      bool
    CanceledAt             *time.Time
    PauseCollection        *SubscriptionPauseCollection
    Metadata               map[string]string
    CreatedAt              time.Time
}

type SubscriptionItem struct {
    ID       string
    PriceID  string
    Quantity int32
    Metadata map[string]string
}

type SubscriptionPauseCollection struct {
    Behavior  string     // "void", "keep_as_draft", "mark_uncollectible"
    ResumesAt *time.Time
}

// GetSubscriptionParams for retrieving subscriptions
type GetSubscriptionParams struct {
    SubscriptionID string
    TenantID       string
    Expand         []string
}

// PauseSubscriptionParams for pausing subscriptions
type PauseSubscriptionParams struct {
    SubscriptionID string
    TenantID       string
    Behavior       string     // "void" recommended
    ResumesAt      *time.Time // nil for manual resume
}

// ResumeSubscriptionParams for resuming subscriptions
type ResumeSubscriptionParams struct {
    SubscriptionID string
    TenantID       string
}

// CancelSubscriptionParams for canceling subscriptions
type CancelSubscriptionParams struct {
    SubscriptionID     string
    TenantID           string
    CancelAtPeriodEnd  bool
    CancellationReason string
}

// CreatePortalSessionParams for customer portal
type CreatePortalSessionParams struct {
    CustomerID string
    TenantID   string
    ReturnURL  string
}

// PortalSession represents a Stripe Customer Portal session
type PortalSession struct {
    ID        string
    URL       string
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

---

## Service Layer Interface

Create `internal/service/subscription.go`:

```go
type SubscriptionService interface {
    // CreateSubscription creates a new subscription for a customer.
    CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*SubscriptionDetail, error)

    // GetSubscription retrieves subscription details.
    GetSubscription(ctx context.Context, params GetSubscriptionParams) (*SubscriptionDetail, error)

    // ListSubscriptionsForUser retrieves all subscriptions for a customer.
    ListSubscriptionsForUser(ctx context.Context, params ListSubscriptionsParams) ([]SubscriptionSummary, error)

    // PauseSubscription pauses a subscription until manually resumed.
    PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*SubscriptionDetail, error)

    // ResumeSubscription resumes a paused subscription immediately.
    ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*SubscriptionDetail, error)

    // CancelSubscription cancels a subscription.
    CancelSubscription(ctx context.Context, params CancelSubscriptionParams) (*SubscriptionDetail, error)

    // CreateCustomerPortalSession creates a Stripe Customer Portal session.
    CreateCustomerPortalSession(ctx context.Context, params PortalSessionParams) (string, error)

    // SyncSubscriptionFromWebhook updates local subscription from Stripe webhook.
    SyncSubscriptionFromWebhook(ctx context.Context, params SyncSubscriptionParams) error
}

type CreateSubscriptionParams struct {
    TenantID          pgtype.UUID
    UserID            pgtype.UUID
    ProductSKUID      pgtype.UUID
    Quantity          int32
    BillingInterval   string // "weekly", "biweekly", "monthly", "every_6_weeks", "every_2_months"
    ShippingAddressID pgtype.UUID
    ShippingMethodID  pgtype.UUID
    PaymentMethodID   pgtype.UUID
}

type GetSubscriptionParams struct {
    TenantID               pgtype.UUID
    SubscriptionID         pgtype.UUID
    IncludeUpcomingInvoice bool
}

type ListSubscriptionsParams struct {
    TenantID pgtype.UUID
    UserID   pgtype.UUID
    Status   *string
    Limit    int32
    Offset   int32
}

type PauseSubscriptionParams struct {
    TenantID       pgtype.UUID
    SubscriptionID pgtype.UUID
    ResumesAt      *time.Time
}

type ResumeSubscriptionParams struct {
    TenantID       pgtype.UUID
    SubscriptionID pgtype.UUID
}

type CancelSubscriptionParams struct {
    TenantID           pgtype.UUID
    SubscriptionID     pgtype.UUID
    CancelAtPeriodEnd  bool
    CancellationReason string
}

type PortalSessionParams struct {
    TenantID  pgtype.UUID
    UserID    pgtype.UUID
    ReturnURL string
}

type SyncSubscriptionParams struct {
    TenantID               pgtype.UUID
    ProviderSubscriptionID string
    EventType              string
    EventID                string
}

// Response types
type SubscriptionDetail struct {
    ID                     pgtype.UUID
    TenantID               pgtype.UUID
    UserID                 pgtype.UUID
    Status                 string
    BillingInterval        string
    SubtotalCents          int32
    TaxCents               int32
    ShippingCents          int32
    TotalCents             int32
    Currency               string
    ProviderSubscriptionID string
    ProviderCustomerID     string
    CurrentPeriodStart     time.Time
    CurrentPeriodEnd       time.Time
    NextBillingDate        time.Time
    CancelAtPeriodEnd      bool
    CancelledAt            *time.Time
    CreatedAt              time.Time
    UpdatedAt              time.Time
    Items                  []SubscriptionItemDetail
    ShippingAddress        *AddressDetail
    PaymentMethod          *PaymentMethodDetail
}

type SubscriptionSummary struct {
    ID                pgtype.UUID
    Status            string
    BillingInterval   string
    TotalCents        int32
    Currency          string
    NextBillingDate   time.Time
    CancelAtPeriodEnd bool
    ProductName       string
    ProductImageURL   string
    CreatedAt         time.Time
}

type SubscriptionItemDetail struct {
    ID             pgtype.UUID
    ProductSKUID   pgtype.UUID
    ProductName    string
    SKU            string
    Quantity       int32
    UnitPriceCents int32
    ImageURL       string
    WeightValue    string
    Grind          string
}
```

---

## SQLc Queries

Create `sqlc/queries/subscriptions.sql`:

```sql
-- name: CreateSubscription :one
INSERT INTO subscriptions (
    tenant_id, user_id, subscription_plan_id, billing_interval, status,
    billing_customer_id, provider, provider_subscription_id,
    subtotal_cents, tax_cents, total_cents, currency,
    shipping_address_id, shipping_method_id, shipping_cents, payment_method_id,
    current_period_start, current_period_end, next_billing_date, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING *;

-- name: CreateSubscriptionItem :one
INSERT INTO subscription_items (
    tenant_id, subscription_id, product_sku_id, quantity, unit_price_cents, metadata
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions WHERE id = $1 AND tenant_id = $2;

-- name: GetSubscriptionByProviderID :one
SELECT * FROM subscriptions
WHERE provider_subscription_id = $1 AND provider = $2 AND tenant_id = $3;

-- name: ListSubscriptionsForUser :many
SELECT * FROM subscriptions
WHERE user_id = $1 AND tenant_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListActiveSubscriptionsForUser :many
SELECT * FROM subscriptions
WHERE user_id = $1 AND tenant_id = $2 AND status IN ('active', 'trialing')
ORDER BY created_at DESC;

-- name: ListSubscriptionItemsForSubscription :many
SELECT si.*, p.name as product_name, ps.sku, ps.weight_value, ps.weight_unit, ps.grind,
       pi.url as product_image_url
FROM subscription_items si
JOIN product_skus ps ON si.product_sku_id = ps.id
JOIN products p ON ps.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = true
WHERE si.subscription_id = $1 AND si.tenant_id = $2
ORDER BY si.created_at;

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions SET
    status = $3,
    current_period_start = COALESCE($4, current_period_start),
    current_period_end = COALESCE($5, current_period_end),
    next_billing_date = COALESCE($6, next_billing_date),
    cancel_at_period_end = COALESCE($7, cancel_at_period_end),
    cancelled_at = COALESCE($8, cancelled_at),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateSubscriptionProviderID :one
UPDATE subscriptions SET
    provider_subscription_id = $3, status = $4, updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateSubscriptionCancellation :one
UPDATE subscriptions SET
    cancel_at_period_end = $3,
    cancelled_at = COALESCE($4, cancelled_at),
    cancellation_reason = COALESCE($5, cancellation_reason),
    status = CASE WHEN $3 = true THEN status ELSE 'cancelled' END,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: GetBillingCustomerForUser :one
SELECT * FROM billing_customers
WHERE user_id = $1 AND tenant_id = $2 AND provider = $3;

-- name: GetDefaultPaymentMethodForUser :one
SELECT pm.* FROM payment_methods pm
JOIN billing_customers bc ON pm.billing_customer_id = bc.id
WHERE bc.user_id = $1 AND bc.tenant_id = $2 AND bc.provider = $3 AND pm.is_default = true;

-- name: CreateSubscriptionScheduleEvent :one
INSERT INTO subscription_schedule (
    tenant_id, subscription_id, event_type, status, scheduled_at, metadata
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
```

---

## Key Flows

### Create Subscription Flow

1. User selects product SKU + frequency + shipping address on storefront
2. Frontend collects payment method via Stripe Elements
3. Backend `CreateSubscription()`:
   - Validate SKU exists and is active
   - Get pricing from customer's price list
   - Get/create Stripe Customer for user
   - Create Stripe Product (if not exists for this SKU)
   - Create Stripe Price with recurring interval
   - Create Stripe Subscription
   - Store local `subscriptions` record with `provider_subscription_id`
   - Store `subscription_items` record
4. Stripe immediately charges first invoice
5. Webhook `invoice.payment_succeeded` → `CreateOrderFromSubscriptionInvoice()`
6. First order created and ready for fulfillment

### Renewal Flow (Order Creation)

**Trigger:** Stripe webhook `invoice.payment_succeeded` where `invoice.subscription != null`

1. Idempotency check via `subscription_schedule` table
2. Retrieve Stripe invoice (contains subscription ID, amount, payment intent)
3. Load local subscription by `provider_subscription_id`
4. Validate tenant_id matches
5. Load `subscription_items` for products/quantities
6. Build order items from subscription items:
   - Use `subscription_items.unit_price_cents` (locked at creation)
   - Use current product details from `product_skus`
7. Get shipping address from `subscriptions.shipping_address_id`
8. Create order via existing OrderService pattern:
   - Generate order number
   - Create order record with `subscription_id` set
   - Create order items
   - Create payment record
   - Decrement inventory
9. Create `subscription_schedule` entry (event_type = "billing", order_id set)
10. Update subscription dates (`current_period_start`, `current_period_end`, `next_billing_date`)

### Pause/Resume/Cancel Flow

1. Customer clicks "Manage Subscription" → Redirected to Stripe Customer Portal
2. Customer pauses/resumes/cancels in Stripe Portal
3. Stripe sends `customer.subscription.updated` or `customer.subscription.deleted` webhook
4. `SyncSubscriptionFromWebhook()` updates local subscription status
5. Create `subscription_schedule` entry for audit trail

---

## Webhook Events

### MVP (Required)

| Event | Handler | Action |
|-------|---------|--------|
| `invoice.payment_succeeded` | `CreateOrderFromSubscriptionInvoice()` | Create order, decrement inventory |
| `invoice.payment_failed` | Update status | Set subscription status to `past_due` |
| `customer.subscription.updated` | `SyncSubscriptionFromWebhook()` | Sync status, dates from Stripe |
| `customer.subscription.deleted` | `SyncSubscriptionFromWebhook()` | Set status to `expired` |

### Post-MVP (Nice to Have)

| Event | Action |
|-------|--------|
| `invoice.upcoming` | Send "your shipment is coming" email |
| `invoice.payment_action_required` | Send "update payment method" email |

---

## Billing Provider Interface Extensions

Add to `internal/billing/billing.go`:

```go
// Subscription methods (add to Provider interface)
CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*Subscription, error)
GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
UpdateSubscription(ctx context.Context, params UpdateSubscriptionParams) (*Subscription, error)
CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) (*Subscription, error)
PauseSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
ResumeSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)

// Price management (for subscription pricing)
CreatePrice(ctx context.Context, params CreatePriceParams) (*Price, error)

// Customer portal
CreateCustomerPortalSession(ctx context.Context, customerID, returnURL string) (*CustomerPortalSession, error)

// Invoice retrieval (for webhook processing)
GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
```

---

## Customer Portal Strategy

**MVP:** Use Stripe Customer Portal (hosted by Stripe)

**How it works:**
1. Backend generates portal session URL via `CreateCustomerPortalSession()`
2. Frontend redirects customer to Stripe-hosted page
3. Customer can: pause, resume, cancel, update payment method
4. Stripe sends webhooks for changes → local DB synced

**Stripe Portal Configuration (in Stripe Dashboard):**
- Enable: Pause subscriptions, Cancel subscriptions, Update payment method
- Disable: Update subscription items (we'll handle product changes ourselves later)

**Post-MVP:** Consider custom portal UI if customers need:
- Product swap mid-cycle
- Skip next delivery
- View upcoming deliveries

---

## Implementation Sequence

### Phase 1: Billing Provider Extensions
- [ ] Add subscription types to `internal/billing/billing.go`
- [ ] Implement Stripe subscription methods in `internal/billing/stripe.go`
- [ ] Add mock implementations for testing
- [ ] Write unit tests

### Phase 2: SQLc Queries
- [ ] Create `sqlc/queries/subscriptions.sql` with:
  - CreateSubscription, GetSubscription, GetUserSubscriptions
  - UpdateSubscriptionStatus, UpdateSubscriptionDates
  - CreateSubscriptionItem, GetSubscriptionItems
  - CreateSubscriptionScheduleEntry

### Phase 3: Subscription Service
- [ ] Create `internal/service/subscription.go`
- [ ] Implement `CreateSubscription()`
- [ ] Implement `GetSubscription()`, `GetUserSubscriptions()`
- [ ] Write unit tests with mocked billing provider

### Phase 4: Webhook Handlers
- [ ] Extend `internal/handler/webhook/stripe.go`
- [ ] Add routing for subscription webhook events
- [ ] Implement `SyncSubscriptionFromWebhook()`
- [ ] Implement `CreateOrderFromSubscriptionInvoice()`
- [ ] Test with Stripe CLI webhook forwarding

### Phase 5: Lifecycle Operations
- [ ] Implement `PauseSubscription()`, `ResumeSubscription()`
- [ ] Implement `CancelSubscription()`
- [ ] Implement `UpdateSubscription()` (for quantity changes)

### Phase 6: HTTP Handlers & UI
- [ ] Create subscription HTTP handlers
- [ ] Add Customer Portal redirect endpoint
- [ ] Create subscription UI templates (list, create, detail)
- [ ] Add "Subscribe" flow to product pages

### Phase 7: Admin UI
- [ ] Admin subscription list view
- [ ] Admin subscription detail view
- [ ] Admin actions: view orders, pause, cancel

---

## Key Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Source of truth | Stripe Billing | Simplicity, reliability, less code |
| Checkout flow | Separate from cart | Cleaner UX, simpler implementation, no mixed carts |
| Order creation | Webhook-driven | Consistency with existing checkout |
| Pricing | Locked at creation | Predictability for customers |
| Customer portal | Stripe hosted (MVP) | Fastest path to launch |
| Stripe Prices | One per subscription | Simplifies price list logic |
| Multi-item | Deferred to post-MVP | Reduces complexity |
| Trial periods | Not implemented | Standard billing from day one |
| Skip delivery | Deferred to post-MVP | Can layer on later |
| Auth requirement | RequireAuth middleware | Subscriptions need saved addresses/payment methods |

---

## Open Questions (Resolved)

| Question | Resolution |
|----------|------------|
| Multi-item subscriptions? | Deferred — single item for MVP |
| Trial periods? | No — standard billing from day one |
| Skip next delivery? | Deferred — can add post-MVP |
| Price list changes? | Subscription pricing locked at creation |
| Product discontinuation? | TBD — may need auto-pause or product swap |

---

## Files to Create/Modify

**New files:**
- `internal/service/subscription.go` — SubscriptionService implementation
- `internal/handler/storefront/subscription.go` — Customer-facing handlers
- `internal/handler/admin/subscription.go` — Admin handlers
- `sqlc/queries/subscriptions.sql` — Database queries
- `web/templates/storefront/subscriptions.html` — Customer subscription list
- `web/templates/storefront/subscribe.html` — Create subscription flow
- `web/templates/admin/subscriptions.html` — Admin subscription list

**Modified files:**
- `internal/billing/billing.go` — Add subscription interfaces
- `internal/billing/stripe.go` — Implement Stripe subscription methods
- `internal/billing/mock.go` — Add mock implementations
- `internal/handler/webhook/stripe.go` — Handle subscription webhooks
- `main.go` — Register new routes
