# Freyja Database Entity Relationship Diagram

This document provides a visual representation of the database schema relationships.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           TENANTS                                │
│                      (Multi-tenant root)                         │
└──────────────────────┬──────────────────────────────────────────┘
                       │
       ┌───────────────┼───────────────────┬──────────────────┐
       │               │                   │                  │
       ▼               ▼                   ▼                  ▼
   IDENTITY       PRODUCTS            SETTINGS          BACKGROUND
  (Users,         (Catalog,           (Shipping,          (Jobs,
  Sessions,       Pricing,            Email                Job
  Addresses)      Inventory)          Templates)           History)
       │               │
       ▼               ▼
   SHOPPING        FULFILLMENT
   (Carts,         (Orders,
    Items)          Shipments)
       │               │
       ▼               ▼
   BILLING         INVOICING
  (Payments,       (Invoices,
   Webhooks)       Payments)
       │
       ▼
 SUBSCRIPTIONS
  (Recurring
   Orders)
```

---

## Core Entity Relationships

### 1. Multi-Tenancy Foundation

```
tenants
├── users (1:N)
├── products (1:N)
├── price_lists (1:N)
├── orders (1:N)
├── invoices (1:N)
├── subscriptions (1:N)
└── ... (all other tables)
```

**Every table references tenants except `tenants` itself.**

---

### 2. Identity & Access

```
tenants
  │
  ├─> users
  │     ├─> sessions (1:N)
  │     ├─> customer_addresses (1:N)
  │     │     └─> addresses (N:1)
  │     ├─> user_price_lists (1:1) → price_lists
  │     ├─> billing_customers (1:N)
  │     │     ├─> payment_methods (1:N)
  │     │     └─> payments (1:N)
  │     ├─> carts (1:N)
  │     ├─> orders (1:N)
  │     ├─> subscriptions (1:N)
  │     ├─> invoices (1:N)
  │     └─> product_reviews (1:N)
  │
  └─> addresses (shared, linked via customer_addresses)
```

**Key Points:**
- Users can have multiple saved addresses (junction: `customer_addresses`)
- Each user can have one assigned price list
- Guest carts: `user_id IS NULL`, linked via `session_id`

---

### 3. Product Catalog & Pricing

```
tenants
  │
  ├─> products
  │     ├─> product_skus (1:N)
  │     │     ├─> price_list_entries (1:N)
  │     │     ├─> cart_items (1:N)
  │     │     ├─> order_items (1:N)
  │     │     └─> subscription_items (1:N)
  │     ├─> product_images (1:N)
  │     ├─> product_category_assignments (N:M) ↔ product_categories
  │     ├─> product_tag_assignments (N:M) ↔ product_tags
  │     └─> product_reviews (1:N)
  │
  ├─> price_lists
  │     ├─> price_list_entries (1:N) → product_skus
  │     └─> user_price_lists (1:N) → users
  │
  └─> product_categories (hierarchical, self-referencing via parent_id)
```

**Key Points:**
- Products have multiple SKUs (variants by weight + grind)
- SKUs have base prices, overridden by price list entries
- Price lists assigned to users determine what they see

---

### 4. Shopping & Cart

```
tenants
  │
  └─> carts
        ├─> cart_items (1:N)
        │     └─> product_skus (N:1)
        ├─> users (N:1, nullable for guest carts)
        └─> sessions (N:1, for guest cart persistence)
```

**Key Points:**
- Guest carts: `user_id IS NULL`, `session_id` set
- User carts: `user_id` set, survive logout
- Converted carts link to orders via `converted_to_order_id`

---

### 5. Orders & Fulfillment

```
tenants
  │
  └─> orders
        ├─> order_items (1:N)
        │     └─> product_skus (N:1)
        ├─> order_status_history (1:N) [audit trail]
        ├─> shipments (1:N)
        │     ├─> shipment_items (1:N) → order_items
        │     └─> shipment_tracking_events (1:N)
        ├─> users (N:1)
        ├─> addresses (N:1 for shipping, N:1 for billing)
        ├─> payments (N:1)
        ├─> subscriptions (N:1, nullable)
        └─> invoices (N:1 via invoice_items)
```

**Key Points:**
- Orders can have multiple shipments (partial fulfillment)
- Shipment items link specific order items to shipments
- Order status history provides audit trail

---

### 6. Billing & Payments

```
tenants
  │
  └─> users
        │
        └─> billing_customers (1:N, one per provider)
              ├─> payment_methods (1:N)
              └─> payments (1:N)
                    ├─> orders (1:N)
                    ├─> subscriptions (1:N)
                    └─> invoice_payments (1:N) → invoices

