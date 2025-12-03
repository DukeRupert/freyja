# Database Schema - Implementation Complete

## Summary

The complete PostgreSQL database schema for Freyja has been designed and documented. All 14 migration files have been created and are ready to apply.

---

## What Was Created

### 1. Migration Files (14 files)

Located in `/home/dukerupert/Repos/freyja/migrations/`:

| File | Description | Tables |
|------|-------------|--------|
| `00001_create_tenants.sql` | Multi-tenant root | 1 table |
| `00002_create_users.sql` | Customer accounts | 1 table |
| `00003_create_sessions.sql` | Authentication | 1 table |
| `00004_create_addresses.sql` | Shipping/billing | 2 tables |
| `00005_create_products.sql` | Product catalog | 3 tables |
| `00006_create_price_lists.sql` | Pricing system | 3 tables |
| `00007_create_carts.sql` | Shopping carts | 2 tables |
| `00008_create_billing.sql` | Payment processing | 4 tables |
| `00009_create_orders.sql` | Order management | 3 tables |
| `00010_create_shipping.sql` | Shipping & fulfillment | 5 tables |
| `00011_create_subscriptions.sql` | Recurring orders | 4 tables |
| `00012_create_invoices.sql` | Wholesale invoicing | 4 tables |
| `00013_create_jobs.sql` | Background jobs | 2 tables |
| `00014_create_supporting_tables.sql` | Additional features | 7 tables |

**Total: 41 tables** covering all 6 MVP phases.

---

### 2. Documentation Files (4 files)

Located in `/home/dukerupert/Repos/freyja/planning/`:

1. **`DATABASE_SCHEMA.md`** (comprehensive reference)
   - Full table documentation
   - Design decisions and tradeoffs
   - Index strategy
   - Trigger explanations
   - Migration order rationale

2. **`DATABASE_ERD.md`** (visual reference)
   - Entity relationship diagrams
   - Data flow examples
   - Query patterns
   - Performance considerations

3. **`DATABASE_QUICK_REFERENCE.md`** (lookup guide)
   - Table cheat sheets
   - Status enum values
   - Common queries
   - Foreign key relationships
   - Critical indexes

4. **`migrations/README.md`** (migration guide)
   - How to run migrations
   - Migration conventions
   - Common patterns
   - Troubleshooting

---

## Schema Features

### Multi-Tenancy
- Row-level isolation via `tenant_id` on all tables (except `tenants`)
- Enforced through sqlc type-safe queries
- Proper cascading deletes for tenant cleanup

### UUID Primary Keys
- All tables use UUIDs for better security and distributed compatibility
- PostgreSQL `uuid-ossp` extension enabled

### Timestamps
- `created_at` and `updated_at` on all tables
- Automatic `updated_at` updates via triggers

### Extensibility
- JSONB `metadata` fields for future flexibility
- Status ENUMs defined but not using PostgreSQL ENUM type (easier migrations)
- Provider abstraction (Stripe initially, but designed for multiple)

### Audit Trails
- `order_status_history`: Logs all order status changes
- `invoice_status_history`: Logs all invoice status changes
- `job_history`: Archives completed/failed jobs
- `shipment_tracking_events`: Tracks delivery status

### Idempotency
- `webhook_events` table with unique constraint on provider event ID
- Prevents duplicate webhook processing

### Performance
- Comprehensive indexing strategy
- Partial indexes for common filters
- Composite indexes for multi-column queries
- Foreign key indexes for efficient joins

---

## Table Breakdown by MVP Phase

### Phase 1: Foundation (Weeks 1-2)
- ✅ `tenants`, `users`, `sessions` (identity)
- ✅ `addresses`, `customer_addresses` (customer data)
- ✅ `products`, `product_skus`, `product_images` (catalog)
- ✅ `price_lists`, `price_list_entries`, `user_price_lists` (pricing)
- ✅ `product_categories`, `product_tags` (organization)

### Phase 2: Storefront & Cart (Weeks 3-4)
- ✅ `carts`, `cart_items` (shopping)

### Phase 3: Billing & Payments (Weeks 5-6)
- ✅ `billing_customers`, `payment_methods`, `payments` (billing)
- ✅ `webhook_events` (idempotency)
- ✅ `orders`, `order_items`, `order_status_history` (orders)

### Phase 4: Shipping (Weeks 7-8)
- ✅ `shipping_methods`, `shipping_rates` (methods)
- ✅ `shipments`, `shipment_items`, `shipment_tracking_events` (fulfillment)

### Phase 5: Subscriptions (Weeks 9-10)
- ✅ `subscription_plans`, `subscriptions`, `subscription_items` (subscriptions)
- ✅ `subscription_schedule` (scheduling)

