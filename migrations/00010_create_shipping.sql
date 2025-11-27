-- +goose Up
-- +goose StatementBegin

-- Shipping methods: available shipping options
CREATE TABLE shipping_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Method identification
    name VARCHAR(255) NOT NULL,
    code VARCHAR(100) NOT NULL, -- e.g., 'usps_priority', 'fedex_ground', 'manual_flat'
    description TEXT,

    -- Provider integration
    provider VARCHAR(50) NOT NULL DEFAULT 'manual' CHECK (provider IN ('manual', 'easypost', 'shippo', 'shipstation')),
    provider_service_code VARCHAR(100), -- e.g., 'Priority' for USPS

    -- Pricing (for manual methods)
    flat_rate_cents INTEGER,
    free_shipping_threshold_cents INTEGER, -- Free if order total exceeds this

    -- Settings
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Estimated delivery (for display)
    estimated_days_min INTEGER,
    estimated_days_max INTEGER,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT shipping_methods_tenant_code_unique UNIQUE (tenant_id, code)
);

-- Shipping rates: calculated rates for shipping methods (cached from providers)
CREATE TABLE shipping_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipping_method_id UUID NOT NULL REFERENCES shipping_methods(id) ON DELETE CASCADE,

    -- Rate calculation context
    origin_postal_code VARCHAR(20),
    destination_postal_code VARCHAR(20) NOT NULL,
    weight_grams INTEGER NOT NULL,

    -- Rate details
    rate_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Validity
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Provider details
    provider_rate_id VARCHAR(255), -- For provider-calculated rates
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Shipments: tracking shipment of orders
CREATE TABLE shipments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    -- Shipment identification
    shipment_number VARCHAR(50) NOT NULL,

    -- Shipping details
    shipping_method_id UUID REFERENCES shipping_methods(id) ON DELETE SET NULL,
    carrier VARCHAR(100),
    service_name VARCHAR(100),

    -- Tracking
    tracking_number VARCHAR(255),
    tracking_url TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending',
        'label_created',
        'in_transit',
        'out_for_delivery',
        'delivered',
        'failed',
        'cancelled',
        'returned'
    )),

    -- Costs
    shipping_cost_cents INTEGER NOT NULL DEFAULT 0,
    label_cost_cents INTEGER, -- Actual cost paid to carrier

    -- Package details
    weight_grams INTEGER,
    length_cm DECIMAL(10, 2),
    width_cm DECIMAL(10, 2),
    height_cm DECIMAL(10, 2),

    -- Provider integration
    provider VARCHAR(50),
    provider_shipment_id VARCHAR(255),
    provider_label_id VARCHAR(255),
    label_url TEXT, -- URL to download shipping label

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Timestamps
    label_created_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT shipments_tenant_number_unique UNIQUE (tenant_id, shipment_number)
);

-- Shipment items: links order items to shipments (for partial shipments)
CREATE TABLE shipment_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    order_item_id UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,

    -- Quantity being shipped (can be less than order item quantity for partial fulfillment)
    quantity INTEGER NOT NULL CHECK (quantity > 0),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each order item can only appear once per shipment
    CONSTRAINT shipment_items_unique UNIQUE (shipment_id, order_item_id)
);

-- Shipment tracking events: history of tracking status updates
CREATE TABLE shipment_tracking_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,

    -- Event details
    status VARCHAR(50) NOT NULL,
    message TEXT,
    location VARCHAR(255),

    -- Event metadata from provider
    provider_event_id VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}',

    event_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_shipping_methods_tenant_id ON shipping_methods(tenant_id);
CREATE INDEX idx_shipping_methods_active ON shipping_methods(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_shipping_methods_code ON shipping_methods(tenant_id, code);

CREATE INDEX idx_shipping_rates_tenant_id ON shipping_rates(tenant_id);
CREATE INDEX idx_shipping_rates_method_id ON shipping_rates(shipping_method_id);
-- Note: Index without predicate - valid rates can be found via query
CREATE INDEX idx_shipping_rates_valid ON shipping_rates(destination_postal_code, weight_grams, valid_until);

CREATE INDEX idx_shipments_tenant_id ON shipments(tenant_id);
CREATE INDEX idx_shipments_order_id ON shipments(order_id);
CREATE INDEX idx_shipments_tracking_number ON shipments(tracking_number);
CREATE INDEX idx_shipments_status ON shipments(tenant_id, status);
CREATE INDEX idx_shipments_created_at ON shipments(created_at);

CREATE INDEX idx_shipment_items_tenant_id ON shipment_items(tenant_id);
CREATE INDEX idx_shipment_items_shipment_id ON shipment_items(shipment_id);
CREATE INDEX idx_shipment_items_order_item_id ON shipment_items(order_item_id);

CREATE INDEX idx_shipment_tracking_events_tenant_id ON shipment_tracking_events(tenant_id);
CREATE INDEX idx_shipment_tracking_events_shipment_id ON shipment_tracking_events(shipment_id);
CREATE INDEX idx_shipment_tracking_events_event_at ON shipment_tracking_events(event_at);

-- Auto-update triggers
CREATE TRIGGER update_shipping_methods_updated_at
    BEFORE UPDATE ON shipping_methods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_shipments_updated_at
    BEFORE UPDATE ON shipments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE shipping_methods IS 'Available shipping options (manual and provider-integrated)';
COMMENT ON TABLE shipping_rates IS 'Cached shipping rates from providers';
COMMENT ON TABLE shipments IS 'Shipment tracking for orders';
COMMENT ON TABLE shipment_items IS 'Links order items to shipments (supports partial shipments)';
COMMENT ON TABLE shipment_tracking_events IS 'History of tracking status updates';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_shipments_updated_at ON shipments;
DROP TRIGGER IF EXISTS update_shipping_methods_updated_at ON shipping_methods;
DROP TABLE IF EXISTS shipment_tracking_events CASCADE;
DROP TABLE IF EXISTS shipment_items CASCADE;
DROP TABLE IF EXISTS shipments CASCADE;
DROP TABLE IF EXISTS shipping_rates CASCADE;
DROP TABLE IF EXISTS shipping_methods CASCADE;
-- +goose StatementEnd
