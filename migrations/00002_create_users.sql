-- +goose Up
-- +goose StatementBegin

-- Users table: customer accounts (both retail and wholesale)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Authentication
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255), -- NULL for magic link only accounts
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,

    -- Account type
    account_type VARCHAR(20) NOT NULL DEFAULT 'retail' CHECK (account_type IN ('retail', 'wholesale', 'admin')),

    -- Profile
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(50),

    -- Wholesale-specific
    company_name VARCHAR(255),
    tax_id VARCHAR(50),
    business_type VARCHAR(50), -- e.g., 'cafe', 'restaurant', 'retailer'

    -- Account status
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended', 'closed')),

    -- Wholesale application
    wholesale_application_status VARCHAR(20) CHECK (wholesale_application_status IN ('pending', 'approved', 'rejected')),
    wholesale_application_notes TEXT,
    wholesale_approved_at TIMESTAMP WITH TIME ZONE,
    wholesale_approved_by UUID,

    -- Terms for wholesale (e.g., 'net_15', 'net_30')
    payment_terms VARCHAR(20),

    -- Metadata for extensibility
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure email is unique within a tenant
    CONSTRAINT users_tenant_email_unique UNIQUE (tenant_id, email)
);

-- Indexes
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_users_account_type ON users(tenant_id, account_type);
CREATE INDEX idx_users_status ON users(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_users_wholesale_application ON users(tenant_id, wholesale_application_status)
    WHERE wholesale_application_status = 'pending';

-- Auto-update trigger
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE users IS 'Customer accounts (retail and wholesale)';
COMMENT ON COLUMN users.account_type IS 'Account type: retail (default), wholesale (approved), admin (internal)';
COMMENT ON COLUMN users.payment_terms IS 'Wholesale payment terms: net_15, net_30, net_60, etc.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TABLE IF EXISTS users CASCADE;
-- +goose StatementEnd
