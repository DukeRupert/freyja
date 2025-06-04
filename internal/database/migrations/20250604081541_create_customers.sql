-- +goose Up
-- +goose StatementBegin
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    password_hash VARCHAR(255) NOT NULL,
    stripe_customer_id VARCHAR(255) UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Ensure email is lowercase and properly indexed
CREATE UNIQUE INDEX idx_customers_email ON customers (LOWER(email));

-- Index for Stripe customer lookups
CREATE INDEX idx_customers_stripe_id ON customers (stripe_customer_id)
WHERE
    stripe_customer_id IS NOT NULL;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_customers_stripe_id;

DROP INDEX IF EXISTS idx_customers_email;

DROP TABLE IF EXISTS customers;

-- +goose StatementEnd
