-- +goose Up
-- +goose StatementBegin

-- Invoices: wholesale billing invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Invoice identification
    invoice_number VARCHAR(50) NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft',
        'sent',
        'viewed',
        'partial',
        'paid',
        'overdue',
        'cancelled',
        'void'
    )),

    -- Amounts
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    shipping_cents INTEGER NOT NULL DEFAULT 0,
    discount_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL,
    paid_cents INTEGER NOT NULL DEFAULT 0,
    balance_cents INTEGER NOT NULL, -- total_cents - paid_cents
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Payment terms
    payment_terms VARCHAR(20) NOT NULL DEFAULT 'net_30', -- e.g., 'net_15', 'net_30', 'due_on_receipt'
    due_date DATE NOT NULL,

    -- Billing provider integration
    billing_customer_id UUID REFERENCES billing_customers(id) ON DELETE SET NULL,
    provider VARCHAR(50),
    provider_invoice_id VARCHAR(255),

    -- Addresses
    billing_address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE RESTRICT,

    -- Notes
    customer_notes TEXT,
    internal_notes TEXT,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Important timestamps
    sent_at TIMESTAMP WITH TIME ZONE,
    viewed_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    voided_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT invoices_tenant_number_unique UNIQUE (tenant_id, invoice_number)
);

-- Invoice items: line items on an invoice
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,

    -- Item type
    item_type VARCHAR(20) NOT NULL DEFAULT 'product' CHECK (item_type IN ('product', 'shipping', 'discount', 'custom')),

    -- Product reference (optional)
    product_sku_id UUID REFERENCES product_skus(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,

    -- Description (for display on invoice)
    description TEXT NOT NULL,

    -- Pricing
    quantity DECIMAL(10, 2) NOT NULL DEFAULT 1,
    unit_price_cents INTEGER NOT NULL,
    total_price_cents INTEGER NOT NULL,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Invoice payments: tracks payments applied to invoices
CREATE TABLE invoice_payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,

    -- Payment reference
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,

    -- Amount applied to this invoice
    amount_cents INTEGER NOT NULL,

    -- Payment details (for manual payments)
    payment_method VARCHAR(50) DEFAULT 'stripe', -- 'stripe', 'check', 'wire', 'cash', etc.
    payment_reference VARCHAR(255), -- Check number, transaction ID, etc.

    -- Notes
    notes TEXT,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    payment_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Invoice status history: audit trail for invoice status changes
CREATE TABLE invoice_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,

    -- Status change
    from_status VARCHAR(20),
    to_status VARCHAR(20) NOT NULL,

    -- Who made the change
    changed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    change_reason TEXT,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_invoice_number ON invoices(tenant_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(tenant_id, status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_overdue ON invoices(tenant_id, status, due_date)
    WHERE status NOT IN ('paid', 'cancelled', 'void') AND due_date < CURRENT_DATE;
CREATE INDEX idx_invoices_provider ON invoices(provider, provider_invoice_id);
CREATE INDEX idx_invoices_created_at ON invoices(created_at);

CREATE INDEX idx_invoice_items_tenant_id ON invoice_items(tenant_id);
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_product_sku_id ON invoice_items(product_sku_id);
CREATE INDEX idx_invoice_items_order_id ON invoice_items(order_id);

CREATE INDEX idx_invoice_payments_tenant_id ON invoice_payments(tenant_id);
CREATE INDEX idx_invoice_payments_invoice_id ON invoice_payments(invoice_id);
CREATE INDEX idx_invoice_payments_payment_id ON invoice_payments(payment_id);
CREATE INDEX idx_invoice_payments_payment_date ON invoice_payments(payment_date);

CREATE INDEX idx_invoice_status_history_tenant_id ON invoice_status_history(tenant_id);
CREATE INDEX idx_invoice_status_history_invoice_id ON invoice_status_history(invoice_id);
CREATE INDEX idx_invoice_status_history_created_at ON invoice_status_history(created_at);

-- Auto-update triggers
CREATE TRIGGER update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoice_items_updated_at
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoice_payments_updated_at
    BEFORE UPDATE ON invoice_payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to log invoice status changes
CREATE OR REPLACE FUNCTION log_invoice_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO invoice_status_history (
            tenant_id,
            invoice_id,
            from_status,
            to_status,
            created_at
        ) VALUES (
            NEW.tenant_id,
            NEW.id,
            OLD.status,
            NEW.status,
            NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER log_invoice_status_changes
    AFTER UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION log_invoice_status_change();

-- Trigger to update invoice balance when payments are added
CREATE OR REPLACE FUNCTION update_invoice_balance()
RETURNS TRIGGER AS $$
DECLARE
    v_total_paid INTEGER;
    v_total_amount INTEGER;
BEGIN
    -- Calculate total paid for this invoice
    SELECT COALESCE(SUM(amount_cents), 0)
    INTO v_total_paid
    FROM invoice_payments
    WHERE invoice_id = COALESCE(NEW.invoice_id, OLD.invoice_id);

    -- Get invoice total
    SELECT total_cents
    INTO v_total_amount
    FROM invoices
    WHERE id = COALESCE(NEW.invoice_id, OLD.invoice_id);

    -- Update invoice paid and balance
    UPDATE invoices
    SET
        paid_cents = v_total_paid,
        balance_cents = v_total_amount - v_total_paid,
        status = CASE
            WHEN v_total_paid = 0 THEN status -- Keep current status if no payments
            WHEN v_total_paid >= v_total_amount THEN 'paid'
            WHEN v_total_paid > 0 THEN 'partial'
            ELSE status
        END,
        paid_at = CASE
            WHEN v_total_paid >= v_total_amount THEN NOW()
            ELSE paid_at
        END
    WHERE id = COALESCE(NEW.invoice_id, OLD.invoice_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_invoice_balance_on_payment
    AFTER INSERT OR UPDATE OR DELETE ON invoice_payments
    FOR EACH ROW
    EXECUTE FUNCTION update_invoice_balance();

COMMENT ON TABLE invoices IS 'Wholesale billing invoices';
COMMENT ON TABLE invoice_items IS 'Line items on invoices';
COMMENT ON TABLE invoice_payments IS 'Payments applied to invoices';
COMMENT ON TABLE invoice_status_history IS 'Audit trail for invoice status changes';
COMMENT ON COLUMN invoices.payment_terms IS 'Payment terms: net_15, net_30, net_60, due_on_receipt';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_invoice_balance_on_payment ON invoice_payments;
DROP FUNCTION IF EXISTS update_invoice_balance();
DROP TRIGGER IF EXISTS log_invoice_status_changes ON invoices;
DROP FUNCTION IF EXISTS log_invoice_status_change();
DROP TRIGGER IF EXISTS update_invoice_payments_updated_at ON invoice_payments;
DROP TRIGGER IF EXISTS update_invoice_items_updated_at ON invoice_items;
DROP TRIGGER IF EXISTS update_invoices_updated_at ON invoices;
DROP TABLE IF EXISTS invoice_status_history CASCADE;
DROP TABLE IF EXISTS invoice_payments CASCADE;
DROP TABLE IF EXISTS invoice_items CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
-- +goose StatementEnd
