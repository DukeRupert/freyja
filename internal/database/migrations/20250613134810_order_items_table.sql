-- +goose Up
-- +goose StatementBegin

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_variant_id INTEGER NOT NULL REFERENCES product_variants (id),
    name VARCHAR(255) NOT NULL, -- snapshot of product name at time of order
    variant_name VARCHAR(500) NOT NULL, -- snapshot of variant name at time of order
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price INTEGER NOT NULL CHECK (price > 0), -- price per unit at time of order
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for order item lookups
CREATE INDEX idx_order_items_order ON order_items (order_id);

-- Index for variant sales tracking
CREATE INDEX idx_order_items_variant ON order_items (product_variant_id);

-- Composite index for efficient order + variant lookups
CREATE INDEX idx_order_items_order_variant ON order_items (order_id, product_variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_order_items_order_variant;
DROP INDEX IF EXISTS idx_order_items_variant;
DROP INDEX IF EXISTS idx_order_items_order;
DROP TABLE IF EXISTS order_items;

-- +goose StatementEnd