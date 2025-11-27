-- +goose Up
-- +goose StatementBegin

-- Sessions table: authentication sessions
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Session token (hashed for security)
    token_hash VARCHAR(255) NOT NULL UNIQUE,

    -- Session metadata
    ip_address INET,
    user_agent TEXT,

    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at) WHERE expires_at > NOW();

-- Auto-update trigger
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE sessions IS 'Authentication sessions for users';
COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of session token';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TABLE IF EXISTS sessions CASCADE;
-- +goose StatementEnd
