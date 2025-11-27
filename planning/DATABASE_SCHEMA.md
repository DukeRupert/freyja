# Freyja Database Schema Documentation

## Overview

The Freyja database is designed as a **multi-tenant SaaS architecture** where all coffee roasters (tenants) share a single PostgreSQL database. Row-level isolation is achieved through `tenant_id` foreign keys on all tables except the root `tenants` table.

**Key Design Principles:**
- UUID primary keys for security and distributed system compatibility
- Timestamps (`created_at`, `updated_at`) on all tables via triggers
- JSONB metadata fields for extensibility
- Provider abstraction (Stripe initially, but designed for flexibility)
- Audit trails for critical state changes (orders, invoices, subscriptions)
- Webhook idempotency via unique constraints

---

## Database Tables (41 total)

### 1. Tenant & Identity (3 tables)

#### `tenants`
Root table for multi-tenancy. Each tenant is a coffee roaster using the platform.
- **No `tenant_id` column** (this is the root)
- Fields: name, slug, email, business details, settings (JSONB), status, trial period
- Indexed: slug (for subdomain routing), status

#### `users`
Customer accounts (retail, wholesale, admin).
- Fields: email, password_hash, account_type, profile, wholesale application, payment terms
- Unique constraint: `(tenant_id, email)`
- Indexed: tenant_id, email, account_type, status, wholesale_application_status

#### `sessions`
Authentication sessions for users.
- Fields: token_hash, user_id, IP, user agent, expires_at
- Indexed: token_hash (for fast lookup), user_id, expires_at

---

### 2. Addresses (2 tables)

#### `addresses`
Physical addresses for shipping and billing.
- Fields: address lines, city, state, postal code, country, phone, email, validation status
- Type: shipping, billing, or both

#### `customer_addresses`
Junction table linking users to their saved addresses.
- Fields: user_id, address_id, is_default_shipping, is_default_billing, label
- Indexed: user_id, defaults

---

### 3. Product Catalog (6 tables)

#### `products`
Coffee products (base offerings like "Ethiopia Yirgacheffe").
- **Coffee-specific fields**: origin, region, producer, process, roast_level, elevation, variety, harvest_year, tasting_notes (array)
- Fields: name, slug, description, visibility (public/wholesale_only/hidden), status
- Indexed: tenant_id, slug, status, visibility, origin, roast_level

#### `product_skus`
Product variants by weight and grind (e.g., "12oz whole bean", "5lb ground").
- Fields: sku, weight (value + unit), grind, base_price_cents, inventory, weight for shipping
- Unique constraint: `(tenant_id, sku)`
- Indexed: tenant_id, product_id, sku, is_active, low_stock

#### `product_images`
Product images with ordering.
- Fields: url, alt_text, dimensions, file size, sort_order, is_primary
- Indexed: product_id, is_primary

#### `product_categories`
Hierarchical categories (self-referencing).
- Fields: name, slug, parent_id, sort_order, SEO fields
- Indexed: tenant_id, parent_id, is_active

#### `product_category_assignments`
Junction table: products ↔ categories.

#### `product_tags`
Flexible product tags.
- Fields: name, slug

#### `product_tag_assignments`
Junction table: products ↔ tags.

---

### 4. Pricing (3 tables)

#### `price_lists`
Named pricing tiers (e.g., "Retail", "Café Tier 1", "Restaurant Tier 2").
- Fields: name, description, list_type (default/wholesale/custom), is_active
- Indexed: tenant_id, is_active, list_type

#### `price_list_entries`
Per-SKU pricing for each price list.
- Fields: price_list_id, product_sku_id, price_cents, compare_at_price_cents, is_available
- Unique constraint: `(price_list_id, product_sku_id)`
- Indexed: price_list_id, product_sku_id, is_available

#### `user_price_lists`
Assigns price lists to users.
- Fields: user_id, price_list_id, assigned_by, notes
- Unique constraint: one price list per user

---

### 5. Shopping Cart (2 tables)

#### `carts`
Shopping carts for guests and authenticated users.
- Fields: user_id (NULL for guests), session_id, status, last_activity_at, converted_to_order_id
- Indexed: tenant_id, user_id, session_id, status, last_activity_at