### Phase 6: Wholesale & Invoicing (Weeks 11-12)
- ✅ `invoices`, `invoice_items`, `invoice_payments` (invoicing)
- ✅ `invoice_status_history` (audit)

### Supporting Features
- ✅ `jobs`, `job_history` (background processing)
- ✅ `product_reviews`, `review_helpfulness` (reviews)
- ✅ `discount_codes`, `discount_code_usage` (promotions)
- ✅ `email_templates` (transactional emails)

---

## Key Design Decisions

### 1. Multi-Tenancy: Row-Level (Not Schema-Per-Tenant)
**Decision**: Share tables, isolate via `tenant_id`
**Rationale**: Simpler deployment, easier cross-tenant analytics
**Tradeoff**: Must be vigilant about tenant_id in queries (mitigated by sqlc)

### 2. Product-SKU Split
**Decision**: Separate `products` (coffee) from `product_skus` (variants)
**Rationale**: Products have attributes; SKUs have inventory and pricing
**Example**: "Ethiopia Yirgacheffe" (product) → "12oz whole bean" (SKU)

### 3. Price List System
**Decision**: Named price lists with junction table entries
**Rationale**: Supports retail, multiple wholesale tiers, and custom pricing
**Pattern**: User → price_list → price_list_entries → SKU prices

### 4. Cart Persistence Strategy
**Decision**: Single `carts` table for guests and users
**Guest**: `user_id IS NULL`, linked by `session_id`
**User**: `user_id` set, survives logout
**Tradeoff**: Slightly complex query logic, avoids duplicate systems

### 5. Billing Abstraction
**Decision**: `billing_customers` maps users to provider IDs
**Fields**: `provider`, `provider_customer_id`, `metadata` (JSONB)
**Rationale**: Easy to support multiple payment processors later

### 6. Webhook Idempotency
**Decision**: `webhook_events` with unique `(provider, provider_event_id)`
**Processing**: Check existence → insert → process → update status
**Rationale**: Safely handles duplicate webhook deliveries

### 7. Status ENUMs + History
**Decision**: VARCHAR status with CHECK constraints + separate history tables
**Rationale**: Easier migrations than PostgreSQL ENUM, automatic audit trail
**Tables**: `order_status_history`, `invoice_status_history`

### 8. Invoice Auto-Balance
**Decision**: Triggers recalculate `balance_cents` on payment changes
**Rationale**: Balance always accurate, no manual sync needed
**Status**: Auto-updates to 'paid' or 'partial' based on balance

---

## Coffee-Specific Fields

The schema includes proper coffee industry attributes:

### Products
- `origin`: Country (e.g., "Ethiopia")
- `region`: Growing region (e.g., "Yirgacheffe")
- `producer`: Farm or cooperative name
- `process`: Processing method (washed, natural, honey)
- `roast_level`: Light, medium, dark
- `elevation_min`, `elevation_max`: Meters above sea level
- `variety`: Coffee variety (Heirloom, Bourbon, Caturra, etc.)
- `harvest_year`: Year harvested
- `tasting_notes`: Array of strings (['blueberry', 'chocolate', 'caramel'])

### Product SKUs
- `weight_value` + `weight_unit`: Flexible weight system (oz, lb, g, kg)
- `grind`: Grind size (whole_bean, coarse, medium, fine, espresso, etc.)
- `inventory_quantity`: Stock tracking per variant
- `inventory_policy`: 'deny' (no backorders) or 'allow' (accept backorders)

---

## Next Steps

### 1. Apply Migrations
```bash
cd /home/dukerupert/Repos/freyja
make migrate
```

This will:
- Connect to PostgreSQL (via Docker)
- Apply all 14 migrations in order
- Create all 41 tables with proper indexes and triggers

### 2. Verify Schema
```bash
# Connect to database
psql -U freyja -d freyja

# List all tables
\dt

# Describe a table
\d products

# Check indexes
\di

# Exit
\q
```

Expected: 41 tables created successfully.

### 3. Write sqlc Queries

Create queries in `/home/dukerupert/Repos/freyja/sqlc/queries/`:

**Example: `users.sql`**
```sql
-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE tenant_id = $1
  AND email = $2
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    account_type
) VALUES (
    $1, $2, $3, $4
) RETURNING *;
```

