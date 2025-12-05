-- +goose Up
-- +goose StatementBegin

-- Tenant provider configurations: stores encrypted API keys and settings for external providers
CREATE TABLE tenant_provider_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Provider type and name
    type VARCHAR(50) NOT NULL CHECK (type IN ('tax', 'shipping', 'billing', 'email')),
    name VARCHAR(50) NOT NULL CHECK (name IN (
        -- Tax providers
        'stripe_tax', 'taxjar', 'avalara', 'percentage', 'no_tax',
        -- Shipping providers
        'shipstation', 'easypost', 'shippo', 'manual',
        -- Billing providers
        'stripe',
        -- Email providers
        'postmark', 'resend', 'ses', 'smtp'
    )),

    -- Status flags
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,

    -- Priority for selecting provider when multiple are active (lower number = higher priority)
    -- Default 100, allows for 99 higher priority and unlimited lower priority providers
    priority INTEGER NOT NULL DEFAULT 100,

    -- Encrypted configuration JSON (API keys, secrets, provider-specific settings)
    -- Encrypted using AES-256-GCM, stored as base64
    -- Example decrypted structure for Stripe:
    --   {"api_key": "sk_test_...", "webhook_secret": "whsec_..."}
    -- Example for percentage tax:
    --   {"rate": 0.08}
    config_encrypted TEXT NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each tenant can have multiple configs per type (for fallback/testing)
    -- But only one can be default per type
    CONSTRAINT tenant_provider_configs_unique UNIQUE (tenant_id, type, name)
);

-- Shipping rates: cached or manual rates for shipping calculations
CREATE TABLE tenant_shipping_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_config_id UUID NOT NULL REFERENCES tenant_provider_configs(id) ON DELETE CASCADE,

    -- Service identification
    service_code VARCHAR(100) NOT NULL, -- e.g., "usps_priority", "fedex_ground"
    service_name VARCHAR(255) NOT NULL, -- e.g., "USPS Priority Mail", "FedEx Ground"

    -- Rate lookup key (for caching provider rates)
    origin_postal_code VARCHAR(20),
    destination_postal_code VARCHAR(20) NOT NULL,
    weight_grams INTEGER NOT NULL,

    -- Rate information
    rate_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Cache expiration (for provider-fetched rates)
    -- NULL for manual rates (never expire)
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Provider-specific metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for tenant_provider_configs
CREATE INDEX idx_tenant_provider_configs_tenant_id ON tenant_provider_configs(tenant_id);
CREATE INDEX idx_tenant_provider_configs_type ON tenant_provider_configs(tenant_id, type);
CREATE INDEX idx_tenant_provider_configs_active ON tenant_provider_configs(tenant_id, type, is_active)
    WHERE is_active = TRUE;
CREATE INDEX idx_tenant_provider_configs_default ON tenant_provider_configs(tenant_id, type, is_default)
    WHERE is_default = TRUE;

-- Indexes for tenant_shipping_rates
CREATE INDEX idx_tenant_shipping_rates_tenant_id ON tenant_shipping_rates(tenant_id);
CREATE INDEX idx_tenant_shipping_rates_provider_config_id ON tenant_shipping_rates(provider_config_id);
CREATE INDEX idx_tenant_shipping_rates_lookup ON tenant_shipping_rates(
    tenant_id,
    destination_postal_code,
    weight_grams
);
CREATE INDEX idx_tenant_shipping_rates_valid_until ON tenant_shipping_rates(valid_until)
    WHERE valid_until IS NOT NULL;

-- Ensure only one default provider per tenant per type
CREATE UNIQUE INDEX idx_tenant_provider_configs_one_default_per_type
    ON tenant_provider_configs(tenant_id, type)
    WHERE is_default = TRUE;

-- Auto-update triggers
CREATE TRIGGER update_tenant_provider_configs_updated_at
    BEFORE UPDATE ON tenant_provider_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE tenant_provider_configs IS 'Tenant-specific provider configurations with encrypted credentials';
COMMENT ON COLUMN tenant_provider_configs.type IS 'Category of provider: tax, shipping, billing, email';
COMMENT ON COLUMN tenant_provider_configs.name IS 'Specific provider implementation name';
COMMENT ON COLUMN tenant_provider_configs.is_default IS 'True if this is the default provider for this type (only one per tenant per type)';
COMMENT ON COLUMN tenant_provider_configs.priority IS 'Selection priority when multiple active providers exist (lower = higher priority)';
COMMENT ON COLUMN tenant_provider_configs.config_encrypted IS 'AES-256-GCM encrypted JSON configuration including API keys';

COMMENT ON TABLE tenant_shipping_rates IS 'Cached or manual shipping rates for rate lookup';
COMMENT ON COLUMN tenant_shipping_rates.valid_until IS 'Cache expiration for provider rates, NULL for manual rates';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tenant_provider_configs_updated_at ON tenant_provider_configs;
DROP INDEX IF EXISTS idx_tenant_provider_configs_one_default_per_type;
DROP INDEX IF EXISTS idx_tenant_shipping_rates_valid_until;
DROP INDEX IF EXISTS idx_tenant_shipping_rates_lookup;
DROP INDEX IF EXISTS idx_tenant_shipping_rates_provider_config_id;
DROP INDEX IF EXISTS idx_tenant_shipping_rates_tenant_id;
DROP INDEX IF EXISTS idx_tenant_provider_configs_default;
DROP INDEX IF EXISTS idx_tenant_provider_configs_active;
DROP INDEX IF EXISTS idx_tenant_provider_configs_type;
DROP INDEX IF EXISTS idx_tenant_provider_configs_tenant_id;
DROP TABLE IF EXISTS tenant_shipping_rates CASCADE;
DROP TABLE IF EXISTS tenant_provider_configs CASCADE;
-- +goose StatementEnd
