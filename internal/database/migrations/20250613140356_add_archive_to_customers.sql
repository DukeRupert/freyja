-- +goose Up
-- +goose StatementBegin
ALTER TABLE customers ADD COLUMN archived_at TIMESTAMP NULL;

-- Create index for soft delete queries (active customers)
CREATE INDEX idx_customers_archived_at ON customers (archived_at) WHERE archived_at IS NULL;

-- Create index for archived customers
CREATE INDEX idx_customers_archived ON customers (archived_at) WHERE archived_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_customers_archived;
DROP INDEX IF EXISTS idx_customers_archived_at;
ALTER TABLE customers DROP COLUMN IF EXISTS archived_at;
-- +goose StatementEnd
