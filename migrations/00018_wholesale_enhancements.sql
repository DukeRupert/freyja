-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- WHOLESALE ENHANCEMENTS MIGRATION
-- ============================================================================
-- This migration adds features inspired by Orderspace API patterns:
-- 1. Payment terms as a first-class entity (not just a string)
-- 2. Extended customer fields for wholesale operations
-- 3. PO number tracking on orders
-- 4. Quantity tracking for partial fulfillment
-- 5. Billing cycles for consolidated invoicing
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. PAYMENT TERMS TABLE
-- ----------------------------------------------------------------------------
-- Reusable payment terms (Net 15, Net 30, etc.) assigned to customers.
-- Replaces the string-based approach with a proper entity.

CREATE TABLE payment_terms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Identification
    name VARCHAR(100) NOT NULL,              -- e.g., "Net 30", "Due on Receipt"
    code VARCHAR(50) NOT NULL,               -- e.g., "net_30", "due_on_receipt"

    -- Terms configuration
    days INTEGER NOT NULL DEFAULT 0,         -- Days until due (0 = due on receipt)

    -- Settings
    is_default BOOLEAN NOT NULL DEFAULT FALSE, -- Default for new wholesale accounts
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Description for customer-facing documents
    description TEXT,                        -- e.g., "Payment due within 30 days of invoice date"

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_terms_tenant_code_unique UNIQUE (tenant_id, code)
);

-- Ensure only one default per tenant
CREATE UNIQUE INDEX idx_payment_terms_default
    ON payment_terms(tenant_id)
    WHERE is_default = TRUE;

CREATE INDEX idx_payment_terms_tenant_id ON payment_terms(tenant_id);
CREATE INDEX idx_payment_terms_active ON payment_terms(tenant_id, is_active) WHERE is_active = TRUE;

CREATE TRIGGER update_payment_terms_updated_at
    BEFORE UPDATE ON payment_terms
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE payment_terms IS 'Reusable payment terms for wholesale invoicing';
COMMENT ON COLUMN payment_terms.days IS 'Days until payment due (0 = due on receipt)';

-- ----------------------------------------------------------------------------
-- 2. EXTENDED CUSTOMER FIELDS FOR WHOLESALE
-- ----------------------------------------------------------------------------

-- Internal notes (admin-only, not visible to customer)
ALTER TABLE users ADD COLUMN internal_note TEXT;
COMMENT ON COLUMN users.internal_note IS 'Admin-only notes hidden from customer (e.g., "prefers Thursday delivery")';

-- Per-customer minimum spend (cents, overrides price list minimum)
ALTER TABLE users ADD COLUMN minimum_spend_cents INTEGER;
COMMENT ON COLUMN users.minimum_spend_cents IS 'Minimum order value in cents for this customer';

-- Split notification emails (wholesale customers often want different recipients)
ALTER TABLE users ADD COLUMN email_orders TEXT;
ALTER TABLE users ADD COLUMN email_dispatches TEXT;
ALTER TABLE users ADD COLUMN email_invoices TEXT;
COMMENT ON COLUMN users.email_orders IS 'Email(s) for order confirmations (comma-separated)';
COMMENT ON COLUMN users.email_dispatches IS 'Email(s) for shipping notifications (comma-separated)';
COMMENT ON COLUMN users.email_invoices IS 'Email(s) for invoice delivery (comma-separated)';

-- Foreign key to payment_terms table (replaces string-based payment_terms column)
ALTER TABLE users ADD COLUMN payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL;
CREATE INDEX idx_users_payment_terms_id ON users(payment_terms_id);
COMMENT ON COLUMN users.payment_terms_id IS 'Assigned payment terms for wholesale invoicing';

-- Billing cycle for consolidated invoicing
ALTER TABLE users ADD COLUMN billing_cycle VARCHAR(20) CHECK (billing_cycle IN ('weekly', 'biweekly', 'monthly', 'on_order'));
ALTER TABLE users ADD COLUMN billing_cycle_day INTEGER CHECK (billing_cycle_day >= 1 AND billing_cycle_day <= 28);
COMMENT ON COLUMN users.billing_cycle IS 'Billing frequency: weekly, biweekly, monthly, or on_order (invoice per order)';
COMMENT ON COLUMN users.billing_cycle_day IS 'Day of week (1-7) or month (1-28) to generate consolidated invoice';

-- Customer reference (internal ID visible to customer but not editable by them)
ALTER TABLE users ADD COLUMN customer_reference VARCHAR(100);
COMMENT ON COLUMN users.customer_reference IS 'Your internal customer reference (visible to customer)';

-- ----------------------------------------------------------------------------
-- 3. PO NUMBER ON ORDERS
-- ----------------------------------------------------------------------------
-- B2B customers need to track their purchase orders for accounting.

ALTER TABLE orders ADD COLUMN customer_po_number VARCHAR(100);
CREATE INDEX idx_orders_po_number ON orders(tenant_id, customer_po_number) WHERE customer_po_number IS NOT NULL;
COMMENT ON COLUMN orders.customer_po_number IS 'Customer purchase order number for B2B tracking';

-- Requested delivery date (when customer wants the order delivered)
ALTER TABLE orders ADD COLUMN requested_delivery_date DATE;
COMMENT ON COLUMN orders.requested_delivery_date IS 'Customer-requested delivery date';