#### `cart_items`
Line items in carts.
- Fields: cart_id, product_sku_id, quantity, unit_price_cents
- Unique constraint: `(cart_id, product_sku_id)`
- Trigger: updates cart.last_activity_at on changes

---

### 6. Orders (3 tables)

#### `orders`
Customer orders (retail, wholesale, subscription).
- Fields: order_number, order_type, status, amounts (subtotal, tax, shipping, discount, total), payment_id, addresses, fulfillment_status, timestamps
- Status flow: pending → payment_processing → paid → processing → shipped → delivered
- Indexed: tenant_id, user_id, order_number, status, payment_status, fulfillment_status, dates

#### `order_items`
Line items in orders.
- Fields: order_id, product_sku_id, product snapshot (name, sku, variant), quantity, pricing, fulfillment_status

#### `order_status_history`
Audit trail for order status changes.
- Trigger: automatically logs status changes from orders table

---

### 7. Billing (4 tables)

#### `billing_customers`
Maps users to payment provider customer IDs.
- Fields: user_id, provider (stripe/manual), provider_customer_id, metadata (JSONB)
- Unique constraint: `(user_id, provider)`
- Indexed: user_id, provider

#### `payment_methods`
Stored payment methods (cards, bank accounts).
- Fields: billing_customer_id, provider, provider_payment_method_id, method_type, display info, is_default
- Indexed: billing_customer_id, is_default

#### `payments`
Payment transactions and status.
- Fields: billing_customer_id, provider, provider_payment_id, amount, status, payment_method_id, failure details, refund info, timestamps
- Status: pending → processing → succeeded/failed/cancelled/refunded
- Indexed: billing_customer_id, provider, status, created_at

#### `webhook_events`
Incoming webhooks for idempotent processing.
- Fields: provider, provider_event_id, event_type, status, payload (JSONB), error info, retry_count
- Unique constraint: `(provider, provider_event_id)` — prevents duplicate processing
- Indexed: provider, status, created_at

---

### 8. Shipping (5 tables)

#### `shipping_methods`
Available shipping options (manual or provider-integrated).
- Fields: name, code, provider (manual/easypost/shippo), flat_rate_cents, free_shipping_threshold, estimated delivery days
- Indexed: tenant_id, code, is_active

#### `shipping_rates`
Cached shipping rates from providers.
- Fields: shipping_method_id, origin/destination postal codes, weight, rate_cents, valid_until

#### `shipments`
Shipment tracking for orders.
- Fields: order_id, shipment_number, carrier, tracking_number, status, costs, package dimensions, provider integration, timestamps
- Status: pending → label_created → in_transit → delivered
- Indexed: order_id, tracking_number, status, created_at

#### `shipment_items`
Links order items to shipments (supports partial shipments).
- Fields: shipment_id, order_item_id, quantity

#### `shipment_tracking_events`
History of tracking status updates.
- Fields: shipment_id, status, message, location, event_at

---

### 9. Subscriptions (4 tables)

#### `subscription_plans`
Templates for recurring subscriptions.
- Fields: name, billing_interval (weekly/biweekly/monthly/etc.), default_product_sku_id, price_cents, trial_period_days, is_active

#### `subscriptions`
Customer subscription instances.
- Fields: user_id, subscription_plan_id, billing_interval, status, billing_customer_id, provider integration, amounts, addresses, payment_method_id, trial period, billing dates, cancellation info
- Status: trial → active → paused → past_due → cancelled → expired
- Indexed: user_id, status, next_billing_date, provider

#### `subscription_items`
Products included in subscriptions.
- Fields: subscription_id, product_sku_id, quantity, unit_price_cents
- Unique constraint: `(subscription_id, product_sku_id)`

#### `subscription_schedule`
Upcoming and past subscription events (billing, skip, pause, cancel).
- Fields: subscription_id, event_type, status, order_id, payment_id, error info, scheduled_at, processed_at
- Indexed: subscription_id, status, scheduled_at

---

### 10. Invoicing (4 tables)

#### `invoices`
Wholesale billing invoices (net-term payment).
- Fields: user_id, invoice_number, status, amounts (subtotal, tax, shipping, total, paid, balance), payment_terms (net_15/net_30), due_date, provider integration, addresses, timestamps
- Status: draft → sent → viewed → partial → paid → overdue
- Indexed: user_id, invoice_number, status, due_date, provider
- Triggers: automatic status updates on payment, balance calculation

