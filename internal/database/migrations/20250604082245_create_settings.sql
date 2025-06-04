-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
);

-- Index for fast key lookups
CREATE UNIQUE INDEX idx_settings_key ON settings (key);

-- Index for category-based queries
CREATE INDEX idx_settings_category ON settings (category);

-- Insert default settings
INSERT INTO
    settings (key, value, description, category)
VALUES
    (
        'tax_rate',
        '0.08',
        'Default tax rate for orders',
        'pricing'
    ),
    (
        'free_shipping_threshold',
        '5000',
        'Free shipping threshold in cents ($50)',
        'shipping'
    ),
    (
        'currency',
        '"USD"',
        'Default currency code',
        'general'
    ),
    (
        'site_name',
        '"Coffee Roasters"',
        'Site name for emails and branding',
        'general'
    ),
    (
        'support_email',
        '"support@example.com"',
        'Customer support email',
        'general'
    );

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_settings_category;

DROP INDEX IF EXISTS idx_settings_key;

DROP TABLE IF EXISTS settings;

-- +goose StatementEnd
