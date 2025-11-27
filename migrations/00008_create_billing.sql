-- +goose Up
-- +goose StatementBegin

-- Billing customers: maps users to payment provider customer IDs
CREATE TABLE billing_customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Payment provider
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe' CHECK (provider IN ('stripe', 'manual')),
    provider_customer_id VARCHAR(255) NOT NULL,

    -- Customer metadata from provider
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each user can only have one billing customer per provider
    CONSTRAINT billing_customers_unique UNIQUE (user_id, provider)
);

-- Payment methods: stored payment methods (credit cards, bank accounts, etc.)
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    billing_customer_id UUID NOT NULL REFERENCES billing_customers(id) ON DELETE CASCADE,

    -- Payment provider
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    provider_payment_method_id VARCHAR(255) NOT NULL,

    -- Payment method type
    method_type VARCHAR(50) NOT NULL CHECK (method_type IN ('card', 'bank_account', 'other')),

    -- Display information (e.g., "Visa •••• 4242", "Bank •••• 6789")
    display_brand VARCHAR(50),
    display_last4 VARCHAR(4),
    display_exp_month INTEGER,
    display_exp_year INTEGER,

    -- Flags
    is_default BOOLEAN NOT NULL DEFAULT FALSE,

    -- Provider metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_methods_provider_unique UNIQUE (provider, provider_payment_method_id)
);

-- Payments: tracks payment attempts and completion
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    billing_customer_id UUID NOT NULL REFERENCES billing_customers(id) ON DELETE CASCADE,

    -- Payment provider
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    provider_payment_id VARCHAR(255) NOT NULL, -- e.g., Stripe Payment Intent ID

    -- Amount
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending',
        'processing',
        'succeeded',
        'failed',
        'cancelled',
        'refunded',
        'partially_refunded'
    )),

    -- Payment method used
    payment_method_id UUID REFERENCES payment_methods(id) ON DELETE SET NULL,

    -- Failure information
    failure_code VARCHAR(100),
    failure_message TEXT,

    -- Refund information
    refunded_amount_cents INTEGER NOT NULL DEFAULT 0,

    -- Provider metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Timestamps
    succeeded_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT payments_provider_unique UNIQUE (provider, provider_payment_id)
);

-- Webhook events: tracks incoming webhooks for idempotency
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Provider information
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    provider_event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,

    -- Processing status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'processed', 'failed')),

    -- Event payload
    payload JSONB NOT NULL,

    -- Processing information
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure we only process each provider event once
    CONSTRAINT webhook_events_provider_unique UNIQUE (provider, provider_event_id)
);

-- Indexes
CREATE INDEX idx_billing_customers_tenant_id ON billing_customers(tenant_id);
CREATE INDEX idx_billing_customers_user_id ON billing_customers(user_id);
CREATE INDEX idx_billing_customers_provider ON billing_customers(provider, provider_customer_id);

CREATE INDEX idx_payment_methods_tenant_id ON payment_methods(tenant_id);
CREATE INDEX idx_payment_methods_billing_customer_id ON payment_methods(billing_customer_id);
CREATE INDEX idx_payment_methods_default ON payment_methods(billing_customer_id, is_default)
    WHERE is_default = TRUE;

CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX idx_payments_billing_customer_id ON payments(billing_customer_id);
CREATE INDEX idx_payments_provider ON payments(provider, provider_payment_id);
CREATE INDEX idx_payments_status ON payments(tenant_id, status);
CREATE INDEX idx_payments_created_at ON payments(created_at);

CREATE INDEX idx_webhook_events_tenant_id ON webhook_events(tenant_id);
CREATE INDEX idx_webhook_events_provider ON webhook_events(provider, provider_event_id);
CREATE INDEX idx_webhook_events_status ON webhook_events(status) WHERE status IN ('pending', 'failed');
CREATE INDEX idx_webhook_events_created_at ON webhook_events(created_at);

-- Auto-update triggers
CREATE TRIGGER update_billing_customers_updated_at
    BEFORE UPDATE ON billing_customers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payment_methods_updated_at
    BEFORE UPDATE ON payment_methods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_webhook_events_updated_at
    BEFORE UPDATE ON webhook_events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE billing_customers IS 'Maps users to payment provider customer IDs';
COMMENT ON TABLE payment_methods IS 'Stored payment methods for customers';
COMMENT ON TABLE payments IS 'Payment transactions and status tracking';
COMMENT ON TABLE webhook_events IS 'Incoming webhook events for idempotent processing';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_webhook_events_updated_at ON webhook_events;
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP TRIGGER IF EXISTS update_payment_methods_updated_at ON payment_methods;
DROP TRIGGER IF EXISTS update_billing_customers_updated_at ON billing_customers;
DROP TABLE IF EXISTS webhook_events CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS payment_methods CASCADE;
DROP TABLE IF EXISTS billing_customers CASCADE;
-- +goose StatementEnd
