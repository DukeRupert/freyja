-- name: GetTaxRateByState :one
-- Get the active tax rate for a specific state within a tenant
SELECT * FROM tax_rates
WHERE tenant_id = $1
  AND state = $2
  AND is_active = TRUE
LIMIT 1;

-- name: ListTaxRates :many
-- List all tax rates for a tenant (admin view)
SELECT * FROM tax_rates
WHERE tenant_id = $1
ORDER BY state ASC;

-- name: CreateTaxRate :one
-- Create a new tax rate
INSERT INTO tax_rates (
    tenant_id,
    state,
    rate,
    tax_shipping,
    name,
    is_active
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTaxRate :one
-- Update an existing tax rate
UPDATE tax_rates
SET
    rate = $3,
    tax_shipping = $4,
    name = $5,
    is_active = $6,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: DeleteTaxRate :exec
-- Delete a tax rate
DELETE FROM tax_rates
WHERE tenant_id = $1
  AND id = $2;