**Example: `products.sql`**
```sql
-- name: ListActiveProducts :many
SELECT
    p.*,
    COUNT(DISTINCT ps.id) as sku_count
FROM products p
LEFT JOIN product_skus ps ON ps.product_id = p.id AND ps.is_active = TRUE
WHERE p.tenant_id = $1
  AND p.status = 'active'
GROUP BY p.id
ORDER BY p.sort_order, p.created_at DESC;

-- name: GetProductWithSKUs :one
SELECT
    p.*,
    json_agg(
        json_build_object(
            'id', ps.id,
            'sku', ps.sku,
            'weight_value', ps.weight_value,
            'weight_unit', ps.weight_unit,
            'grind', ps.grind,
            'base_price_cents', ps.base_price_cents,
            'inventory_quantity', ps.inventory_quantity
        )
    ) as skus
FROM products p
LEFT JOIN product_skus ps ON ps.product_id = p.id AND ps.is_active = TRUE
WHERE p.tenant_id = $1
  AND p.id = $2
GROUP BY p.id;
```

### 4. Generate Type-Safe Code
```bash
make sqlc-gen
```

This generates Go code in `/home/dukerupert/Repos/freyja/internal/repository/`:
- `db.go`: Database connection
- `models.go`: Struct types for all tables
- `querier.go`: Interface with all query methods
- `users.sql.go`: Generated user query functions
- `products.sql.go`: Generated product query functions
- etc.

### 5. Create Seed Data (Optional)

Create `/home/dukerupert/Repos/freyja/scripts/seed.sql`:

```sql
-- Seed development tenant
INSERT INTO tenants (id, name, slug, email, status)
VALUES (
    'c7f8f9d0-1234-5678-9012-123456789012'::uuid,
    'Test Roastery',
    'test',
    'admin@test.com',
    'active'
);

-- Seed admin user
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    account_type,
    first_name,
    last_name,
    status
) VALUES (
    'c7f8f9d0-1234-5678-9012-123456789012'::uuid,
    'admin@test.com',
    '$2a$10$...',  -- bcrypt hash of 'password'
    'admin',
    'Admin',
    'User',
    'active'
);

-- Seed default price list
INSERT INTO price_lists (tenant_id, name, list_type, is_active)
VALUES (
    'c7f8f9d0-1234-5678-9012-123456789012'::uuid,
    'Retail',
    'default',
    TRUE
);

-- Add more seed data...
```

Run seed:
```bash
psql -U freyja -d freyja < scripts/seed.sql
```

---

## Testing the Schema

### 1. Test Multi-Tenancy Isolation

```sql
-- Create two tenants
INSERT INTO tenants (name, slug, email, status)
VALUES ('Roastery A', 'roastery-a', 'a@example.com', 'active');

INSERT INTO tenants (name, slug, email, status)
VALUES ('Roastery B', 'roastery-b', 'b@example.com', 'active');

-- Create products for each
INSERT INTO products (tenant_id, name, slug, status)
SELECT id, 'Ethiopia Yirgacheffe', 'ethiopia', 'active'
FROM tenants WHERE slug = 'roastery-a';

INSERT INTO products (tenant_id, name, slug, status)
SELECT id, 'Colombia Supremo', 'colombia', 'active'
FROM tenants WHERE slug = 'roastery-b';

-- Verify isolation
SELECT p.name, t.name as tenant
FROM products p
JOIN tenants t ON t.id = p.tenant_id;

-- Should show:
-- Ethiopia Yirgacheffe | Roastery A
-- Colombia Supremo | Roastery B
```

### 2. Test Price List System

```sql
-- Get tenant ID
SET @tenant_id = (SELECT id FROM tenants WHERE slug = 'roastery-a');

-- Create product + SKU
INSERT INTO products (tenant_id, name, slug, status)
VALUES (@tenant_id, 'Test Coffee', 'test', 'active')
RETURNING id AS product_id \gset

INSERT INTO product_skus (tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents)
VALUES (@tenant_id, :product_id, 'TEST-12OZ', 12, 'oz', 'whole_bean', 1500)
RETURNING id AS sku_id \gset

-- Create wholesale price list
INSERT INTO price_lists (tenant_id, name, list_type, is_active)
VALUES (@tenant_id, 'Wholesale Tier 1', 'wholesale', TRUE)
RETURNING id AS price_list_id \gset

-- Add discounted price
INSERT INTO price_list_entries (tenant_id, price_list_id, product_sku_id, price_cents)
VALUES (@tenant_id, :price_list_id, :sku_id, 1200);

-- Verify
SELECT
    p.name as product,
    ps.sku,
    ps.base_price_cents as retail_price,
    ple.price_cents as wholesale_price
FROM products p
JOIN product_skus ps ON ps.product_id = p.id
LEFT JOIN price_list_entries ple ON ple.product_sku_id = ps.id
WHERE p.tenant_id = @tenant_id;
```

### 3. Test Order Flow

