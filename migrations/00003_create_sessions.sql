-- +goose Up
-- +goose StatementBegin

-- Sessions table: for cart persistence and authentication
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Session token
    token VARCHAR(255) NOT NULL UNIQUE,

    -- Session data (JSONB for flexibility)
    data JSONB NOT NULL DEFAULT '{}',

    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Auto-update trigger
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE sessions IS 'Sessions for cart persistence and user authentication';
COMMENT ON COLUMN sessions.token IS 'Session token (should be cryptographically secure)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TABLE IF EXISTS sessions CASCADE;
-- +goose StatementEnd
