-- +goose Up
-- +goose StatementBegin
-- Add stripe_charge_id to orders table
ALTER TABLE orders
ADD COLUMN stripe_charge_id VARCHAR(255) NULL;

-- Add purchase type and subscription info to order_items table
ALTER TABLE order_items
ADD COLUMN purchase_type VARCHAR(20) NOT NULL DEFAULT 'one_time';

ALTER TABLE order_items
ADD COLUMN subscription_interval VARCHAR(20) NULL;

ALTER TABLE order_items
ADD COLUMN stripe_price_id VARCHAR(255) NOT NULL;

-- Add constraints for purchase_type
ALTER TABLE order_items ADD CONSTRAINT check_purchase_type CHECK (purchase_type IN ('one_time', 'subscription'));

-- Add constraint for subscription_interval
ALTER TABLE order_items ADD CONSTRAINT check_subscription_interval CHECK (
    (
        purchase_type = 'subscription'
        AND subscription_interval IN ('14_day', '21_day', '30_day', '60_day')
    )
    OR (
        purchase_type = 'one_time'
        AND subscription_interval IS NULL
    )
);

-- Create index for Stripe charge ID lookups
CREATE INDEX idx_orders_stripe_charge_id ON orders (stripe_charge_id)
WHERE
    stripe_charge_id IS NOT NULL;

-- Create index for order items purchase type
CREATE INDEX idx_order_items_purchase_type ON order_items (purchase_type);

-- Create index for subscription intervals
CREATE INDEX idx_order_items_subscription_interval ON order_items (subscription_interval)
WHERE
    subscription_interval IS NOT NULL;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Remove indexes
DROP INDEX IF EXISTS idx_order_items_subscription_interval;

DROP INDEX IF EXISTS idx_order_items_purchase_type;

DROP INDEX IF EXISTS idx_orders_stripe_charge_id;

-- Remove constraints
ALTER TABLE order_items
DROP CONSTRAINT IF EXISTS check_subscription_interval;

ALTER TABLE order_items
DROP CONSTRAINT IF EXISTS check_purchase_type;

-- Remove columns
ALTER TABLE order_items
DROP COLUMN IF EXISTS stripe_price_id;

ALTER TABLE order_items
DROP COLUMN IF EXISTS subscription_interval;

ALTER TABLE order_items
DROP COLUMN IF EXISTS purchase_type;

ALTER TABLE orders
DROP COLUMN IF EXISTS stripe_charge_id;

-- +goose StatementEnd