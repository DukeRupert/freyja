# Database Quick Reference

Quick lookup guide for Freyja database schema.

## Table Count: 41 Tables

### By Domain

| Domain | Tables | Count |
|--------|--------|-------|
| **Tenant & Identity** | tenants, users, sessions | 3 |
| **Addresses** | addresses, customer_addresses | 2 |
| **Products** | products, product_skus, product_images, product_categories, product_category_assignments, product_tags, product_tag_assignments | 7 |
| **Pricing** | price_lists, price_list_entries, user_price_lists | 3 |
| **Shopping** | carts, cart_items | 2 |
| **Orders** | orders, order_items, order_status_history | 3 |
| **Billing** | billing_customers, payment_methods, payments, webhook_events | 4 |
| **Shipping** | shipping_methods, shipping_rates, shipments, shipment_items, shipment_tracking_events | 5 |
| **Subscriptions** | subscription_plans, subscriptions, subscription_items, subscription_schedule | 4 |
| **Invoicing** | invoices, invoice_items, invoice_payments, invoice_status_history | 4 |
| **Jobs** | jobs, job_history | 2 |
| **Supporting** | product_reviews, review_helpfulness, discount_codes, discount_code_usage, email_templates | 5 |

---

## Key Tables Cheat Sheet

### Users
```sql
users
  - id, tenant_id
  - email, password_hash, email_verified
  - account_type: 'retail' | 'wholesale' | 'admin'
  - status: 'pending' | 'active' | 'suspended' | 'closed'
  - wholesale_application_status: 'pending' | 'approved' | 'rejected'
  - payment_terms: 'net_15' | 'net_30' | 'net_60'
```

### Products & SKUs
```sql
products
  - id, tenant_id
  - name, slug, description
  - origin, region, producer, process, roast_level, elevation
  - tasting_notes (TEXT[])
  - status: 'draft' | 'active' | 'archived'
  - visibility: 'public' | 'wholesale_only' | 'hidden'

product_skus
  - id, tenant_id, product_id
  - sku, weight_value, weight_unit, grind
  - base_price_cents, inventory_quantity
  - inventory_policy: 'deny' | 'allow'
```

### Price Lists
```sql
price_lists
  - id, tenant_id
  - name, list_type: 'default' | 'wholesale' | 'custom'

price_list_entries
  - price_list_id, product_sku_id
  - price_cents, compare_at_price_cents
  - is_available

user_price_lists
  - user_id, price_list_id (one per user)
```

### Carts
```sql
carts
  - id, tenant_id
  - user_id (NULL for guests)
  - session_id (for guest persistence)
  - status: 'active' | 'abandoned' | 'converted' | 'expired'

cart_items
  - cart_id, product_sku_id
  - quantity, unit_price_cents
```

### Orders
```sql
orders
  - id, tenant_id, user_id
  - order_number, order_type: 'retail' | 'wholesale' | 'subscription'
  - status: 'pending' | 'payment_processing' | 'paid' | 'processing' | 'shipped' | 'delivered' | 'cancelled' | 'refunded'
  - amounts: subtotal_cents, tax_cents, shipping_cents, discount_cents, total_cents
  - payment_id, shipping_address_id, billing_address_id
  - fulfillment_status: 'unfulfilled' | 'partial' | 'fulfilled' | 'cancelled'
```

### Billing
```sql
billing_customers
  - user_id, provider: 'stripe' | 'manual'
  - provider_customer_id

payments
  - billing_customer_id, provider, provider_payment_id
  - amount_cents, currency
  - status: 'pending' | 'processing' | 'succeeded' | 'failed' | 'cancelled' | 'refunded' | 'partially_refunded'

webhook_events
  - provider, provider_event_id (UNIQUE)
  - event_type, status: 'pending' | 'processing' | 'processed' | 'failed'
  - payload (JSONB)
```

### Subscriptions
```sql
subscriptions
  - id, tenant_id, user_id
  - billing_interval: 'weekly' | 'biweekly' | 'monthly' | 'every_6_weeks' | 'every_2_months'
  - status: 'trial' | 'active' | 'paused' | 'past_due' | 'cancelled' | 'expired'
  - provider, provider_subscription_id
  - next_billing_date

subscription_items
  - subscription_id, product_sku_id
  - quantity, unit_price_cents

subscription_schedule
  - subscription_id
  - event_type: 'billing' | 'renewal' | 'skip' | 'pause' | 'resume' | 'cancel' | 'payment_failed'
  - scheduled_at, processed_at
```

