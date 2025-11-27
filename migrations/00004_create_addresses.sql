-- +goose Up
-- +goose StatementBegin

-- Addresses table: shipping and billing addresses
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Address fields
    full_name VARCHAR(255),
    company VARCHAR(255),
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(2) NOT NULL DEFAULT 'US', -- ISO 3166-1 alpha-2

    -- Contact
    phone VARCHAR(50),
    email VARCHAR(255),

    -- Address type
    address_type VARCHAR(20) NOT NULL DEFAULT 'shipping' CHECK (address_type IN ('shipping', 'billing', 'both')),

    -- Validation
    is_validated BOOLEAN NOT NULL DEFAULT FALSE,
    validation_metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Customer address associations
CREATE TABLE customer_addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,

    -- Flags
    is_default_shipping BOOLEAN NOT NULL DEFAULT FALSE,
    is_default_billing BOOLEAN NOT NULL DEFAULT FALSE,

    -- Label for customer's reference
    label VARCHAR(100), -- e.g., 'Home', 'Office', 'Main Warehouse'

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT customer_addresses_user_address_unique UNIQUE (user_id, address_id)
);

-- Indexes
CREATE INDEX idx_addresses_tenant_id ON addresses(tenant_id);
CREATE INDEX idx_customer_addresses_tenant_id ON customer_addresses(tenant_id);
CREATE INDEX idx_customer_addresses_user_id ON customer_addresses(user_id);
CREATE INDEX idx_customer_addresses_address_id ON customer_addresses(address_id);
CREATE INDEX idx_customer_addresses_defaults ON customer_addresses(user_id, is_default_shipping, is_default_billing);

-- Auto-update triggers
CREATE TRIGGER update_addresses_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE addresses IS 'Shipping and billing addresses';
COMMENT ON TABLE customer_addresses IS 'Links users to their saved addresses';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_addresses_updated_at ON addresses;
DROP TABLE IF EXISTS customer_addresses CASCADE;
DROP TABLE IF EXISTS addresses CASCADE;
-- +goose StatementEnd
