-- +goose Up
-- +goose StatementBegin

-- Orders: customer orders (both retail and wholesale)
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Order identification
    order_number VARCHAR(50) NOT NULL,

    -- Order type
    order_type VARCHAR(20) NOT NULL DEFAULT 'retail' CHECK (order_type IN ('retail', 'wholesale', 'subscription')),

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending',
        'payment_processing',
        'paid',
        'processing',
        'shipped',
        'delivered',
        'cancelled',
        'refunded'
    )),

    -- Pricing
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    shipping_cents INTEGER NOT NULL DEFAULT 0,
    discount_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Payment
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    payment_status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Addresses
    shipping_address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE RESTRICT,
    billing_address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE RESTRICT,

    -- Shipping method
    shipping_method VARCHAR(100),
    shipping_carrier VARCHAR(100),

    -- Customer notes
    customer_notes TEXT,
    internal_notes TEXT,

    -- Fulfillment
    fulfillment_status VARCHAR(50) NOT NULL DEFAULT 'unfulfilled' CHECK (fulfillment_status IN (
        'unfulfilled',
        'partial',
        'fulfilled',
        'cancelled'
    )),

    -- Related entities
    cart_id UUID REFERENCES carts(id) ON DELETE SET NULL,
    subscription_id UUID, -- Will be linked later when subscriptions table exists

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Important timestamps
    paid_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT orders_tenant_number_unique UNIQUE (tenant_id, order_number)
);

-- Order items: line items in an order
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_sku_id UUID NOT NULL REFERENCES product_skus(id) ON DELETE RESTRICT,

    -- Product snapshot at time of order (in case product changes later)
    product_name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    variant_description TEXT, -- e.g., "12oz, Whole Bean"

    -- Pricing
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL,
    total_price_cents INTEGER NOT NULL,

    -- Fulfillment
    fulfillment_status VARCHAR(50) NOT NULL DEFAULT 'unfulfilled' CHECK (fulfillment_status IN (
        'unfulfilled',
        'fulfilled',
        'cancelled'
    )),

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Order status history: audit trail for status changes
CREATE TABLE order_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    -- Status change
    from_status VARCHAR(50),
    to_status VARCHAR(50) NOT NULL,

    -- Who made the change
    changed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    change_reason TEXT,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_order_number ON orders(tenant_id, order_number);
CREATE INDEX idx_orders_status ON orders(tenant_id, status);
CREATE INDEX idx_orders_payment_status ON orders(tenant_id, payment_status);
CREATE INDEX idx_orders_fulfillment_status ON orders(tenant_id, fulfillment_status);
CREATE INDEX idx_orders_order_type ON orders(tenant_id, order_type);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_paid_at ON orders(paid_at) WHERE paid_at IS NOT NULL;
CREATE INDEX idx_orders_pending_fulfillment ON orders(tenant_id, status, fulfillment_status)
    WHERE status = 'paid' AND fulfillment_status = 'unfulfilled';

CREATE INDEX idx_order_items_tenant_id ON order_items(tenant_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_sku_id ON order_items(product_sku_id);
CREATE INDEX idx_order_items_fulfillment_status ON order_items(tenant_id, fulfillment_status);

CREATE INDEX idx_order_status_history_tenant_id ON order_status_history(tenant_id);
CREATE INDEX idx_order_status_history_order_id ON order_status_history(order_id);
CREATE INDEX idx_order_status_history_created_at ON order_status_history(created_at);

-- Auto-update triggers
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_order_items_updated_at
    BEFORE UPDATE ON order_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to log status changes to history table
CREATE OR REPLACE FUNCTION log_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO order_status_history (
            tenant_id,
            order_id,
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

CREATE TRIGGER log_order_status_changes
    AFTER UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION log_order_status_change();

COMMENT ON TABLE orders IS 'Customer orders (retail, wholesale, subscription)';
COMMENT ON TABLE order_items IS 'Line items in orders';
COMMENT ON TABLE order_status_history IS 'Audit trail for order status changes';
COMMENT ON COLUMN orders.order_type IS 'retail: one-time order, wholesale: invoice-based, subscription: recurring';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS log_order_status_changes ON orders;
DROP FUNCTION IF EXISTS log_order_status_change();
DROP TRIGGER IF EXISTS update_order_items_updated_at ON order_items;
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP TABLE IF EXISTS order_status_history CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
-- +goose StatementEnd
