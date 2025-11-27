-- +goose Up
-- +goose StatementBegin

-- Products table: represents a coffee offering (e.g., "Ethiopia Yirgacheffe")
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    short_description TEXT,

    -- Coffee-specific attributes
    origin VARCHAR(100), -- Country
    region VARCHAR(100),
    producer VARCHAR(255),
    process VARCHAR(100), -- e.g., 'washed', 'natural', 'honey'
    roast_level VARCHAR(50), -- e.g., 'light', 'medium', 'dark'
    elevation_min INTEGER, -- meters
    elevation_max INTEGER, -- meters
    variety VARCHAR(255), -- e.g., 'Heirloom', 'Bourbon', 'Caturra'
    harvest_year INTEGER,

    -- Tasting notes (stored as array)
    tasting_notes TEXT[],

    -- Product status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'archived')),

    -- Visibility controls
    visibility VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'wholesale_only', 'hidden')),

    -- SEO
    meta_title VARCHAR(255),
    meta_description TEXT,

    -- Sorting
    sort_order INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT products_tenant_slug_unique UNIQUE (tenant_id, slug)
);

-- Product images
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,

    -- Image storage (URL or path to S3/local storage)
    url VARCHAR(500) NOT NULL,
    alt_text VARCHAR(255),

    -- Image metadata
    width INTEGER,
    height INTEGER,
    file_size INTEGER,

    -- Ordering
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Product SKUs: represents variants (weight + grind combinations)
CREATE TABLE product_skus (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,

    -- SKU identifier
    sku VARCHAR(100) NOT NULL,

    -- Variant attributes
    weight_value DECIMAL(10, 2) NOT NULL, -- e.g., 12, 5
    weight_unit VARCHAR(10) NOT NULL DEFAULT 'oz' CHECK (weight_unit IN ('oz', 'lb', 'g', 'kg')),
    grind VARCHAR(50) NOT NULL DEFAULT 'whole_bean',

    -- Base price (can be overridden by price lists)
    base_price_cents INTEGER NOT NULL,

    -- Inventory
    inventory_quantity INTEGER NOT NULL DEFAULT 0,
    inventory_policy VARCHAR(20) NOT NULL DEFAULT 'deny' CHECK (inventory_policy IN ('deny', 'allow')),
    low_stock_threshold INTEGER,

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Physical properties for shipping
    weight_grams INTEGER, -- Actual shipping weight
    requires_shipping BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT product_skus_tenant_sku_unique UNIQUE (tenant_id, sku)
);

-- Indexes
CREATE INDEX idx_products_tenant_id ON products(tenant_id);
CREATE INDEX idx_products_slug ON products(tenant_id, slug);
CREATE INDEX idx_products_status ON products(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_products_visibility ON products(tenant_id, visibility);
CREATE INDEX idx_products_origin ON products(origin);
CREATE INDEX idx_products_roast_level ON products(roast_level);

CREATE INDEX idx_product_images_tenant_id ON product_images(tenant_id);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);
CREATE INDEX idx_product_images_primary ON product_images(product_id, is_primary) WHERE is_primary = TRUE;

CREATE INDEX idx_product_skus_tenant_id ON product_skus(tenant_id);
CREATE INDEX idx_product_skus_product_id ON product_skus(product_id);
CREATE INDEX idx_product_skus_sku ON product_skus(tenant_id, sku);
CREATE INDEX idx_product_skus_active ON product_skus(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_product_skus_low_stock ON product_skus(tenant_id)
    WHERE inventory_quantity <= low_stock_threshold AND is_active = TRUE;

-- Auto-update triggers
CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_product_skus_updated_at
    BEFORE UPDATE ON product_skus
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE products IS 'Coffee products (base offerings)';
COMMENT ON TABLE product_skus IS 'Product variants by weight and grind';
COMMENT ON TABLE product_images IS 'Product images with ordering';
COMMENT ON COLUMN products.visibility IS 'public: all customers, wholesale_only: wholesale accounts only, hidden: not shown';
COMMENT ON COLUMN product_skus.inventory_policy IS 'deny: prevent orders when out of stock, allow: allow backorders';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_product_skus_updated_at ON product_skus;
DROP TRIGGER IF EXISTS update_products_updated_at ON products;
DROP TABLE IF EXISTS product_skus CASCADE;
DROP TABLE IF EXISTS product_images CASCADE;
DROP TABLE IF EXISTS products CASCADE;
-- +goose StatementEnd
