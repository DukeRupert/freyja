-- +goose Up
-- +goose StatementBegin
ALTER TABLE products
ADD COLUMN stripe_product_id VARCHAR(255) UNIQUE;

ALTER TABLE products
ADD COLUMN stripe_price_onetime_id VARCHAR(255);

ALTER TABLE products
ADD COLUMN stripe_price_14day_id VARCHAR(255);

ALTER TABLE products
ADD COLUMN stripe_price_21day_id VARCHAR(255);

ALTER TABLE products
ADD COLUMN stripe_price_30day_id VARCHAR(255);

ALTER TABLE products
ADD COLUMN stripe_price_60day_id VARCHAR(255);

-- Index for Stripe product lookups
CREATE INDEX idx_products_stripe_product_id ON products (stripe_product_id)
WHERE
    stripe_product_id IS NOT NULL;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_stripe_product_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_price_60day_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_price_30day_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_price_21day_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_price_14day_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_price_onetime_id;

ALTER TABLE products
DROP COLUMN IF EXISTS stripe_product_id;

-- +goose StatementEnd
