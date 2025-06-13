-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES customers (id),
    session_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    changes JSONB,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Index for user activity tracking
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id)
WHERE
    user_id IS NOT NULL;

-- Index for session tracking
CREATE INDEX idx_audit_logs_session ON audit_logs (session_id)
WHERE
    session_id IS NOT NULL;

-- Index for action-based queries
CREATE INDEX idx_audit_logs_action ON audit_logs (action);

-- Index for resource-based queries
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id);

-- Index for time-based queries (compliance)
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_session;
DROP INDEX IF EXISTS idx_audit_logs_user;
DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd
