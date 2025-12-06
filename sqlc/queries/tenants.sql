-- Tenants: Coffee roasters using the platform (multi-tenant root)

-- name: CreateTenant :one
-- Create a new tenant (called after Stripe checkout)
INSERT INTO tenants (
    name,
    slug,
    email,
    status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetTenantByID :one
-- Get tenant by ID
SELECT *
FROM tenants
WHERE id = $1
LIMIT 1;

-- name: GetTenantBySlug :one
-- Get tenant by slug (for subdomain/path routing)
SELECT *
FROM tenants
WHERE slug = $1
LIMIT 1;

-- name: TenantSlugExists :one
-- Check if a slug is already taken
SELECT EXISTS(
    SELECT 1
    FROM tenants
    WHERE slug = $1
) as exists;

-- name: GetTenantByStripeCustomerID :one
-- Get tenant by Stripe customer ID (for webhook processing)
SELECT *
FROM tenants
WHERE stripe_customer_id = $1
LIMIT 1;

-- name: GetTenantByStripeSubscriptionID :one
-- Get tenant by Stripe subscription ID (for webhook processing)
SELECT *
FROM tenants
WHERE stripe_subscription_id = $1
LIMIT 1;

-- name: UpdateTenantStripeCustomer :exec
-- Set Stripe customer ID for a tenant
UPDATE tenants
SET
    stripe_customer_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateTenantStripeSubscription :exec
-- Set Stripe subscription ID for a tenant
UPDATE tenants
SET
    stripe_subscription_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: SetTenantStatus :exec
-- Update tenant status
UPDATE tenants
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ActivateTenant :exec
-- Activate a pending tenant after password setup
UPDATE tenants
SET
    status = 'active',
    updated_at = NOW()
WHERE id = $1
  AND status = 'pending';

-- name: StartTenantGracePeriod :exec
-- Start grace period after payment failure
UPDATE tenants
SET
    status = 'past_due',
    grace_period_started_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: ClearTenantGracePeriod :exec
-- Clear grace period after successful payment
UPDATE tenants
SET
    status = 'active',
    grace_period_started_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SuspendTenant :exec
-- Suspend a tenant (grace period expired or manual suspension)
UPDATE tenants
SET
    status = 'suspended',
    updated_at = NOW()
WHERE id = $1;

-- name: CancelTenant :exec
-- Cancel a tenant subscription
UPDATE tenants
SET
    status = 'cancelled',
    updated_at = NOW()
WHERE id = $1;

-- name: GetTenantsWithExpiredGracePeriod :many
-- Get tenants whose grace period has expired (for suspension job)
-- Grace period is 7 days (168 hours)
SELECT *
FROM tenants
WHERE status = 'past_due'
  AND grace_period_started_at IS NOT NULL
  AND grace_period_started_at <= NOW() - INTERVAL '168 hours'
ORDER BY grace_period_started_at ASC;

-- name: ListActiveTenants :many
-- List all active tenants (for admin/reporting)
SELECT *
FROM tenants
WHERE status = 'active'
ORDER BY created_at DESC;

-- name: CountTenantsByStatus :one
-- Count tenants by status
SELECT COUNT(*)
FROM tenants
WHERE status = $1;

-- name: UpdateTenantProfile :one
-- Update tenant profile information
UPDATE tenants
SET
    name = COALESCE(sqlc.narg('name'), name),
    email = COALESCE(sqlc.narg('email'), email),
    phone = COALESCE(sqlc.narg('phone'), phone),
    website = COALESCE(sqlc.narg('website'), website),
    business_name = COALESCE(sqlc.narg('business_name'), business_name),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
