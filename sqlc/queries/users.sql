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
SELECT *
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

-- Admin queries

-- name: CountUsers :one
-- Count total users for pagination
SELECT COUNT(*)
FROM users
WHERE tenant_id = $1
  AND status != 'closed';

-- name: ListUsersByAccountType :many
-- List users filtered by account type
SELECT *
FROM users
WHERE tenant_id = $1
  AND account_type = $2
  AND status != 'closed'
ORDER BY created_at DESC;

-- name: ListWholesaleApplications :many
-- List pending wholesale applications
SELECT
    id,
    email,
    first_name,
    last_name,
    company_name,
    wholesale_application_status,
    wholesale_application_notes,
    created_at
FROM users
WHERE tenant_id = $1
  AND wholesale_application_status = 'pending'
ORDER BY created_at ASC;

-- name: GetUserStats :one
-- Get user statistics for dashboard
SELECT
    COUNT(*) as total_users,
    COUNT(*) FILTER (WHERE account_type = 'retail') as retail_users,
    COUNT(*) FILTER (WHERE account_type = 'wholesale') as wholesale_users,
    COUNT(*) FILTER (WHERE wholesale_application_status = 'pending') as pending_applications
FROM users
WHERE tenant_id = $1
  AND status != 'closed';

-- =============================================================================
-- WHOLESALE CUSTOMER QUERIES
-- =============================================================================

-- name: GetWholesaleCustomer :one
-- Get wholesale customer with payment terms details
SELECT
    u.*,
    pt.name as payment_terms_name,
    pt.code as payment_terms_code,
    pt.days as payment_terms_days,
    pl.name as price_list_name
FROM users u
LEFT JOIN payment_terms pt ON pt.id = u.payment_terms_id
LEFT JOIN user_price_lists upl ON upl.user_id = u.id
LEFT JOIN price_lists pl ON pl.id = upl.price_list_id
WHERE u.id = $1
  AND u.account_type = 'wholesale'
  AND u.status != 'closed'
LIMIT 1;

-- name: ListWholesaleCustomers :many
-- List wholesale customers with payment terms and billing info
SELECT
    u.id,
    u.tenant_id,
    u.email,
    u.first_name,
    u.last_name,
    u.company_name,
    u.phone,
    u.status,
    u.billing_cycle,
    u.minimum_spend_cents,
    u.customer_reference,
    u.created_at,
    pt.name as payment_terms_name,
    pt.days as payment_terms_days,
    pl.name as price_list_name
FROM users u
LEFT JOIN payment_terms pt ON pt.id = u.payment_terms_id
LEFT JOIN user_price_lists upl ON upl.user_id = u.id
LEFT JOIN price_lists pl ON pl.id = upl.price_list_id
WHERE u.tenant_id = $1
  AND u.account_type = 'wholesale'
  AND u.status != 'closed'
ORDER BY u.company_name ASC, u.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWholesaleCustomer :exec
-- Update wholesale customer settings
UPDATE users
SET
    company_name = COALESCE($2, company_name),
    payment_terms_id = $3,
    billing_cycle = $4,
    billing_cycle_day = $5,
    minimum_spend_cents = $6,
    customer_reference = $7,
    internal_note = $8,
    email_orders = $9,
    email_dispatches = $10,
    email_invoices = $11,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateWholesaleApplicationWithTerms :exec
-- Approve wholesale application with payment terms assignment
UPDATE users
SET
    wholesale_application_status = $2,
    wholesale_application_notes = $3,
    wholesale_approved_at = CASE WHEN $2 = 'approved' THEN NOW() ELSE NULL END,
    wholesale_approved_by = CASE WHEN $2 = 'approved' THEN $4 ELSE NULL END,
    account_type = CASE WHEN $2 = 'approved' THEN 'wholesale' ELSE account_type END,
    payment_terms_id = CASE WHEN $2 = 'approved' THEN $5 ELSE payment_terms_id END,
    billing_cycle = CASE WHEN $2 = 'approved' THEN $6 ELSE billing_cycle END
WHERE id = $1;

-- name: GetCustomersForBillingCycle :many
-- Get wholesale customers due for consolidated invoice generation
-- Used by billing cycle job to find accounts ready for invoicing
SELECT
    u.id,
    u.tenant_id,
    u.email,
    u.company_name,
    u.billing_cycle,
    u.billing_cycle_day,
    u.payment_terms_id,
    pt.days as payment_terms_days
FROM users u
LEFT JOIN payment_terms pt ON pt.id = u.payment_terms_id
WHERE u.tenant_id = $1
  AND u.account_type = 'wholesale'
  AND u.status = 'active'
  AND u.billing_cycle = $2
  AND u.billing_cycle IS NOT NULL;

-- name: GetUserNotificationEmails :one
-- Get notification email addresses for a user (with fallback to primary email)
SELECT
    id,
    email,
    COALESCE(NULLIF(email_orders, ''), email) as email_for_orders,
    COALESCE(NULLIF(email_dispatches, ''), email) as email_for_dispatches,
    COALESCE(NULLIF(email_invoices, ''), email) as email_for_invoices
FROM users
WHERE id = $1;