webhook_events (tracks idempotency)
  └─> provider_event_id [UNIQUE]
```

**Key Points:**
- `billing_customers` abstracts payment providers (Stripe initially)
- `webhook_events` ensures idempotent webhook processing
- Payments link to orders, subscriptions, or invoices

---

### 7. Subscriptions

```
tenants
  │
  ├─> subscription_plans (templates)
  │     └─> default_product_sku_id → product_skus
  │
  └─> subscriptions (customer instances)
        ├─> subscription_items (1:N) → product_skus
        ├─> subscription_schedule (1:N) [upcoming events]
        ├─> users (N:1)
        ├─> addresses (N:1)
        ├─> billing_customers (N:1)
        ├─> payment_methods (N:1)
        └─> orders (1:N) [generated on billing]
```

**Key Points:**
- Plans are templates; subscriptions are instances
- Subscriptions can have multiple products (via `subscription_items`)
- Schedule tracks upcoming billing, skip, pause events
- Each billing event generates an order

---

### 8. Invoicing (Wholesale)

```
tenants
  │
  └─> invoices
        ├─> invoice_items (1:N)
        │     ├─> product_skus (N:1, nullable)
        │     └─> orders (N:1, nullable) [for order consolidation]
        ├─> invoice_payments (1:N)
        │     └─> payments (N:1, nullable)
        ├─> invoice_status_history (1:N) [audit trail]
        ├─> users (N:1)
        ├─> addresses (N:1 for billing)
        └─> billing_customers (N:1)
```

**Key Points:**
- Invoices can consolidate multiple orders (wholesale)
- Invoice items can be products, shipping, discounts, or custom
- Payments trigger automatic balance/status updates
- Supports net terms (Net 15, Net 30, etc.)

---

### 9. Shipping

```
tenants
  │
  ├─> shipping_methods
  │     └─> shipping_rates (1:N, cached rates)
  │
  └─> orders
        └─> shipments (1:N)
              ├─> shipping_methods (N:1)
              ├─> shipment_items (1:N) → order_items
              └─> shipment_tracking_events (1:N)
```

**Key Points:**
- Shipping methods can be manual or provider-integrated
- Rates cached for performance
- Tracking events provide delivery status history

---

### 10. Background Jobs

```
tenants
  │
  └─> jobs
        └─> job_history (on completion) [archived jobs]
```

**Key Points:**
- Database-backed queue (no Redis/RabbitMQ needed for MVP)
- Jobs auto-archived to history on completion/failure
- Supports priority, retry, scheduling

---

### 11. Supporting Features

```
tenants
  │
  ├─> discount_codes
  │     └─> discount_code_usage (1:N)
  │           ├─> users (N:1)
  │           └─> orders (N:1)
  │
  ├─> product_reviews
  │     ├─> review_helpfulness (1:N)
  │     ├─> users (N:1)
  │     └─> products (N:1)
  │
  └─> email_templates (customizable transactional emails)
```

---

## Data Flow Examples

### Retail Purchase Flow

```
1. Customer browses products (filtered by price_list)
2. Adds items to cart → cart_items
3. Checkout:
   - Creates billing_customer (if first time)
   - Creates payment → payments
   - Payment succeeds
4. Creates order → orders, order_items
5. Creates shipment → shipments, shipment_items
6. Updates order status → order_status_history
7. Sends confirmation → email_templates
```

### Subscription Billing Flow

```
1. Customer creates subscription
   - subscription → subscriptions
   - Products → subscription_items
   - Links billing_customer, payment_method
2. Scheduled billing event → subscription_schedule
3. Worker processes event:
   - Creates payment → payments
   - Payment succeeds
4. Creates order → orders, order_items
5. Updates subscription → next_billing_date
6. Sends renewal confirmation
```

### Wholesale Invoice Flow

```
1. Customer places multiple orders
   - orders → order_items (accumulate over billing period)
2. End of billing cycle:
   - Creates invoice → invoices
   - Consolidates orders → invoice_items
   - Status: draft
3. Admin reviews and sends:
   - Status: sent
   - Links billing_customer
4. Payment received:
   - Creates payment → payments
   - Links → invoice_payments
   - Triggers auto-update: balance, status → paid
5. Sends receipt
```

### Webhook Processing Flow

```
1. Stripe sends webhook (e.g., payment_intent.succeeded)
2. Check webhook_events for provider_event_id
   - If exists: ignore (idempotency)
   - If new: insert with status=pending
3. Process event:
   - Update payment status
   - Trigger downstream actions (create order, send email)