```sql
-- Create user
INSERT INTO users (tenant_id, email, account_type, status)
VALUES (@tenant_id, 'customer@example.com', 'retail', 'active')
RETURNING id AS user_id \gset

-- Create billing customer
INSERT INTO billing_customers (tenant_id, user_id, provider, provider_customer_id)
VALUES (@tenant_id, :user_id, 'stripe', 'cus_test123')
RETURNING id AS billing_customer_id \gset

-- Create payment
INSERT INTO payments (tenant_id, billing_customer_id, provider, provider_payment_id, amount_cents, status)
VALUES (@tenant_id, :billing_customer_id, 'stripe', 'pi_test123', 1500, 'succeeded')
RETURNING id AS payment_id \gset

-- Create address
INSERT INTO addresses (tenant_id, address_line1, city, state, postal_code, country)
VALUES (@tenant_id, '123 Main St', 'Portland', 'OR', '97201', 'US')
RETURNING id AS address_id \gset

-- Create order
INSERT INTO orders (
    tenant_id,
    user_id,
    order_number,
    order_type,
    status,
    subtotal_cents,
    tax_cents,
    shipping_cents,
    total_cents,
    payment_id,
    shipping_address_id,
    billing_address_id
) VALUES (
    @tenant_id,
    :user_id,
    'ORD-001',
    'retail',
    'paid',
    1500,
    150,
    0,
    1650,
    :payment_id,
    :address_id,
    :address_id
) RETURNING id AS order_id \gset

-- Create order item
INSERT INTO order_items (
    tenant_id,
    order_id,
    product_sku_id,
    product_name,
    sku,
    quantity,
    unit_price_cents,
    total_price_cents
) VALUES (
    @tenant_id,
    :order_id,
    :sku_id,
    'Test Coffee',
    'TEST-12OZ',
    1,
    1500,
    1500
);

-- Verify order
SELECT
    o.order_number,
    o.status,
    o.total_cents,
    u.email as customer_email,
    COUNT(oi.id) as item_count
FROM orders o
JOIN users u ON u.id = o.user_id
JOIN order_items oi ON oi.order_id = o.id
WHERE o.tenant_id = @tenant_id
GROUP BY o.id, u.email;
```

---

## Schema Maintenance

### Adding New Tables
1. Create migration: `make migrate-create NAME=add_table_name`
2. Follow conventions (tenant_id, timestamps, indexes)
3. Test both UP and DOWN
4. Apply: `make migrate`

### Modifying Existing Tables
**Never modify existing migrations!** Instead:
1. Create new migration to ALTER table
2. Add/remove columns or indexes
3. Test both UP and DOWN
4. Apply: `make migrate`

### Rolling Back
```bash
# Rollback last migration
make migrate-down

# Rollback multiple
make migrate-down  # Repeat as needed
```

---

## Files Reference

### Migration Files
```
/home/dukerupert/Repos/freyja/migrations/
├── 00001_create_tenants.sql
├── 00002_create_users.sql
├── 00003_create_sessions.sql
├── 00004_create_addresses.sql
├── 00005_create_products.sql
├── 00006_create_price_lists.sql
├── 00007_create_carts.sql
├── 00008_create_billing.sql
├── 00009_create_orders.sql
├── 00010_create_shipping.sql
├── 00011_create_subscriptions.sql
├── 00012_create_invoices.sql
├── 00013_create_jobs.sql
├── 00014_create_supporting_tables.sql
└── README.md
```

### Documentation Files
```
/home/dukerupert/Repos/freyja/planning/
├── DATABASE_SCHEMA.md          # Comprehensive reference
├── DATABASE_ERD.md             # Visual diagrams
├── DATABASE_QUICK_REFERENCE.md # Lookup guide
└── ROADMAP.md                  # Feature roadmap
```

### This File
```
/home/dukerupert/Repos/freyja/SCHEMA_COMPLETE.md
```

---

## Success Criteria

Schema is complete and ready when:
- ✅ All 14 migration files created
- ✅ All 41 tables defined
- ✅ All indexes specified
- ✅ All triggers implemented
- ✅ All foreign keys with proper cascades
- ✅ All unique constraints defined
- ✅ Documentation complete
- ⏳ Migrations applied (`make migrate`)
- ⏳ sqlc queries written
- ⏳ Type-safe code generated (`make sqlc-gen`)

**Status**: Schema design complete. Ready to apply migrations and begin implementation.

---

## Questions or Issues?

Refer to:
1. **Full schema docs**: `/home/dukerupert/Repos/freyja/planning/DATABASE_SCHEMA.md`
2. **ERD diagrams**: `/home/dukerupert/Repos/freyja/planning/DATABASE_ERD.md`
3. **Quick reference**: `/home/dukerupert/Repos/freyja/planning/DATABASE_QUICK_REFERENCE.md`
4. **Migration guide**: `/home/dukerupert/Repos/freyja/migrations/README.md`

For architectural questions, consult the Freyja Architect agent.
