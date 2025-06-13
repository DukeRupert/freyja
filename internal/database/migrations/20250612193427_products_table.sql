-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for active products (most common query)
CREATE INDEX idx_products_active ON products (active) WHERE active = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_active;
DROP TABLE IF EXISTS products;
-- +goose StatementEnd
