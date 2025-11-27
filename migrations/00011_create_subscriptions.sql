-- +goose Up
-- +goose StatementBegin

-- Subscription plans: templates for recurring subscriptions
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Plan identification
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Billing frequency
    billing_interval VARCHAR(20) NOT NULL CHECK (billing_interval IN ('weekly', 'biweekly', 'monthly', 'every_6_weeks', 'every_2_months')),

    -- Default product (optional - customers can customize)
    default_product_sku_id UUID REFERENCES product_skus(id) ON DELETE SET NULL,
    default_quantity INTEGER DEFAULT 1,

    -- Pricing (can be overridden per subscription)
    price_cents INTEGER,

    -- Settings
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    trial_period_days INTEGER DEFAULT 0,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Subscriptions: customer subscription instances
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Plan reference (optional - subscriptions can be custom)
    subscription_plan_id UUID REFERENCES subscription_plans(id) ON DELETE SET NULL,

    -- Billing frequency (stored on subscription for flexibility)
    billing_interval VARCHAR(20) NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN (
        'trial',
        'active',
        'paused',
        'past_due',
        'cancelled',
        'expired'
    )),

    -- Billing provider integration
    billing_customer_id UUID REFERENCES billing_customers(id) ON DELETE RESTRICT,
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    provider_subscription_id VARCHAR(255),

    -- Pricing
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Shipping
    shipping_address_id UUID NOT NULL REFERENCES addresses(id) ON DELETE RESTRICT,
    shipping_method_id UUID REFERENCES shipping_methods(id) ON DELETE SET NULL,
    shipping_cents INTEGER NOT NULL DEFAULT 0,

    -- Payment
    payment_method_id UUID REFERENCES payment_methods(id) ON DELETE SET NULL,

    -- Trial period
    trial_ends_at TIMESTAMP WITH TIME ZONE,

    -- Scheduling
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    next_billing_date TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Cancellation
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancellation_reason TEXT,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT subscriptions_provider_unique UNIQUE (provider, provider_subscription_id)
);

-- Subscription items: products in a subscription
CREATE TABLE subscription_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    product_sku_id UUID NOT NULL REFERENCES product_skus(id) ON DELETE RESTRICT,

    -- Quantity
    quantity INTEGER NOT NULL CHECK (quantity > 0),

    -- Pricing at time of subscription creation
    unit_price_cents INTEGER NOT NULL,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each SKU can only appear once per subscription
    CONSTRAINT subscription_items_unique UNIQUE (subscription_id, product_sku_id)
);

-- Subscription schedule: upcoming and past subscription events
CREATE TABLE subscription_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,

    -- Event type
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'billing',
        'renewal',
        'skip',
        'pause',
        'resume',
        'cancel',
        'payment_failed'
    )),

    -- Event status
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'processing', 'completed', 'failed', 'cancelled')),

    -- Related entities
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,

    -- Failure information
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add foreign key to orders table for subscription_id
ALTER TABLE orders ADD CONSTRAINT orders_subscription_id_fkey
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE SET NULL;

-- Indexes
CREATE INDEX idx_subscription_plans_tenant_id ON subscription_plans(tenant_id);
CREATE INDEX idx_subscription_plans_active ON subscription_plans(tenant_id, is_active) WHERE is_active = TRUE;

CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_plan_id ON subscriptions(subscription_plan_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_provider ON subscriptions(provider, provider_subscription_id);
CREATE INDEX idx_subscriptions_next_billing ON subscriptions(tenant_id, next_billing_date)
    WHERE status IN ('trial', 'active');
CREATE INDEX idx_subscriptions_active ON subscriptions(tenant_id, status)
    WHERE status IN ('trial', 'active');

CREATE INDEX idx_subscription_items_tenant_id ON subscription_items(tenant_id);
CREATE INDEX idx_subscription_items_subscription_id ON subscription_items(subscription_id);
CREATE INDEX idx_subscription_items_product_sku_id ON subscription_items(product_sku_id);

CREATE INDEX idx_subscription_schedule_tenant_id ON subscription_schedule(tenant_id);
CREATE INDEX idx_subscription_schedule_subscription_id ON subscription_schedule(subscription_id);
CREATE INDEX idx_subscription_schedule_status ON subscription_schedule(status)
    WHERE status IN ('scheduled', 'failed');
CREATE INDEX idx_subscription_schedule_scheduled_at ON subscription_schedule(scheduled_at)
    WHERE status = 'scheduled';

-- Auto-update triggers
CREATE TRIGGER update_subscription_plans_updated_at
    BEFORE UPDATE ON subscription_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscription_items_updated_at
    BEFORE UPDATE ON subscription_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscription_schedule_updated_at
    BEFORE UPDATE ON subscription_schedule
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE subscription_plans IS 'Templates for recurring subscriptions';
COMMENT ON TABLE subscriptions IS 'Customer subscription instances';
COMMENT ON TABLE subscription_items IS 'Products included in subscriptions';
COMMENT ON TABLE subscription_schedule IS 'Upcoming and past subscription events';
COMMENT ON COLUMN subscriptions.billing_interval IS 'weekly, biweekly, monthly, every_6_weeks, every_2_months';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_subscription_id_fkey;

DROP TRIGGER IF EXISTS update_subscription_schedule_updated_at ON subscription_schedule;
DROP TRIGGER IF EXISTS update_subscription_items_updated_at ON subscription_items;
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
DROP TRIGGER IF EXISTS update_subscription_plans_updated_at ON subscription_plans;

DROP TABLE IF EXISTS subscription_schedule CASCADE;
DROP TABLE IF EXISTS subscription_items CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS subscription_plans CASCADE;
-- +goose StatementEnd
