-- +goose Up
-- +goose StatementBegin
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers (id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW (),
    -- Ensure cart belongs to either a customer OR has a session
    CONSTRAINT cart_owner_check CHECK (
        (
            customer_id IS NOT NULL
            AND session_id IS NULL
        )
        OR (
            customer_id IS NULL
            AND session_id IS NOT NULL
        )
    )
);

-- Index for customer cart lookups
CREATE INDEX idx_carts_customer ON carts (customer_id);

-- Index for session cart lookups (guest users)
CREATE INDEX idx_carts_session ON carts (session_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_carts_session;

DROP INDEX IF EXISTS idx_carts_customer;

DROP TABLE IF EXISTS carts;
-- +goose StatementEnd