#### `invoice_items`
Line items on invoices.
- Fields: invoice_id, item_type (product/shipping/discount/custom), product_sku_id, order_id, description, quantity, pricing

#### `invoice_payments`
Payments applied to invoices.
- Fields: invoice_id, payment_id, amount_cents, payment_method, payment_reference, notes, payment_date
- Trigger: updates invoice.paid_cents and invoice.balance_cents

#### `invoice_status_history`
Audit trail for invoice status changes.
- Trigger: automatically logs status changes from invoices table

---

### 11. Background Jobs (2 tables)

#### `jobs`
Database-backed job queue.
- Fields: job_type, queue, status, payload (JSONB), priority, retry config, scheduling, worker_id, error info, timeout
- Status: pending → processing → completed/failed/cancelled
- Indexed: queue + status + priority (for job selection), status, scheduled_at

#### `job_history`
Completed/failed jobs for analysis.
- Fields: job_id, job_type, status, duration_ms, error info, timestamps
- Trigger: automatically archives completed/failed jobs

---

### 12. Supporting Features (7 tables)

#### `product_reviews`
Customer product reviews with moderation.
- Fields: product_id, user_id, order_id, rating (1-5), title, review_text, status (pending/approved/rejected), moderation info, helpful counts, is_verified_purchase
- Unique constraint: `(user_id, product_id)`
- Indexed: product_id, status

#### `review_helpfulness`
User votes on review helpfulness.
- Fields: review_id, user_id, is_helpful
- Trigger: updates product_reviews.helpful_count and not_helpful_count

#### `discount_codes`
Promotional discount codes.
- Fields: code, discount_type (percentage/fixed_amount/free_shipping), discount_value, applies_to (order/product/category), minimum_order, usage limits, validity dates
- Indexed: code, is_active

#### `discount_code_usage`
Tracking of discount code redemptions.
- Fields: discount_code_id, user_id, order_id, discount_amount_cents
- Trigger: increments discount_codes.usage_count

#### `email_templates`
Customizable transactional email templates.
- Fields: template_type (order_confirmation/shipping_confirmation/etc.), subject, body_html, body_text, is_active
- Unique constraint: `(tenant_id, template_type)`

---

## Multi-Tenancy Strategy

**Row-Level Isolation:**
- Every table (except `tenants`) has a `tenant_id` column referencing `tenants(id)`
- All queries **must** include `tenant_id` in WHERE clauses
- sqlc helps enforce this pattern through type-safe query generation
- Foreign key cascades ensure data integrity on tenant deletion

**Benefits:**
- Simple deployment (one database)
- Easy cross-tenant analytics
- Shared infrastructure reduces costs

**Tradeoffs:**
- Must be vigilant about tenant_id in queries (mitigated by sqlc)
- Less isolation than schema-per-tenant (acceptable for MVP)

---

## Index Strategy

**Always Indexed:**
1. `tenant_id` on every table (enables row-level filtering)
2. Foreign keys (enables efficient joins)
3. `created_at` (supports time-based pagination)

**Partial Indexes** (for common filters):
- `WHERE status = 'active'` on users, price_lists, products
- `WHERE status = 'pending'` on jobs, webhook_events
- `WHERE expires_at > NOW()` on sessions, shipping_rates

**Composite Indexes** (for common query patterns):
- `(tenant_id, user_id)` on user-scoped tables
- `(tenant_id, status, created_at)` on filtered listings
- `(queue, status, priority, scheduled_at)` on jobs for efficient dequeue

**Lookup Indexes:**
- Unique identifiers: order_number, invoice_number, sku, code
- Provider IDs: provider_customer_id, provider_event_id, tracking_number
- Email addresses, slugs

---

## Key Design Patterns

### 1. Provider Abstraction
- `billing_customers` maps users to payment providers (Stripe initially)
- `provider` and `provider_*_id` columns separate internal IDs from external
- JSONB `metadata` stores provider-specific data
- **Benefit:** Easy to support multiple payment processors later

### 2. Webhook Idempotency
- `webhook_events` table with unique constraint on `(provider, provider_event_id)`
- Processing flow: check existence → insert → process → update status
- **Benefit:** Safely handle duplicate webhook deliveries