### Invoices
```sql
invoices
  - id, tenant_id, user_id
  - invoice_number
  - status: 'draft' | 'sent' | 'viewed' | 'partial' | 'paid' | 'overdue' | 'cancelled' | 'void'
  - amounts: subtotal_cents, tax_cents, total_cents, paid_cents, balance_cents
  - payment_terms: 'net_15' | 'net_30' | 'net_60' | 'due_on_receipt'
  - due_date

invoice_items
  - invoice_id
  - item_type: 'product' | 'shipping' | 'discount' | 'custom'
  - product_sku_id, order_id (nullable)

invoice_payments
  - invoice_id, payment_id
  - amount_cents
  - Triggers auto-update of invoice.balance_cents
```

### Jobs
```sql
jobs
  - job_type, queue, status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled'
  - payload (JSONB), priority
  - max_retries, retry_count
  - scheduled_at

job_history
  - job_id, job_type, status, duration_ms
  - Auto-populated by trigger on job completion
```

---

## Common Status Enums

### User Status
- `pending`: Awaiting email verification
- `active`: Normal account
- `suspended`: Temporarily disabled
- `closed`: Permanently closed

### Order Status
- `pending`: Created, not paid
- `payment_processing`: Payment in progress
- `paid`: Payment successful
- `processing`: Being fulfilled
- `shipped`: Sent to customer
- `delivered`: Confirmed delivery
- `cancelled`: Cancelled before shipment
- `refunded`: Refunded after payment

### Payment Status
- `pending`: Created, not processed
- `processing`: Being processed
- `succeeded`: Completed successfully
- `failed`: Payment failed
- `cancelled`: Cancelled before completion
- `refunded`: Fully refunded
- `partially_refunded`: Partially refunded

### Subscription Status
- `trial`: In trial period
- `active`: Active subscription
- `paused`: Temporarily paused
- `past_due`: Payment failed, retrying
- `cancelled`: Cancelled by customer
- `expired`: Ended naturally

### Invoice Status
- `draft`: Not yet sent
- `sent`: Sent to customer
- `viewed`: Customer viewed
- `partial`: Partially paid
- `paid`: Fully paid
- `overdue`: Past due date
- `cancelled`: Cancelled
- `void`: Voided (invalid)

---

## Foreign Key Relationships

### User-Centric
```
users
├─> sessions (1:N)
├─> customer_addresses (1:N) → addresses
├─> user_price_lists (1:1) → price_lists
├─> billing_customers (1:N)
├─> carts (1:N)
├─> orders (1:N)
├─> subscriptions (1:N)
└─> invoices (1:N)
```

### Product-Centric
```
products
├─> product_skus (1:N)
│   ├─> price_list_entries (1:N)
│   ├─> cart_items (1:N)
│   ├─> order_items (1:N)
│   └─> subscription_items (1:N)
├─> product_images (1:N)
└─> product_reviews (1:N)
```

### Order-Centric
```
orders
├─> order_items (1:N)
├─> order_status_history (1:N)
├─> shipments (1:N)
│   ├─> shipment_items (1:N)
│   └─> shipment_tracking_events (1:N)
├─> payments (N:1)
└─> invoices (via invoice_items)
```

---

## Unique Constraints

| Table | Constraint | Columns |
|-------|------------|---------|
| tenants | slug | `slug` |
| users | email per tenant | `(tenant_id, email)` |
| products | slug per tenant | `(tenant_id, slug)` |
| product_skus | sku per tenant | `(tenant_id, sku)` |
| price_lists | name per tenant | `(tenant_id, name)` |
| price_list_entries | sku per list | `(price_list_id, product_sku_id)` |
| user_price_lists | one list per user | `(user_id)` |
| cart_items | sku per cart | `(cart_id, product_sku_id)` |
| orders | order number per tenant | `(tenant_id, order_number)` |
| billing_customers | provider per user | `(user_id, provider)` |
| payments | provider payment ID | `(provider, provider_payment_id)` |
| webhook_events | provider event ID | `(provider, provider_event_id)` |
| subscriptions | provider subscription ID | `(provider, provider_subscription_id)` |
| subscription_items | sku per subscription | `(subscription_id, product_sku_id)` |
| invoices | invoice number per tenant | `(tenant_id, invoice_number)` |
| email_templates | template type per tenant | `(tenant_id, template_type)` |
| discount_codes | code per tenant | `(tenant_id, code)` |
| product_reviews | one review per user per product | `(user_id, product_id)` |

---

## Critical Indexes

### For Multi-Tenant Queries
```sql
-- Every table (except tenants)
CREATE INDEX idx_<table>_tenant_id ON <table>(tenant_id);
```

### For User Lookups
```sql
-- Fast email login
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

-- Fast session lookup
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
```

