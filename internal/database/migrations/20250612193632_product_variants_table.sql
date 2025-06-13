-- +goose Up
-- +goose StatementBegin
-- Product variants table (max 24 per product)
CREATE TABLE product_variants (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL, -- Auto-generated from options
    price INTEGER NOT NULL CHECK (price > 0), -- cents
    stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    is_subscription BOOLEAN NOT NULL DEFAULT false,
    archived_at TIMESTAMP NULL, -- For soft delete
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Stripe integration fields
    stripe_product_id TEXT,
    stripe_price_onetime_id TEXT,
    stripe_price_14day_id TEXT,
    stripe_price_21day_id TEXT,
    stripe_price_30day_id TEXT,
    stripe_price_60day_id TEXT
);

-- Indexes for efficient variant querying
CREATE INDEX idx_product_variants_product_id ON product_variants(product_id) WHERE archived_at IS NULL;
CREATE INDEX idx_product_variants_active ON product_variants(active) WHERE active = true AND archived_at IS NULL;
CREATE INDEX idx_product_variants_price ON product_variants(price) WHERE archived_at IS NULL;
CREATE INDEX idx_product_variants_subscription ON product_variants(is_subscription) WHERE is_subscription = true AND archived_at IS NULL;
CREATE INDEX idx_product_variants_stripe_product ON product_variants(stripe_product_id) WHERE stripe_product_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_variants_stripe_product;
DROP INDEX IF EXISTS idx_product_variants_subscription;
DROP INDEX IF EXISTS idx_product_variants_price;
DROP INDEX IF EXISTS idx_product_variants_active;
DROP INDEX IF EXISTS idx_product_variants_product_id;
DROP TABLE IF EXISTS product_variants;
-- +goose StatementEnd
