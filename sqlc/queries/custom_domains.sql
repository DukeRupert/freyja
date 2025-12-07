-- ============================================================================
-- CUSTOM DOMAIN QUERIES
-- ============================================================================
-- These queries manage tenant custom domains including:
-- - DNS verification workflow
-- - Domain activation and deactivation
-- - Health monitoring
-- ============================================================================

-- ============================================================================
-- TENANT RESOLUTION
-- ============================================================================

-- name: GetTenantByCustomDomain :one
-- Lookup tenant by custom domain (used in tenant resolution middleware)
-- CRITICAL PATH: This query runs on every request to a custom domain
-- Returns tenant only if domain status is 'active'
SELECT id, slug, name, subdomain, custom_domain, custom_domain_status, status
FROM tenants
WHERE custom_domain = $1
  AND custom_domain_status = 'active'
LIMIT 1;

-- ============================================================================
-- CUSTOM DOMAIN MANAGEMENT
-- ============================================================================

-- name: SetCustomDomain :exec
-- Initiate custom domain setup
-- Sets domain to 'pending' status and stores verification token hash
-- Parameters:
--   $1: tenant_id (UUID)
--   $2: custom_domain (VARCHAR) - e.g., "shop.example.com"
--   $3: verification_token_hash (VARCHAR) - SHA-256 hash of verification token
UPDATE tenants
SET custom_domain = $2,
    custom_domain_status = 'pending',
    custom_domain_verification_token = $3,
    custom_domain_verified_at = NULL,
    custom_domain_activated_at = NULL,
    custom_domain_last_checked_at = NULL,
    custom_domain_error_message = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkDomainVerifying :exec
-- Mark domain as 'verifying' during DNS check
-- Used to prevent concurrent verification attempts
UPDATE tenants
SET custom_domain_status = 'verifying',
    custom_domain_error_message = NULL,
    updated_at = NOW()
WHERE id = $1
  AND custom_domain_status IN ('pending', 'failed');

-- name: MarkDomainVerified :exec
-- Mark domain as 'verified' after successful DNS verification
-- Domain can now be activated by tenant
UPDATE tenants
SET custom_domain_status = 'verified',
    custom_domain_verified_at = NOW(),
    custom_domain_error_message = NULL,
    updated_at = NOW()
WHERE id = $1
  AND custom_domain_status = 'verifying';

-- name: MarkDomainVerificationFailed :exec
-- Mark domain verification as failed with error message
-- Parameters:
--   $1: tenant_id (UUID)
--   $2: error_message (TEXT)
UPDATE tenants
SET custom_domain_status = 'failed',
    custom_domain_error_message = $2,
    updated_at = NOW()
WHERE id = $1
  AND custom_domain_status = 'verifying';

-- name: ActivateCustomDomain :exec
-- Activate a verified custom domain
-- Domain must be in 'verified' status
-- After this, Caddy will provision TLS certificate and serve traffic
UPDATE tenants
SET custom_domain_status = 'active',
    custom_domain_activated_at = NOW(),
    custom_domain_last_checked_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND custom_domain_status = 'verified';

-- name: DeactivateCustomDomain :exec
-- Deactivate a custom domain (set back to 'none')
-- Removes all custom domain data
-- Used when tenant removes their custom domain
UPDATE tenants
SET custom_domain = NULL,
    custom_domain_status = 'none',
    custom_domain_verification_token = NULL,
    custom_domain_verified_at = NULL,
    custom_domain_activated_at = NULL,
    custom_domain_last_checked_at = NULL,
    custom_domain_error_message = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: GetCustomDomainStatus :one
-- Get current custom domain status for a tenant
-- Returns all custom domain fields for display in admin UI
SELECT
    custom_domain,
    custom_domain_status,
    custom_domain_verified_at,
    custom_domain_activated_at,
    custom_domain_last_checked_at,
    custom_domain_error_message
FROM tenants
WHERE id = $1;

-- ============================================================================
-- CADDY VALIDATION
-- ============================================================================

-- name: ValidateDomainForCaddy :one
-- Validate domain for Caddy's on-demand TLS 'ask' endpoint
-- Returns true if domain is valid and active, false otherwise
-- Caddy calls this before issuing a Let's Encrypt certificate
SELECT EXISTS(
    SELECT 1 FROM tenants
    WHERE custom_domain = $1
      AND custom_domain_status = 'active'
) as is_valid;

-- ============================================================================
-- BACKGROUND JOBS - HEALTH MONITORING
-- ============================================================================

-- name: GetActiveCustomDomains :many
-- Get all tenants with active custom domains for health monitoring
-- Used by daily background job to verify CNAME records are still valid
SELECT
    id,
    custom_domain,
    custom_domain_last_checked_at
FROM tenants
WHERE custom_domain_status = 'active'
  AND custom_domain IS NOT NULL
ORDER BY custom_domain_last_checked_at ASC NULLS FIRST;

-- name: UpdateCustomDomainHealthCheck :exec
-- Update last_checked_at timestamp after health check
-- Parameters:
--   $1: tenant_id (UUID)
--   $2: is_healthy (BOOLEAN) - true if CNAME still valid
--   $3: error_message (TEXT) - NULL if healthy, error message if unhealthy
UPDATE tenants
SET custom_domain_last_checked_at = NOW(),
    custom_domain_status = CASE
        WHEN $2 = true THEN 'active'
        ELSE 'failed'
    END,
    custom_domain_error_message = CASE
        WHEN $2 = true THEN NULL
        ELSE $3
    END,
    updated_at = NOW()
WHERE id = $1
  AND custom_domain_status = 'active';

-- ============================================================================
-- BACKGROUND JOBS - CLEANUP
-- ============================================================================

-- name: CleanupStalePendingDomains :exec
-- Remove domains stuck in 'pending' status for > 30 days
-- Used by weekly cleanup job to free up abandoned domain claims
DELETE FROM tenants
WHERE custom_domain_status = 'pending'
  AND updated_at < NOW() - INTERVAL '30 days';

-- ============================================================================
-- ADMIN QUERIES
-- ============================================================================

-- name: GetCustomDomainsByStatus :many
-- Get all custom domains filtered by status
-- Used for admin reporting and monitoring
SELECT
    t.id,
    t.slug,
    t.name,
    t.custom_domain,
    t.custom_domain_status,
    t.custom_domain_verified_at,
    t.custom_domain_activated_at,
    t.custom_domain_last_checked_at,
    t.custom_domain_error_message
FROM tenants t
WHERE t.custom_domain_status = $1
  AND t.custom_domain IS NOT NULL
ORDER BY t.updated_at DESC;

-- name: GetCustomDomainCount :one
-- Get count of custom domains by status
-- Used for admin dashboard metrics
SELECT
    COUNT(*) FILTER (WHERE custom_domain_status = 'active') as active_count,
    COUNT(*) FILTER (WHERE custom_domain_status = 'verified') as verified_count,
    COUNT(*) FILTER (WHERE custom_domain_status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE custom_domain_status = 'failed') as failed_count
FROM tenants
WHERE custom_domain IS NOT NULL;
