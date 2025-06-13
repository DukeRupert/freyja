-- +goose Up
-- +goose StatementBegin
-- Product options table (max 3 per product)
CREATE TABLE product_options (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    option_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure unique option keys per product
    CONSTRAINT uq_product_options_key UNIQUE (product_id, option_key)
);

-- Index for efficient product option lookups
CREATE INDEX idx_product_options_product_id ON product_options(product_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_options_product_id;
DROP TABLE IF EXISTS product_options;
-- +goose StatementEnd