4. Update webhook_event status=processed
```

---

## Index Strategy Summary

### Primary Indexes (all tables)
- `id` (PRIMARY KEY, UUID)
- `tenant_id` (foreign key, multi-tenant filtering)
- `created_at` (time-based queries, pagination)
- `updated_at` (change tracking)

### Foreign Key Indexes
All foreign key columns indexed for join performance.

### Status Indexes (partial)
- `WHERE status = 'active'` (users, products, price_lists)
- `WHERE status = 'pending'` (jobs, webhook_events)
- `WHERE status IN ('active', 'trial')` (subscriptions)

### Lookup Indexes
- Business keys: `order_number`, `invoice_number`, `sku`, `code`
- Provider IDs: `provider_customer_id`, `provider_event_id`
- Auth: `email`, `token_hash`
- Tracking: `tracking_number`

### Composite Indexes
- `(tenant_id, user_id)` (user-scoped queries)
- `(tenant_id, status, created_at)` (filtered listings)
- `(queue, status, priority, scheduled_at)` (job dequeue)

---

## Constraint Summary

### Unique Constraints
- `(tenant_id, email)` on users
- `(tenant_id, slug)` on products, categories
- `(tenant_id, sku)` on product_skus
- `(tenant_id, order_number)` on orders
- `(tenant_id, invoice_number)` on invoices
- `(provider, provider_event_id)` on webhook_events (idempotency)
- `(user_id)` on user_price_lists (one price list per user)

### Check Constraints
- Status enums (e.g., `CHECK (status IN ('active', 'paused', 'cancelled'))`)
- Positive quantities (e.g., `CHECK (quantity > 0)`)
- Rating range (e.g., `CHECK (rating >= 1 AND rating <= 5)`)

### Foreign Key Cascade Rules
- `ON DELETE CASCADE`: Most tenant_id relationships (delete tenant → delete all data)
- `ON DELETE RESTRICT`: Critical references (e.g., product_sku_id in orders — can't delete SKU with active orders)
- `ON DELETE SET NULL`: Optional relationships (e.g., cart_id on orders — cart can be deleted after order)

---

## Multi-Tenant Data Isolation

### Row-Level Security (RLS) Approach

All queries must include `tenant_id` in WHERE clause:

```sql
-- CORRECT: Filtered by tenant
SELECT * FROM products WHERE tenant_id = $1 AND status = 'active';

-- WRONG: Missing tenant_id (will see all tenants' data)
SELECT * FROM products WHERE status = 'active';
```

**sqlc helps enforce this:**
- Query parameters always include `tenant_id`
- Generated code requires `tenant_id` argument
- Type safety prevents accidental cross-tenant queries

### Tenant Isolation Verification

```sql
-- Count tables missing tenant_id (should be 1: tenants)
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
  AND table_name NOT IN (
    SELECT table_name
    FROM information_schema.columns
    WHERE column_name = 'tenant_id'
  );
-- Expected: 1 (tenants table)
```

---

## Performance Considerations

### Query Patterns

**Efficient (indexed):**
```sql
-- Filtered by tenant + indexed column
SELECT * FROM orders WHERE tenant_id = $1 AND status = 'pending';

-- Join with tenant filtering on both sides
SELECT o.*, u.email
FROM orders o
JOIN users u ON u.id = o.user_id AND u.tenant_id = $1
WHERE o.tenant_id = $1;
```

**Inefficient (missing indexes):**
```sql
-- Full table scan (no tenant_id)
SELECT * FROM products WHERE name ILIKE '%coffee%';

-- Should add: WHERE tenant_id = $1 AND name ILIKE '%coffee%'
```

### Large Table Considerations

As tables grow, consider:

1. **Partitioning** (PostgreSQL 10+):
   - Partition `orders` by created_at (monthly/yearly)
   - Partition `job_history` by created_at (monthly)

2. **Archival**:
   - Move old orders to `orders_archive` table
   - Move completed jobs to `job_history` (already done via trigger)

3. **Materialized Views**:
   - Pre-aggregate sales metrics
   - Cache expensive JOIN queries

---

## Next Steps

1. **Run migrations**: `make migrate`
2. **Verify schema**: Connect to DB and inspect tables
3. **Write sqlc queries**: Start with user authentication and product catalog
4. **Generate code**: `make sqlc-gen`
5. **Seed data**: Create development tenant with sample products

---

## References

- Full schema documentation: `/planning/DATABASE_SCHEMA.md`
- Migration files: `/migrations/*.sql`
- Migration guide: `/migrations/README.md`
