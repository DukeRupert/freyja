-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table: represents each coffee roaster using the platform
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    website VARCHAR(255),

    -- Business details
    business_name VARCHAR(255),
    tax_id VARCHAR(50),

    -- Settings stored as JSONB for flexibility
    settings JSONB NOT NULL DEFAULT '{}',

    -- Subscription status
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled')),

    -- Trial period
    trial_ends_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for lookup by slug (used in subdomain routing)
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE status = 'active';

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenants IS 'Coffee roasters using the platform (multi-tenant root)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS "uuid-ossp";
-- +goose StatementEnd
