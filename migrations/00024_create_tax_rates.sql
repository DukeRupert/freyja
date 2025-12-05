-- +goose Up
-- +goose StatementBegin

-- Create tax_rates table to store state-level tax configuration
CREATE TABLE tax_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    state VARCHAR(2) NOT NULL,
    rate DECIMAL(5,4) NOT NULL CHECK (rate >= 0.0000 AND rate <= 1.0000),
    tax_shipping BOOLEAN NOT NULL DEFAULT TRUE,
    name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tax_rates_tenant_state_unique UNIQUE (tenant_id, state),
    CONSTRAINT tax_rates_state_code_check CHECK (
        state IN (
            'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
            'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
            'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
            'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
            'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY',
            'DC'
        )
    )
);

-- Create indexes for efficient lookups
CREATE INDEX idx_tax_rates_tenant_id ON tax_rates(tenant_id);
CREATE INDEX idx_tax_rates_tenant_active ON tax_rates(tenant_id, is_active) WHERE is_active = TRUE;

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_tax_rates_updated_at
    BEFORE UPDATE ON tax_rates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS update_tax_rates_updated_at ON tax_rates;
DROP INDEX IF EXISTS idx_tax_rates_tenant_active;
DROP INDEX IF EXISTS idx_tax_rates_tenant_id;
DROP TABLE IF EXISTS tax_rates;

-- +goose StatementEnd