### 3. Status Enums + History
- Main tables have `status` VARCHAR with CHECK constraints
- Separate `*_status_history` tables log transitions
- Triggers automatically populate history on status changes
- **Benefit:** Audit trail without manual logging

### 4. Product-SKU Split
- `products` = coffee offering (e.g., "Ethiopia Yirgacheffe")
- `product_skus` = purchasable variants (12oz whole bean, 5lb ground, etc.)
- **Benefit:** Flexible inventory and pricing per variant

### 5. Price List System
- `price_lists` = named tiers (Retail, Café Tier 1, etc.)
- `price_list_entries` = per-SKU pricing override
- `user_price_lists` = assignment to customers
- **Benefit:** Supports retail + multiple wholesale pricing tiers

### 6. Guest vs. Authenticated Carts
- Single `carts` table for both
- Guest carts: `user_id IS NULL`, linked by `session_id`
- User carts: `user_id` set, survives logout
- **Benefit:** Avoids duplicate cart systems

### 7. Invoice Balance Auto-Update
- `invoice_payments` trigger recalculates `invoices.paid_cents` and `balance_cents`
- Status automatically updates to 'paid' or 'partial'
- **Benefit:** Balance always accurate, no manual sync

### 8. Job Queue Archival
- Completed/failed jobs moved to `job_history` via trigger
- Keeps `jobs` table small for efficient queue operations
- **Benefit:** Fast job dequeue, historical analysis available

---

## Migration Order

Migrations are ordered by dependencies:

1. **00001**: `tenants` (root table, UUID extension)
2. **00002**: `users` (depends on tenants)
3. **00003**: `sessions` (depends on users)
4. **00004**: `addresses` + `customer_addresses` (depends on users)
5. **00005**: `products` + `product_skus` + `product_images` (depends on tenants)
6. **00006**: `price_lists` + `price_list_entries` + `user_price_lists` (depends on users, product_skus)
7. **00007**: `carts` + `cart_items` (depends on users, sessions, product_skus)
8. **00008**: `billing_customers` + `payment_methods` + `payments` + `webhook_events` (depends on users)
9. **00009**: `orders` + `order_items` + `order_status_history` (depends on users, addresses, product_skus, payments)
10. **00010**: `shipping_methods` + `shipping_rates` + `shipments` + `shipment_items` + `shipment_tracking_events` (depends on orders, order_items)
11. **00011**: `subscription_plans` + `subscriptions` + `subscription_items` + `subscription_schedule` (depends on users, addresses, product_skus, billing, shipping, payments)
12. **00012**: `invoices` + `invoice_items` + `invoice_payments` + `invoice_status_history` (depends on users, addresses, orders, payments)
13. **00013**: `jobs` + `job_history` (depends on tenants)
14. **00014**: `product_categories`, `product_tags`, `product_reviews`, `discount_codes`, `email_templates` (supporting tables)

All migrations include proper UP and DOWN statements for reversibility.

---

## Triggers Summary

| Trigger | Table | Purpose |
|---------|-------|---------|
| `update_*_updated_at` | All tables | Auto-update `updated_at` timestamp |
| `update_cart_last_activity` | cart_items | Update cart.last_activity_at on item changes |
| `log_order_status_change` | orders | Populate order_status_history |
| `log_invoice_status_change` | invoices | Populate invoice_status_history |
| `update_invoice_balance` | invoice_payments | Recalculate invoice balance and status |
| `archive_completed_job` | jobs | Move completed/failed jobs to job_history |
| `update_review_helpfulness_counts` | review_helpfulness | Update review helpful/not_helpful counts |
| `increment_discount_code_usage` | discount_code_usage | Increment usage_count on discount_codes |

---

## Next Steps

1. **Run migrations**: `make migrate`
2. **Write sqlc queries**: Define queries in `sqlc/queries/*.sql`
3. **Generate code**: `make sqlc-gen`
4. **Seed data**: Create seed script for development tenants, users, products

---

## Schema Evolution

Future considerations (not implemented yet):
- **Audit log table**: System-wide change tracking
- **File storage table**: Track uploaded files (product images, documents)
- **Notifications table**: In-app notifications for users
- **API keys table**: For customer integrations
- **Multi-location inventory**: Separate inventory per warehouse
- **Cupping notes**: Professional coffee tasting notes
- **Green coffee inventory**: Pre-roast coffee tracking

These can be added via new migrations as needs arise.
