-- +goose Up
-- +goose StatementBegin

-- Carts: shopping carts for both guests and authenticated users
CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- User association (NULL for guest carts)
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Session association (for guest cart persistence)
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,

    -- Cart status
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'abandoned', 'converted', 'expired')),

    -- Cart metadata
    notes TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Timestamps
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    converted_to_order_id UUID, -- Set when cart becomes an order
    expires_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Cart items: line items in a cart
CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_sku_id UUID NOT NULL REFERENCES product_skus(id) ON DELETE CASCADE,

    -- Quantity
    quantity INTEGER NOT NULL CHECK (quantity > 0),

    -- Price at time of adding (captured for stability)
    unit_price_cents INTEGER NOT NULL,

    -- Item metadata (e.g., gift message, special instructions)
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Prevent duplicate SKUs in the same cart (quantity should be updated instead)
    CONSTRAINT cart_items_cart_sku_unique UNIQUE (cart_id, product_sku_id)
);

-- Indexes
CREATE INDEX idx_carts_tenant_id ON carts(tenant_id);
CREATE INDEX idx_carts_user_id ON carts(user_id);
CREATE INDEX idx_carts_session_id ON carts(session_id);
CREATE INDEX idx_carts_status ON carts(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_carts_last_activity ON carts(last_activity_at);
-- Note: Index without predicate - abandoned carts can be found via query
CREATE INDEX idx_carts_abandoned ON carts(tenant_id, status, last_activity_at)
    WHERE status = 'active';

CREATE INDEX idx_cart_items_tenant_id ON cart_items(tenant_id);
CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX idx_cart_items_product_sku_id ON cart_items(product_sku_id);

-- Auto-update triggers
CREATE TRIGGER update_carts_updated_at
    BEFORE UPDATE ON carts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cart_items_updated_at
    BEFORE UPDATE ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update cart's last_activity_at when items change
CREATE OR REPLACE FUNCTION update_cart_last_activity()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE carts
    SET last_activity_at = NOW()
    WHERE id = COALESCE(NEW.cart_id, OLD.cart_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_cart_activity_on_item_change
    AFTER INSERT OR UPDATE OR DELETE ON cart_items
    FOR EACH ROW
    EXECUTE FUNCTION update_cart_last_activity();

COMMENT ON TABLE carts IS 'Shopping carts for guests and authenticated users';
COMMENT ON TABLE cart_items IS 'Line items in shopping carts';
COMMENT ON COLUMN carts.user_id IS 'NULL for guest carts, set for authenticated users';
COMMENT ON COLUMN carts.session_id IS 'Links guest carts to sessions for persistence';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_cart_activity_on_item_change ON cart_items;
DROP FUNCTION IF EXISTS update_cart_last_activity();
DROP TRIGGER IF EXISTS update_cart_items_updated_at ON cart_items;
DROP TRIGGER IF EXISTS update_carts_updated_at ON carts;
DROP TABLE IF EXISTS cart_items CASCADE;
DROP TABLE IF EXISTS carts CASCADE;
-- +goose StatementEnd