-- ----------------------------------------------------------------------------
-- 4. QUANTITY TRACKING FOR PARTIAL FULFILLMENT
-- ----------------------------------------------------------------------------
-- Track how many units of each line item have been dispatched.
-- Works with the existing shipment_items table.

ALTER TABLE order_items ADD COLUMN quantity_dispatched INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_order_items_unfulfilled ON order_items(order_id)
    WHERE quantity_dispatched < quantity;
COMMENT ON COLUMN order_items.quantity_dispatched IS 'Units shipped (for partial fulfillment tracking)';

-- ----------------------------------------------------------------------------
-- 5. INVOICE ENHANCEMENTS
-- ----------------------------------------------------------------------------

-- Link invoices to payment_terms entity
ALTER TABLE invoices ADD COLUMN payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL;
CREATE INDEX idx_invoices_payment_terms_id ON invoices(payment_terms_id);
COMMENT ON COLUMN invoices.payment_terms_id IS 'Payment terms applied to this invoice';

-- Billing period for consolidated invoices
ALTER TABLE invoices ADD COLUMN billing_period_start DATE;
ALTER TABLE invoices ADD COLUMN billing_period_end DATE;
COMMENT ON COLUMN invoices.billing_period_start IS 'Start of billing period for consolidated invoices';
COMMENT ON COLUMN invoices.billing_period_end IS 'End of billing period for consolidated invoices';

-- Proforma flag (preliminary invoice before final billing)
ALTER TABLE invoices ADD COLUMN is_proforma BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN invoices.is_proforma IS 'True for preliminary invoices (not final billing)';

-- ----------------------------------------------------------------------------
-- 6. ORDER-INVOICE LINKING TABLE
-- ----------------------------------------------------------------------------
-- For consolidated invoicing, multiple orders can appear on one invoice.
-- The existing invoice_items.order_id handles line-level linking.
-- This table provides order-level summary linking.

CREATE TABLE invoice_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    -- Order snapshot at time of invoicing
    order_number VARCHAR(50) NOT NULL,
    order_total_cents INTEGER NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT invoice_orders_unique UNIQUE (invoice_id, order_id)
);

CREATE INDEX idx_invoice_orders_tenant_id ON invoice_orders(tenant_id);
CREATE INDEX idx_invoice_orders_invoice_id ON invoice_orders(invoice_id);
CREATE INDEX idx_invoice_orders_order_id ON invoice_orders(order_id);

COMMENT ON TABLE invoice_orders IS 'Links orders to invoices for consolidated billing';

-- ----------------------------------------------------------------------------
-- 7. SEED DEFAULT PAYMENT TERMS
-- ----------------------------------------------------------------------------
-- Note: These need tenant_id, so they must be created per-tenant.
-- This is just a reference for what to create during tenant setup.

-- Example SQL to run per tenant (not auto-executed):
-- INSERT INTO payment_terms (tenant_id, name, code, days, is_default, sort_order, description)
-- VALUES
--     ($tenant_id, 'Due on Receipt', 'due_on_receipt', 0, false, 1, 'Payment due immediately upon invoice'),
--     ($tenant_id, 'Net 15', 'net_15', 15, false, 2, 'Payment due within 15 days'),
--     ($tenant_id, 'Net 30', 'net_30', 30, true, 3, 'Payment due within 30 days'),
--     ($tenant_id, 'Net 45', 'net_45', 45, false, 4, 'Payment due within 45 days'),
--     ($tenant_id, 'Net 60', 'net_60', 60, false, 5, 'Payment due within 60 days');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop invoice_orders table
DROP TABLE IF EXISTS invoice_orders CASCADE;

-- Remove invoice enhancements
ALTER TABLE invoices DROP COLUMN IF EXISTS is_proforma;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_end;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_start;
DROP INDEX IF EXISTS idx_invoices_payment_terms_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_terms_id;

-- Remove order_items quantity tracking
DROP INDEX IF EXISTS idx_order_items_unfulfilled;
ALTER TABLE order_items DROP COLUMN IF EXISTS quantity_dispatched;

-- Remove order enhancements
ALTER TABLE orders DROP COLUMN IF EXISTS requested_delivery_date;
DROP INDEX IF EXISTS idx_orders_po_number;
ALTER TABLE orders DROP COLUMN IF EXISTS customer_po_number;

-- Remove user wholesale extensions
ALTER TABLE users DROP COLUMN IF EXISTS customer_reference;
ALTER TABLE users DROP COLUMN IF EXISTS billing_cycle_day;
ALTER TABLE users DROP COLUMN IF EXISTS billing_cycle;
DROP INDEX IF EXISTS idx_users_payment_terms_id;
ALTER TABLE users DROP COLUMN IF EXISTS payment_terms_id;
ALTER TABLE users DROP COLUMN IF EXISTS email_invoices;
ALTER TABLE users DROP COLUMN IF EXISTS email_dispatches;
ALTER TABLE users DROP COLUMN IF EXISTS email_orders;
ALTER TABLE users DROP COLUMN IF EXISTS minimum_spend_cents;
ALTER TABLE users DROP COLUMN IF EXISTS internal_note;

-- Drop payment_terms table
DROP TRIGGER IF EXISTS update_payment_terms_updated_at ON payment_terms;
DROP INDEX IF EXISTS idx_payment_terms_active;
DROP INDEX IF EXISTS idx_payment_terms_tenant_id;
DROP INDEX IF EXISTS idx_payment_terms_default;
DROP TABLE IF EXISTS payment_terms CASCADE;

-- +goose StatementEnd
