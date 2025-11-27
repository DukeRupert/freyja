# Database Migrations

This directory contains Goose SQL migrations for the Freyja database schema.

## Migration Files

| File | Description | Tables Created |
|------|-------------|----------------|
| `00001_create_tenants.sql` | Multi-tenant root table | `tenants` |
| `00002_create_users.sql` | Customer accounts | `users` |
| `00003_create_sessions.sql` | Authentication sessions | `sessions` |
| `00004_create_addresses.sql` | Shipping/billing addresses | `addresses`, `customer_addresses` |
| `00005_create_products.sql` | Product catalog | `products`, `product_skus`, `product_images` |
| `00006_create_price_lists.sql` | Pricing system | `price_lists`, `price_list_entries`, `user_price_lists` |
| `00007_create_carts.sql` | Shopping carts | `carts`, `cart_items` |
| `00008_create_billing.sql` | Payment processing | `billing_customers`, `payment_methods`, `payments`, `webhook_events` |
| `00009_create_orders.sql` | Order management | `orders`, `order_items`, `order_status_history` |
| `00010_create_shipping.sql` | Shipping & fulfillment | `shipping_methods`, `shipping_rates`, `shipments`, `shipment_items`, `shipment_tracking_events` |
| `00011_create_subscriptions.sql` | Recurring subscriptions | `subscription_plans`, `subscriptions`, `subscription_items`, `subscription_schedule` |
| `00012_create_invoices.sql` | Wholesale invoicing | `invoices`, `invoice_items`, `invoice_payments`, `invoice_status_history` |
| `00013_create_jobs.sql` | Background job queue | `jobs`, `job_history` |
| `00014_create_supporting_tables.sql` | Additional features | `product_categories`, `product_tags`, `product_reviews`, `discount_codes`, `email_templates` |

## Running Migrations

### Apply all migrations
```bash
make migrate
```

### Rollback last migration
```bash
make migrate-down
```

### Create new migration
```bash
make migrate-create NAME=add_new_feature
```

### Check migration status
```bash
goose -dir migrations postgres "user=freyja password=password dbname=freyja sslmode=disable" status
```

## Migration Conventions

1. **Naming**: `NNNNN_description.sql` where NNNNN is a 5-digit sequence number
2. **Structure**: Each file has both `-- +goose Up` and `-- +goose Down` sections
3. **Reversibility**: All migrations must be reversible (DOWN must undo UP)
4. **Dependencies**: Migrations are ordered by table dependencies
5. **Idempotency**: Use `IF EXISTS` / `IF NOT EXISTS` where appropriate

## Key Features

### UUID Primary Keys
All tables use UUIDs for primary keys:
```sql
id UUID PRIMARY KEY DEFAULT uuid_generate_v4()
```

### Tenant Isolation
All tables (except `tenants`) have `tenant_id`:
```sql
tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
```

### Timestamps
All tables have `created_at` and `updated_at` with auto-update triggers:
```sql
created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
```

### JSONB Metadata
Extensibility via JSONB fields:
```sql
metadata JSONB NOT NULL DEFAULT '{}'
```

### Status Enums
CHECK constraints for valid statuses:
```sql
status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled'))
```

### Indexes
- `tenant_id` on all tables
- Foreign keys
- Status fields (partial indexes for active/pending states)
- Unique constraints for business keys (email, sku, order_number, etc.)

## Database Setup

### Initial setup (first time only)
1. Start Docker services: `make docker-up`
2. Run migrations: `make migrate`
3. (Optional) Seed data: `make seed`

### Reset database (development only)
```bash
# WARNING: This deletes all data!
make migrate-down  # Rollback all migrations
make migrate       # Reapply all migrations
make seed          # Reseed data
```

## Testing Migrations

Before committing:
1. Apply migration: `make migrate`
2. Verify schema: Check tables exist with correct columns
3. Test rollback: `make migrate-down`
4. Verify cleanup: Check tables are removed
5. Reapply: `make migrate`

## Migration Guidelines

### DO:
- ✅ Keep migrations small and focused
- ✅ Test both UP and DOWN
- ✅ Add indexes for foreign keys
- ✅ Use CHECK constraints for enums
- ✅ Add comments for complex logic
- ✅ Include ON DELETE behavior for foreign keys

### DON'T:
- ❌ Modify existing migrations (create new ones instead)
- ❌ Skip DOWN migrations
- ❌ Add data migrations (use seeds instead)
- ❌ Forget tenant_id on new tables
- ❌ Create tables without indexes on foreign keys

## Common Patterns

### Adding a new table
```sql
-- +goose Up
CREATE TABLE table_name (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- your columns here

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_table_name_tenant_id ON table_name(tenant_id);

CREATE TRIGGER update_table_name_updated_at
    BEFORE UPDATE ON table_name
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS update_table_name_updated_at ON table_name;
DROP TABLE IF EXISTS table_name CASCADE;
```

### Adding a new column
```sql
-- +goose Up
ALTER TABLE table_name
ADD COLUMN new_column VARCHAR(255);

-- +goose Down
ALTER TABLE table_name
DROP COLUMN IF EXISTS new_column;
```

### Adding an index
```sql
-- +goose Up
CREATE INDEX idx_table_name_column ON table_name(column);

-- +goose Down
DROP INDEX IF EXISTS idx_table_name_column;
```

## Troubleshooting

### Migration fails with "relation already exists"
```bash
# Check which migrations have been applied
goose -dir migrations postgres "user=freyja password=password dbname=freyja sslmode=disable" status

# Manually rollback if needed
make migrate-down
```

### Need to reset Goose version tracking
```bash
# Connect to database
psql -U freyja -d freyja

# Check current version
SELECT * FROM goose_db_version ORDER BY id DESC LIMIT 5;

# Manually set version (careful!)
DELETE FROM goose_db_version WHERE version_id > 5;  -- Example: rollback to version 5
```

### Migration is slow
Large indexes can be slow. Consider:
- Creating indexes CONCURRENTLY (requires separate transaction)
- Splitting large migrations into smaller ones
- Running during low-traffic periods

## See Also

- Full schema documentation: `/planning/DATABASE_SCHEMA.md`
- Goose documentation: https://github.com/pressly/goose
- sqlc documentation: https://docs.sqlc.dev/
