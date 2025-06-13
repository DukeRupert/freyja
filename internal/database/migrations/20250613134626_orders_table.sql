-- +goose Up
-- +goose StatementBegin
CREATE TYPE order_status AS ENUM (
    'pending',
    'payment_processing',
    'confirmed',
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'refunded'
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers (id),
    status order_status NOT NULL DEFAULT 'pending',
    total INTEGER NOT NULL CHECK (total > 0), -- total in cents
    stripe_session_id VARCHAR(255),
    stripe_payment_intent_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Index for customer order lookups
CREATE INDEX idx_orders_customer ON orders (customer_id);

-- Index for status-based queries (admin dashboard)
CREATE INDEX idx_orders_status ON orders (status);

-- Index for date-based queries (reporting)
CREATE INDEX idx_orders_created_at ON orders (created_at);

-- Index for Stripe session lookups
CREATE INDEX idx_orders_stripe_session ON orders (stripe_session_id)
WHERE
    stripe_session_id IS NOT NULL;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_orders_stripe_session;

DROP INDEX IF EXISTS idx_orders_created_at;

DROP INDEX IF EXISTS idx_orders_status;

DROP INDEX IF EXISTS idx_orders_customer;

DROP TABLE IF EXISTS orders;

DROP TYPE IF EXISTS order_status;

-- +goose StatementEnd
