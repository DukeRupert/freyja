-- name: CreateUser :one
-- Create a new user (retail account by default)
INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    first_name,
    last_name,
    account_type,
    status
) VALUES (
    $1, $2, $3, $4, $5, 'retail', 'active'
) RETURNING *;

-- name: GetUserByEmail :one
-- Get user by email within a tenant
SELECT *
FROM users
WHERE tenant_id = $1
  AND email = $2
  AND status != 'closed'
LIMIT 1;

-- name: GetUserByID :one
-- Get user by ID
SELECT *
FROM users
WHERE id = $1
  AND status != 'closed'
LIMIT 1;

-- name: UpdateUserProfile :exec
-- Update user profile information
UPDATE users
SET
    first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    phone = COALESCE($4, phone)
WHERE id = $1;

-- name: UpdateUserPassword :exec
-- Update user password
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: VerifyUserEmail :exec
-- Mark user email as verified
UPDATE users
SET email_verified = TRUE
WHERE id = $1;

-- name: ListUsers :many
-- List all users for a tenant (admin only)
SELECT
    id,
    email,
    email_verified,
    account_type,
    first_name,
    last_name,
    company_name,
    status,
    wholesale_application_status,
    created_at
FROM users
WHERE tenant_id = $1
  AND status != 'closed'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWholesaleApplication :exec
-- Update wholesale application status
UPDATE users
SET
    wholesale_application_status = $2,
    wholesale_application_notes = $3,
    wholesale_approved_at = CASE WHEN $2 = 'approved' THEN NOW() ELSE NULL END,
    wholesale_approved_by = CASE WHEN $2 = 'approved' THEN $4 ELSE NULL END,
    account_type = CASE WHEN $2 = 'approved' THEN 'wholesale' ELSE account_type END,
    payment_terms = CASE WHEN $2 = 'approved' THEN COALESCE($5, 'net_30') ELSE payment_terms END
WHERE id = $1;

-- name: UpdateUserStatus :exec
-- Update user status (active, suspended, closed)
UPDATE users
SET status = $2
WHERE id = $1;
