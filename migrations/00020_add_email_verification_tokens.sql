-- +goose Up
-- +goose StatementBegin

-- Email verification tokens table
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tenant and user association
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Token (stored as SHA-256 hash)
    token_hash VARCHAR(64) NOT NULL,

    -- Status tracking
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,

    -- Expiration (24 hours for email verification)
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Rate limiting metadata
    ip_address VARCHAR(45), -- IPv4 or IPv6
    user_agent TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for fast lookups
-- Partial index for unused tokens (expiration checked at query time)
CREATE INDEX idx_email_verification_tokens_token_hash
ON email_verification_tokens(token_hash)
WHERE used = FALSE;

CREATE INDEX idx_email_verification_tokens_user_id
ON email_verification_tokens(tenant_id, user_id, created_at DESC);

CREATE INDEX idx_email_verification_tokens_ip_address
ON email_verification_tokens(ip_address, created_at DESC)
WHERE ip_address IS NOT NULL;

CREATE INDEX idx_email_verification_tokens_expires_at
ON email_verification_tokens(expires_at);

COMMENT ON TABLE email_verification_tokens IS 'Email verification tokens with rate limiting';
COMMENT ON COLUMN email_verification_tokens.token_hash IS 'SHA-256 hash of the token (raw token sent to user via email)';
COMMENT ON COLUMN email_verification_tokens.ip_address IS 'IP address that requested verification (for rate limiting)';
COMMENT ON COLUMN email_verification_tokens.user_agent IS 'User agent of the request (for logging/debugging)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS email_verification_tokens CASCADE;

-- +goose StatementEnd
