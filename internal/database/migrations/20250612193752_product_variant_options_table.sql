-- +goose Up
-- +goose StatementBegin
-- Junction table linking variants to their option value combinations
CREATE TABLE product_variant_options (
    id SERIAL PRIMARY KEY,
    product_variant_id INTEGER NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    product_option_id INTEGER NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    product_option_value_id INTEGER NOT NULL REFERENCES product_option_values(id) ON DELETE CASCADE,
    
    -- Ensure each variant has unique option combinations
    CONSTRAINT uq_variant_option_combination UNIQUE (product_variant_id, product_option_id)
);

-- Indexes for efficient joins and lookups
CREATE INDEX idx_variant_options_variant_id ON product_variant_options(product_variant_id);
CREATE INDEX idx_variant_options_option_id ON product_variant_options(product_option_id);
CREATE INDEX idx_variant_options_value_id ON product_variant_options(product_option_value_id);

-- Composite index for efficient combination lookups
CREATE INDEX idx_variant_options_combination ON product_variant_options(product_option_id, product_option_value_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_variant_options_combination;
DROP INDEX IF EXISTS idx_variant_options_value_id;
DROP INDEX IF EXISTS idx_variant_options_option_id;
DROP INDEX IF EXISTS idx_variant_options_variant_id;
DROP TABLE IF EXISTS product_variant_options;
-- +goose StatementEnd
