-- +goose Up
-- +goose StatementBegin

-- Add 'warehouse' to the address_type constraint
ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_address_type_check;
ALTER TABLE addresses ADD CONSTRAINT addresses_address_type_check
    CHECK (address_type IN ('shipping', 'billing', 'both', 'warehouse'));

-- Seed a warehouse address for the master tenant
-- This is used as the shipping origin for rate calculations
INSERT INTO addresses (
    tenant_id,
    full_name,
    company,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    country,
    address_type,
    is_validated
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Serenity Roasters',
    'Firefly Software',
    '123 Coffee Lane',
    'Suite 100',
    'Portland',
    'OR',
    '97201',
    'US',
    'warehouse',
    TRUE
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove the warehouse address
DELETE FROM addresses
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND address_type = 'warehouse';

-- Restore original constraint
ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_address_type_check;
ALTER TABLE addresses ADD CONSTRAINT addresses_address_type_check
    CHECK (address_type IN ('shipping', 'billing', 'both'));

-- +goose StatementEnd
