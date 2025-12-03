-- Payment Terms Queries
-- Manages reusable payment terms for wholesale invoicing

-- name: CreatePaymentTerms :one
-- Create a new payment terms record
INSERT INTO payment_terms (
    tenant_id,
    name,
    code,
    days,
    is_default,
    is_active,
    sort_order,
    description
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPaymentTermsByID :one
-- Get payment terms by ID
SELECT * FROM payment_terms
WHERE id = $1
  AND tenant_id = $2
LIMIT 1;

-- name: GetPaymentTermsByCode :one
-- Get payment terms by code within a tenant
SELECT * FROM payment_terms
WHERE tenant_id = $1
  AND code = $2
  AND is_active = TRUE
LIMIT 1;

-- name: GetDefaultPaymentTerms :one
-- Get the default payment terms for a tenant
SELECT * FROM payment_terms
WHERE tenant_id = $1
  AND is_default = TRUE
  AND is_active = TRUE
LIMIT 1;

-- name: ListPaymentTerms :many
-- List all payment terms for a tenant
SELECT * FROM payment_terms
WHERE tenant_id = $1
  AND is_active = TRUE
ORDER BY sort_order ASC, name ASC;

-- name: ListAllPaymentTerms :many
-- List all payment terms including inactive (for admin)
SELECT * FROM payment_terms
WHERE tenant_id = $1
ORDER BY sort_order ASC, name ASC;

-- name: UpdatePaymentTerms :exec
-- Update payment terms
UPDATE payment_terms
SET
    name = $3,
    code = $4,
    days = $5,
    is_default = $6,
    is_active = $7,
    sort_order = $8,
    description = $9,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: SetDefaultPaymentTerms :exec
-- Set payment terms as default (clears previous default)
UPDATE payment_terms
SET is_default = (id = $2)
WHERE tenant_id = $1;

-- name: DeletePaymentTerms :exec
-- Soft delete by deactivating (preserves referential integrity)
UPDATE payment_terms
SET is_active = FALSE,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: CountUsersWithPaymentTerms :one
-- Count users using specific payment terms (for safe deletion check)
SELECT COUNT(*)
FROM users
WHERE payment_terms_id = $1;
