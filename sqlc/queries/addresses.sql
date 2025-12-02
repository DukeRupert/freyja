-- name: ListAddressesForUser :many
-- Get all addresses for a user with their associations
SELECT
    a.id,
    a.tenant_id,
    a.full_name,
    a.company,
    a.address_line1,
    a.address_line2,
    a.city,
    a.state,
    a.postal_code,
    a.country,
    a.phone,
    a.email,
    a.address_type,
    a.is_validated,
    a.created_at,
    a.updated_at,
    ca.is_default_shipping,
    ca.is_default_billing,
    ca.label
FROM addresses a
INNER JOIN customer_addresses ca ON ca.address_id = a.id
WHERE ca.tenant_id = $1
  AND ca.user_id = $2
ORDER BY ca.is_default_shipping DESC, a.created_at DESC;

-- name: GetAddressByIDForUser :one
-- Get a single address by ID (validates user ownership via customer_addresses)
SELECT
    a.id,
    a.tenant_id,
    a.full_name,
    a.company,
    a.address_line1,
    a.address_line2,
    a.city,
    a.state,
    a.postal_code,
    a.country,
    a.phone,
    a.email,
    a.address_type,
    a.is_validated,
    a.created_at,
    a.updated_at
FROM addresses a
INNER JOIN customer_addresses ca ON ca.address_id = a.id
WHERE a.id = $1
  AND ca.tenant_id = $2
  AND ca.user_id = $3
LIMIT 1;

-- name: GetAddressByID :one
-- Get a single address by ID (no user validation - for system use)
SELECT * FROM addresses
WHERE id = $1
LIMIT 1;

-- name: GetDefaultShippingAddress :one
-- Get the default shipping address for a user
SELECT
    a.id,
    a.tenant_id,
    a.full_name,
    a.company,
    a.address_line1,
    a.address_line2,
    a.city,
    a.state,
    a.postal_code,
    a.country,
    a.phone,
    a.email,
    a.address_type,
    a.is_validated,
    a.created_at,
    a.updated_at
FROM addresses a
INNER JOIN customer_addresses ca ON ca.address_id = a.id
WHERE ca.tenant_id = $1
  AND ca.user_id = $2
  AND ca.is_default_shipping = TRUE
LIMIT 1;

-- name: CreateAddress :one
-- Create a new address
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
    phone,
    email,
    address_type
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: CreateCustomerAddress :one
-- Link an address to a user
INSERT INTO customer_addresses (
    tenant_id,
    user_id,
    address_id,
    is_default_shipping,
    is_default_billing,
    label
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateAddress :one
-- Update an address
UPDATE addresses
SET
    full_name = $3,
    company = $4,
    address_line1 = $5,
    address_line2 = $6,
    city = $7,
    state = $8,
    postal_code = $9,
    country = $10,
    phone = $11,
    email = $12,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: SetDefaultShippingAddress :exec
-- Set an address as the default shipping address for a user
UPDATE customer_addresses
SET is_default_shipping = (address_id = $3)
WHERE tenant_id = $1
  AND user_id = $2;

-- name: DeleteCustomerAddress :exec
-- Remove association between user and address
DELETE FROM customer_addresses
WHERE tenant_id = $1
  AND user_id = $2
  AND address_id = $3;
