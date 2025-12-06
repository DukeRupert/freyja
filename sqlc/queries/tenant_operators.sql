-- Tenant Operators: People who manage a tenant (roaster staff who pay for Freyja)
-- Separate from users table (storefront customers)

-- name: CreateTenantOperator :one
-- Create a new tenant operator (called after Stripe checkout)
INSERT INTO tenant_operators (
    tenant_id,
    email,
    name,
    role,
    setup_token_hash,
    setup_token_expires_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, 'pending'
) RETURNING *;

-- name: GetTenantOperatorByID :one
-- Get operator by ID
SELECT *
FROM tenant_operators
WHERE id = $1
LIMIT 1;

-- name: GetTenantOperatorByEmail :one
-- Get operator by email (global lookup for login)
SELECT *
FROM tenant_operators
WHERE email = $1
LIMIT 1;

-- name: GetTenantOperatorByEmailAndTenant :one
-- Get operator by email within a specific tenant
SELECT *
FROM tenant_operators
WHERE tenant_id = $1
  AND email = $2
LIMIT 1;

-- name: GetTenantOperatorByIDAndTenant :one
-- Get operator by ID within a specific tenant (for session validation)
SELECT *
FROM tenant_operators
WHERE id = $1
  AND tenant_id = $2
LIMIT 1;

-- name: GetTenantOperatorBySetupToken :one
-- Get operator by valid (non-expired) setup token
SELECT *
FROM tenant_operators
WHERE setup_token_hash = $1
  AND setup_token_expires_at > NOW()
  AND status = 'pending'
LIMIT 1;

-- name: GetTenantOperatorByResetToken :one
-- Get operator by valid (non-expired) reset token
SELECT *
FROM tenant_operators
WHERE reset_token_hash = $1
  AND reset_token_expires_at > NOW()
LIMIT 1;

-- name: SetOperatorPassword :exec
-- Set operator password and activate account (called during setup)
UPDATE tenant_operators
SET
    password_hash = $2,
    status = 'active',
    setup_token_hash = NULL,
    setup_token_expires_at = NULL,
    reset_token_hash = NULL,
    reset_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorPassword :exec
-- Update operator password (for password resets)
UPDATE tenant_operators
SET
    password_hash = $2,
    reset_token_hash = NULL,
    reset_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SetOperatorSetupToken :exec
-- Set or refresh setup token for an operator
UPDATE tenant_operators
SET
    setup_token_hash = $2,
    setup_token_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: SetOperatorResetToken :exec
-- Set password reset token for an operator
UPDATE tenant_operators
SET
    reset_token_hash = $2,
    reset_token_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ClearOperatorSetupToken :exec
-- Clear setup token after successful use
UPDATE tenant_operators
SET
    setup_token_hash = NULL,
    setup_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorLastLogin :exec
-- Update last login timestamp
UPDATE tenant_operators
SET
    last_login_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorProfile :one
-- Update operator profile information
UPDATE tenant_operators
SET
    name = COALESCE(sqlc.narg('name'), name),
    email = COALESCE(sqlc.narg('email'), email),
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2
RETURNING *;

-- name: SuspendOperator :exec
-- Suspend an operator account
UPDATE tenant_operators
SET
    status = 'suspended',
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2;

-- name: ActivateOperator :exec
-- Activate a suspended operator account
UPDATE tenant_operators
SET
    status = 'active',
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2;

-- name: ListTenantOperators :many
-- List all operators for a tenant (for future multi-user support)
SELECT *
FROM tenant_operators
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: CountOperatorsByTenant :one
-- Count operators for a tenant
SELECT COUNT(*)
FROM tenant_operators
WHERE tenant_id = $1;

-- name: DeleteTenantOperator :exec
-- Delete an operator (for cleanup/testing)
DELETE FROM tenant_operators
WHERE id = $1
  AND tenant_id = $2;
