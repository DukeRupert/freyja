-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- SaaS Onboarding: Tenant Operators and Operator Sessions
-- ============================================================================
-- This migration adds tables for tenant operators (people who manage a tenant/
-- roaster store) and their sessions. This separates platform operators from
-- storefront customers (who remain in the users table).
-- ============================================================================

-- Step 1: Add Stripe integration columns to tenants table
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS stripe_subscription_id VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS grace_period_started_at TIMESTAMPTZ;

-- Step 2: Update tenants status constraint to include new states
-- First drop the existing constraint, then add the new one
ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants
    ADD CONSTRAINT tenants_status_check
        CHECK (status IN ('pending', 'active', 'past_due', 'suspended', 'cancelled'));

-- Step 3: Create indexes for Stripe columns
CREATE INDEX IF NOT EXISTS idx_tenants_stripe_customer_id ON tenants(stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_stripe_subscription_id ON tenants(stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_grace_period ON tenants(grace_period_started_at)
    WHERE status = 'past_due';

COMMENT ON COLUMN tenants.stripe_customer_id IS 'Stripe customer ID for platform subscription billing';
COMMENT ON COLUMN tenants.stripe_subscription_id IS 'Stripe subscription ID for $149/month platform fee';
COMMENT ON COLUMN tenants.grace_period_started_at IS 'When grace period started after payment failure (7 days before suspension)';

-- Step 4: Create tenant_operators table
-- These are people who PAY for Freyja and manage a roaster's store
-- Separate from users table (which contains storefront customers)
CREATE TABLE tenant_operators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Authentication
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255), -- NULL until setup complete

    -- Profile
    name VARCHAR(255),

    -- Role (for future multi-user support)
    role VARCHAR(50) NOT NULL DEFAULT 'owner',
    -- owner: full access, billing management
    -- admin: full access except billing (future)
    -- staff: limited access (future)

    -- Setup/reset tokens (stored as SHA-256 hashes)
    setup_token_hash VARCHAR(255),
    setup_token_expires_at TIMESTAMPTZ,
    reset_token_hash VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'suspended')),
    -- pending: invited, hasn't set password
    -- active: can log in
    -- suspended: access revoked

    last_login_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_operators_tenant_email_unique UNIQUE(tenant_id, email)
);

-- Indexes for tenant_operators
CREATE INDEX idx_tenant_operators_tenant_id ON tenant_operators(tenant_id);
CREATE INDEX idx_tenant_operators_email ON tenant_operators(email);
CREATE INDEX idx_tenant_operators_setup_token ON tenant_operators(setup_token_hash)
    WHERE setup_token_hash IS NOT NULL;
CREATE INDEX idx_tenant_operators_reset_token ON tenant_operators(reset_token_hash)
    WHERE reset_token_hash IS NOT NULL;

-- Auto-update trigger
CREATE TRIGGER update_tenant_operators_updated_at
    BEFORE UPDATE ON tenant_operators
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenant_operators IS 'People who manage a tenant (roaster staff who pay for Freyja)';
COMMENT ON COLUMN tenant_operators.role IS 'owner (full access), admin (future), staff (future)';
COMMENT ON COLUMN tenant_operators.setup_token_hash IS 'SHA-256 hash of setup token sent via email (48h expiry)';
COMMENT ON COLUMN tenant_operators.reset_token_hash IS 'SHA-256 hash of password reset token (1h expiry)';

-- Step 5: Create operator_sessions table (separate from customer sessions)
CREATE TABLE operator_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operator_id UUID NOT NULL REFERENCES tenant_operators(id) ON DELETE CASCADE,

    token_hash VARCHAR(255) NOT NULL, -- SHA-256 of session token

    -- Session metadata
    user_agent TEXT,
    ip_address INET,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for operator_sessions
CREATE INDEX idx_operator_sessions_token_hash ON operator_sessions(token_hash);
CREATE INDEX idx_operator_sessions_operator_id ON operator_sessions(operator_id);
CREATE INDEX idx_operator_sessions_expires_at ON operator_sessions(expires_at);

COMMENT ON TABLE operator_sessions IS 'Sessions for tenant operators (separate from customer sessions)';
COMMENT ON COLUMN operator_sessions.token_hash IS 'SHA-256 hash of session token (not stored in plaintext)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop operator tables
DROP TRIGGER IF EXISTS update_tenant_operators_updated_at ON tenant_operators;
DROP TABLE IF EXISTS operator_sessions CASCADE;
DROP TABLE IF EXISTS tenant_operators CASCADE;

-- Drop tenant Stripe columns and indexes
DROP INDEX IF EXISTS idx_tenants_grace_period;
DROP INDEX IF EXISTS idx_tenants_stripe_subscription_id;
DROP INDEX IF EXISTS idx_tenants_stripe_customer_id;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS grace_period_started_at,
    DROP COLUMN IF EXISTS stripe_subscription_id,
    DROP COLUMN IF EXISTS stripe_customer_id;

-- Restore original status constraint
ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants
    ADD CONSTRAINT tenants_status_check
        CHECK (status IN ('active', 'suspended', 'cancelled'));

-- +goose StatementEnd
