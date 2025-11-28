-- +goose Up
-- +goose StatementBegin
ALTER TABLE products 
ADD COLUMN is_white_label BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN base_product_id UUID REFERENCES products(id) ON DELETE CASCADE,
ADD COLUMN white_label_customer_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_products_white_label ON products(tenant_id, is_white_label) 
  WHERE is_white_label = TRUE;
CREATE INDEX idx_products_base_product ON products(base_product_id) 
  WHERE base_product_id IS NOT NULL;
CREATE INDEX idx_products_white_label_customer ON products(white_label_customer_id) 
  WHERE white_label_customer_id IS NOT NULL;

ALTER TABLE products
ADD CONSTRAINT check_white_label_consistency 
CHECK (
  (is_white_label = FALSE AND base_product_id IS NULL AND white_label_customer_id IS NULL) OR
  (is_white_label = TRUE AND base_product_id IS NOT NULL AND white_label_customer_id IS NOT NULL)
);

COMMENT ON COLUMN products.is_white_label IS 'TRUE if this product is a white-label variant of a base product';
COMMENT ON COLUMN products.base_product_id IS 'References the base product this white-label is derived from';
COMMENT ON COLUMN products.white_label_customer_id IS 'The specific customer (user) this white-label product is restricted to';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products 
DROP CONSTRAINT IF EXISTS check_white_label_consistency,
DROP COLUMN IF EXISTS white_label_customer_id,
DROP COLUMN IF EXISTS base_product_id,
DROP COLUMN IF EXISTS is_white_label;
-- +goose StatementEnd