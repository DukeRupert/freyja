-- +goose Up
-- +goose StatementBegin
-- Product option values table (max 8 per option)
CREATE TABLE product_option_values (
    id SERIAL PRIMARY KEY,
    product_option_id INTEGER NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure unique values per option
    CONSTRAINT uq_product_option_values UNIQUE (product_option_id, value)
);

-- Index for efficient option value lookups
CREATE INDEX idx_product_option_values_option_id ON product_option_values(product_option_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_option_values_option_id;
DROP TABLE IF EXISTS product_option_values;
-- +goose StatementEnd
