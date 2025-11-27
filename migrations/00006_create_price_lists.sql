-- +goose Up
-- +goose StatementBegin

-- Price lists: named pricing tiers (e.g., "Retail", "Café Tier 1", "Restaurant Tier 2")
CREATE TABLE price_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Identification
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Type
    list_type VARCHAR(20) NOT NULL DEFAULT 'custom' CHECK (list_type IN ('default', 'wholesale', 'custom')),

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT price_lists_tenant_name_unique UNIQUE (tenant_id, name)
);

-- Price list entries: per-SKU pricing for each price list
CREATE TABLE price_list_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    price_list_id UUID NOT NULL REFERENCES price_lists(id) ON DELETE CASCADE,
    product_sku_id UUID NOT NULL REFERENCES product_skus(id) ON DELETE CASCADE,

    -- Pricing
    price_cents INTEGER NOT NULL,

    -- Optional: compare-at price for displaying discounts
    compare_at_price_cents INTEGER,

    -- Availability: if FALSE, this SKU is not available on this price list
    is_available BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each SKU can only appear once per price list
    CONSTRAINT price_list_entries_unique UNIQUE (price_list_id, product_sku_id)
);

-- User price list assignment: which price list applies to each user
CREATE TABLE user_price_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price_list_id UUID NOT NULL REFERENCES price_lists(id) ON DELETE CASCADE,

    -- Assignment metadata
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    notes TEXT,

    -- Each user can only have one price list
    CONSTRAINT user_price_lists_unique UNIQUE (user_id)
);

-- Indexes
CREATE INDEX idx_price_lists_tenant_id ON price_lists(tenant_id);
CREATE INDEX idx_price_lists_active ON price_lists(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_price_lists_type ON price_lists(tenant_id, list_type);

CREATE INDEX idx_price_list_entries_tenant_id ON price_list_entries(tenant_id);
CREATE INDEX idx_price_list_entries_price_list_id ON price_list_entries(price_list_id);
CREATE INDEX idx_price_list_entries_product_sku_id ON price_list_entries(product_sku_id);
CREATE INDEX idx_price_list_entries_available ON price_list_entries(price_list_id, is_available)
    WHERE is_available = TRUE;

CREATE INDEX idx_user_price_lists_tenant_id ON user_price_lists(tenant_id);
CREATE INDEX idx_user_price_lists_user_id ON user_price_lists(user_id);
CREATE INDEX idx_user_price_lists_price_list_id ON user_price_lists(price_list_id);

-- Auto-update triggers
CREATE TRIGGER update_price_lists_updated_at
    BEFORE UPDATE ON price_lists
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_price_list_entries_updated_at
    BEFORE UPDATE ON price_list_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE price_lists IS 'Named pricing tiers (retail, wholesale, custom)';
COMMENT ON TABLE price_list_entries IS 'Per-SKU pricing for each price list';
COMMENT ON TABLE user_price_lists IS 'Price list assignment to users';
COMMENT ON COLUMN price_lists.list_type IS 'default: used for guests/unassigned, wholesale: for wholesale accounts, custom: special pricing';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_price_list_entries_updated_at ON price_list_entries;
DROP TRIGGER IF EXISTS update_price_lists_updated_at ON price_lists;
DROP TABLE IF EXISTS user_price_lists CASCADE;
DROP TABLE IF EXISTS price_list_entries CASCADE;
DROP TABLE IF EXISTS price_lists CASCADE;
-- +goose StatementEnd
