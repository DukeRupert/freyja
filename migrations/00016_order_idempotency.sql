-- +goose Up
-- +goose StatementBegin

-- Add unique constraint on payments table to ensure idempotent payment intent processing
-- This prevents duplicate orders from webhook retries
ALTER TABLE payments
ADD CONSTRAINT unique_provider_payment_id
UNIQUE (tenant_id, provider_payment_id);

-- Add index for fast idempotency checks when processing webhooks
-- Query: "Check if we've already created an order for this payment_intent_id"
CREATE INDEX idx_payments_provider_payment_id
ON payments (tenant_id, provider_payment_id)
WHERE provider_payment_id IS NOT NULL;

-- Add index for cart-to-order conversion tracking
-- Query: "Find all orders created from a specific cart"
CREATE INDEX idx_orders_cart_id
ON orders (tenant_id, cart_id)
WHERE cart_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_orders_cart_id;
DROP INDEX IF EXISTS idx_payments_provider_payment_id;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS unique_provider_payment_id;

-- +goose StatementEnd
