-- +goose Up
-- +goose StatementBegin

CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_variant_id INTEGER NOT NULL REFERENCES product_variants (id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price INTEGER NOT NULL CHECK (price > 0), -- locked-in price at time of add
    purchase_type VARCHAR(20) NOT NULL CHECK (purchase_type IN ('one_time', 'subscription')),
    subscription_interval VARCHAR(20) CHECK (
        (
            purchase_type = 'subscription'
            AND subscription_interval IN ('14_day', '21_day', '30_day', '60_day')
        )
        OR (
            purchase_type = 'one_time'
            AND subscription_interval IS NULL
        )
    ),
    stripe_price_id VARCHAR(255) NOT NULL, -- The Stripe Price ID to use for checkout
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Prevent duplicate variants with same purchase type and interval in same cart
    UNIQUE (
        cart_id,
        product_variant_id,
        purchase_type,
        subscription_interval
    )
);

-- Index for cart item lookups
CREATE INDEX idx_cart_items_cart ON cart_items (cart_id);

-- Index for variant usage tracking
CREATE INDEX idx_cart_items_variant ON cart_items (product_variant_id);

-- Index for purchase type filtering
CREATE INDEX idx_cart_items_purchase_type ON cart_items (purchase_type);

-- Composite index for efficient cart + variant lookups
CREATE INDEX idx_cart_items_cart_variant ON cart_items (cart_id, product_variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_cart_items_cart_variant;
DROP INDEX IF EXISTS idx_cart_items_purchase_type;
DROP INDEX IF EXISTS idx_cart_items_variant;
DROP INDEX IF EXISTS idx_cart_items_cart;
DROP TABLE IF EXISTS cart_items;

-- +goose StatementEnd