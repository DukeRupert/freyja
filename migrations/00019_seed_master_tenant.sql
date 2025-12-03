-- +goose Up
-- +goose StatementBegin

-- Seed the master tenant (Firefly Software - platform owner)
-- This tenant uses the well-known UUID that matches the default TENANT_ID in config
-- "Serenity" is the ship from Firefly - fitting for the master/founding tenant
INSERT INTO tenants (
    id,
    name,
    slug,
    email,
    business_name,
    status,
    settings
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Serenity',
    'serenity',
    'admin@fireflysoftware.dev',
    'Firefly Software',
    'active',
    '{"is_master": true}'::jsonb
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001';
-- +goose StatementEnd
