-- +goose Up
-- +goose StatementBegin
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL, -- UUID for deduplication
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL, -- order, product, customer, etc.
    payload JSONB NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Index for event type queries
CREATE INDEX idx_events_type ON events (event_type);

-- Index for aggregate lookups (event sourcing)
CREATE INDEX idx_events_aggregate ON events (aggregate_type, aggregate_id);

-- Index for chronological event processing
CREATE INDEX idx_events_created_at ON events (created_at);

-- Index for event deduplication
CREATE UNIQUE INDEX idx_events_event_id ON events (event_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_event_id;
DROP INDEX IF EXISTS idx_events_created_at;
DROP INDEX IF EXISTS idx_events_aggregate;
DROP INDEX IF EXISTS idx_events_type;
DROP TABLE IF EXISTS events;

-- +goose StatementEnd
