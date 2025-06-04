-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price INTEGER NOT NULL CHECK (price > 0), -- cents
    stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Create index for active products (most common query)
CREATE INDEX idx_products_active ON products (active)
WHERE
    active = true;

-- Create index for price range queries
CREATE INDEX idx_products_price ON products (price);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_price;

DROP INDEX IF EXISTS idx_products_active;

DROP TABLE IF EXISTS products;

-- +goose StatementEnd
