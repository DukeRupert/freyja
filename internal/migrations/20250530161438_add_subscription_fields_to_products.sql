-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN subscription_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE products ADD COLUMN subscription_intervals TEXT[] DEFAULT '{}';
ALTER TABLE products ADD COLUMN min_subscription_quantity INTEGER DEFAULT 1;
ALTER TABLE products ADD COLUMN max_subscription_quantity INTEGER;
ALTER TABLE products ADD COLUMN subscription_discount_percentage DECIMAL(5,2) DEFAULT 0.00;
ALTER TABLE products ADD COLUMN subscription_priority INTEGER DEFAULT 0;

-- Add constraint to ensure valid subscription discount percentage (0-100)
ALTER TABLE products ADD CONSTRAINT chk_subscription_discount_percentage 
    CHECK (subscription_discount_percentage >= 0 AND subscription_discount_percentage <= 100);

-- Add constraint to ensure min <= max quantity when both are set
ALTER TABLE products ADD CONSTRAINT chk_subscription_quantities 
    CHECK (min_subscription_quantity > 0 AND 
           (max_subscription_quantity IS NULL OR max_subscription_quantity >= min_subscription_quantity));

-- Add constraint to ensure valid subscription intervals
ALTER TABLE products ADD CONSTRAINT chk_subscription_intervals
    CHECK (subscription_intervals <@ ARRAY['weekly', 'biweekly', 'monthly', 'bimonthly', 'quarterly', 'semiannually', 'annually']::TEXT[]);

-- Add indexes for subscription queries
CREATE INDEX idx_products_subscription_enabled ON products(subscription_enabled) WHERE subscription_enabled = true;
CREATE INDEX idx_products_subscription_priority ON products(subscription_priority DESC) WHERE subscription_enabled = true;

-- Add comments for documentation
COMMENT ON COLUMN products.subscription_enabled IS 'Whether this product can be purchased as a subscription';
COMMENT ON COLUMN products.subscription_intervals IS 'Available subscription intervals (weekly, biweekly, monthly, etc.)';
COMMENT ON COLUMN products.min_subscription_quantity IS 'Minimum quantity for subscription orders';
COMMENT ON COLUMN products.max_subscription_quantity IS 'Maximum quantity for subscription orders (NULL = no limit)';
COMMENT ON COLUMN products.subscription_discount_percentage IS 'Percentage discount applied to subscription orders (0-100)';
COMMENT ON COLUMN products.subscription_priority IS 'Priority for displaying in subscription lists (higher = more prominent)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_subscription_discount_percentage;
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_subscription_quantities;
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_subscription_intervals;

DROP INDEX IF EXISTS idx_products_subscription_enabled;
DROP INDEX IF EXISTS idx_products_subscription_priority;

ALTER TABLE products DROP COLUMN IF EXISTS subscription_enabled;
ALTER TABLE products DROP COLUMN IF EXISTS subscription_intervals;
ALTER TABLE products DROP COLUMN IF EXISTS min_subscription_quantity;
ALTER TABLE products DROP COLUMN IF EXISTS max_subscription_quantity;
ALTER TABLE products DROP COLUMN IF EXISTS subscription_discount_percentage;
ALTER TABLE products DROP COLUMN IF EXISTS subscription_priority;
-- +goose StatementEnd