### For Product Browsing
```sql
-- Active products by tenant
CREATE INDEX idx_products_status ON products(tenant_id, status) WHERE status = 'active';

-- SKU lookup
CREATE INDEX idx_product_skus_sku ON product_skus(tenant_id, sku);
```

### For Order Management
```sql
-- Pending orders
CREATE INDEX idx_orders_status ON orders(tenant_id, status);

-- Unfulfilled orders
CREATE INDEX idx_orders_pending_fulfillment ON orders(tenant_id, status, fulfillment_status)
    WHERE status = 'paid' AND fulfillment_status = 'unfulfilled';
```

### For Job Queue
```sql
-- Efficient job dequeue
CREATE INDEX idx_jobs_queue ON jobs(queue, status, priority, scheduled_at)
    WHERE status = 'pending';
```

### For Webhook Idempotency
```sql
-- Fast duplicate detection
CREATE INDEX idx_webhook_events_provider ON webhook_events(provider, provider_event_id);
```

---

## Triggers Summary

| Trigger | Purpose | Fires On |
|---------|---------|----------|
| `update_*_updated_at` | Auto-update timestamp | All tables on UPDATE |
| `update_cart_last_activity` | Track cart activity | cart_items INSERT/UPDATE/DELETE |
| `log_order_status_change` | Audit order status | orders UPDATE (status change) |
| `log_invoice_status_change` | Audit invoice status | invoices UPDATE (status change) |
| `update_invoice_balance` | Recalculate balance | invoice_payments INSERT/UPDATE/DELETE |
| `archive_completed_job` | Move to history | jobs UPDATE (to completed/failed) |
| `update_review_helpfulness_counts` | Update counts | review_helpfulness INSERT/UPDATE/DELETE |
| `increment_discount_code_usage` | Track usage | discount_code_usage INSERT |

---

## JSONB Metadata Fields

Many tables have `metadata JSONB NOT NULL DEFAULT '{}'` for extensibility:

- `tenants.settings`: Tenant-specific configuration
- `users.metadata`: Custom user data
- `products.metadata`: Product-specific attributes
- `orders.metadata`: Custom order data
- `payments.metadata`: Provider-specific payment data
- `webhook_events.payload`: Full webhook payload
- `jobs.payload`: Job parameters

**Usage:**
```sql
-- Store custom data
UPDATE products
SET metadata = jsonb_set(metadata, '{custom_field}', '"value"')
WHERE id = $1 AND tenant_id = $2;

-- Query by JSONB field
SELECT * FROM products
WHERE tenant_id = $1
  AND metadata->>'custom_field' = 'value';
```

---

## Coffee-Specific Fields

### Products Table
- `origin`: Country (e.g., "Ethiopia")
- `region`: Growing region (e.g., "Yirgacheffe")
- `producer`: Farm/cooperative name
- `process`: Processing method (washed, natural, honey, etc.)
- `roast_level`: Light, medium, dark
- `elevation_min`, `elevation_max`: Growing elevation in meters
- `variety`: Coffee variety (Heirloom, Bourbon, etc.)
- `harvest_year`: Year harvested
- `tasting_notes`: Array of strings (e.g., ['blueberry', 'chocolate', 'caramel'])

### Product SKUs Table
- `weight_value`: Numeric weight (e.g., 12, 5)
- `weight_unit`: 'oz' | 'lb' | 'g' | 'kg'
- `grind`: Grind size (whole_bean, coarse, medium, fine, espresso, etc.)

---

## Multi-Tenant Query Pattern

**Always include tenant_id:**

```sql
-- ✅ CORRECT
SELECT * FROM products
WHERE tenant_id = $1 AND status = 'active';

-- ❌ WRONG (sees all tenants' data)
SELECT * FROM products
WHERE status = 'active';
```

**sqlc query example:**
```sql
-- name: GetActiveProducts :many
SELECT *
FROM products
WHERE tenant_id = $1
  AND status = 'active'
ORDER BY created_at DESC;
```

Generated Go function:
```go
func (q *Queries) GetActiveProducts(ctx context.Context, tenantID uuid.UUID) ([]Product, error)
```

---

## Migration Commands

```bash
# Apply all migrations
make migrate

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_feature

# Check status
goose -dir migrations postgres "user=freyja password=password dbname=freyja sslmode=disable" status

# Apply specific version
goose -dir migrations postgres "..." up-to 5
```

---

## See Also

- **Full Documentation**: `/planning/DATABASE_SCHEMA.md`
- **ERD Diagrams**: `/planning/DATABASE_ERD.md`
- **Migration Files**: `/migrations/*.sql`
- **Migration Guide**: `/migrations/README.md`
