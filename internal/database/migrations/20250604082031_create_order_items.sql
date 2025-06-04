-- +goose Up
-- +goose StatementBegin
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products (id),
    name VARCHAR(255) NOT NULL, -- snapshot of product name at time of order
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price INTEGER NOT NULL CHECK (price > 0), -- price per unit at time of order
    created_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Index for order item lookups
CREATE INDEX idx_order_items_order ON order_items (order_id);

-- Index for product sales tracking
CREATE INDEX idx_order_items_product ON order_items (product_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_order_items_product;

DROP INDEX IF EXISTS idx_order_items_order;

DROP TABLE IF EXISTS order_items;

-- +goose StatementEnd
