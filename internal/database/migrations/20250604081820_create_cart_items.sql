-- +goose Up
-- +goose StatementBegin
CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products (id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price INTEGER NOT NULL CHECK (price > 0), -- locked-in price at time of add
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    -- Prevent duplicate products in same cart
    UNIQUE (cart_id, product_id)
);

-- Index for cart item lookups
CREATE INDEX idx_cart_items_cart ON cart_items (cart_id);

-- Index for product usage tracking
CREATE INDEX idx_cart_items_product ON cart_items (product_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cart_items_product;

DROP INDEX IF EXISTS idx_cart_items_cart;

DROP TABLE IF EXISTS cart_items;

-- +goose StatementEnd
