-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- CUSTOM DOMAINS
-- ============================================================================
-- This migration adds support for custom domains on tenants.
-- Tenants can serve their storefront on their own domain (e.g., shop.example.com)
-- instead of the default Freyja subdomain (example.freyja.app).
--
-- Architecture:
-- - Caddy On-Demand TLS provisions Let's Encrypt certificates automatically
-- - DNS verification (TXT + CNAME) required before activation
-- - Storefront routes redirect to custom domain when active
-- - Admin/SaaS routes remain accessible on both domains
-- - Subdomain-only initially (no apex domain support)
-- ============================================================================

-- Add custom domain columns to tenants table
ALTER TABLE tenants
    -- The custom domain (e.g., "shop.roastercoffee.com")
    -- UNIQUE constraint prevents two tenants from claiming the same domain
    ADD COLUMN custom_domain VARCHAR(255) UNIQUE,

    -- Domain status state machine:
    --   none      → No custom domain configured
    --   pending   → Domain entered, waiting for DNS records
    --   verifying → DNS verification in progress
    --   verified  → DNS verified, ready to activate
    --   active    → Domain is live, serving traffic
    --   failed    → Verification or DNS health check failed
    ADD COLUMN custom_domain_status VARCHAR(50) NOT NULL DEFAULT 'none'
        CHECK (custom_domain_status IN ('none', 'pending', 'verifying', 'verified', 'active', 'failed')),

    -- SHA-256 hash of verification token (for DNS TXT record)
    -- Raw token is shown in UI, hash is stored for security
    -- Prevents database breach from exposing valid tokens
    ADD COLUMN custom_domain_verification_token VARCHAR(64),

    -- Timestamp when domain was verified
    ADD COLUMN custom_domain_verified_at TIMESTAMPTZ,

    -- Timestamp when domain was activated (went live)
    ADD COLUMN custom_domain_activated_at TIMESTAMPTZ,

    -- Last DNS health check timestamp (background job updates this daily)
    ADD COLUMN custom_domain_last_checked_at TIMESTAMPTZ,

    -- Error message if verification or health check failed
    -- Shown in admin UI to help tenant troubleshoot
    ADD COLUMN custom_domain_error_message TEXT;

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Primary index for domain lookup (used by Caddy validation endpoint)
-- Partial index: Only active domains need to be looked up quickly
-- This is the CRITICAL PATH for tenant resolution via custom domain
CREATE UNIQUE INDEX idx_tenants_custom_domain_active
    ON tenants(custom_domain)
    WHERE custom_domain IS NOT NULL AND custom_domain_status = 'active';

-- Index for verification token lookup (used during DNS verification)
CREATE INDEX idx_tenants_custom_domain_verification
    ON tenants(custom_domain_verification_token)
    WHERE custom_domain_verification_token IS NOT NULL;

-- Index for finding domains that need health checks
CREATE INDEX idx_tenants_custom_domain_health_check
    ON tenants(custom_domain_last_checked_at)
    WHERE custom_domain_status = 'active';

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON COLUMN tenants.custom_domain IS 'Tenant custom domain (e.g., shop.example.com); UNIQUE across all tenants';
COMMENT ON COLUMN tenants.custom_domain_status IS 'Domain status: none, pending, verifying, verified, active, failed';
COMMENT ON COLUMN tenants.custom_domain_verification_token IS 'SHA-256 hash of verification token (for DNS TXT record)';
COMMENT ON COLUMN tenants.custom_domain_verified_at IS 'Timestamp when DNS verification succeeded';
COMMENT ON COLUMN tenants.custom_domain_activated_at IS 'Timestamp when domain went live';
COMMENT ON COLUMN tenants.custom_domain_last_checked_at IS 'Last DNS health check (background job)';
COMMENT ON COLUMN tenants.custom_domain_error_message IS 'Error message if verification/health check failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_tenants_custom_domain_health_check;
DROP INDEX IF EXISTS idx_tenants_custom_domain_verification;
DROP INDEX IF EXISTS idx_tenants_custom_domain_active;

-- Drop columns
ALTER TABLE tenants
    DROP COLUMN IF EXISTS custom_domain_error_message,
    DROP COLUMN IF EXISTS custom_domain_last_checked_at,
    DROP COLUMN IF EXISTS custom_domain_activated_at,
    DROP COLUMN IF EXISTS custom_domain_verified_at,
    DROP COLUMN IF EXISTS custom_domain_verification_token,
    DROP COLUMN IF EXISTS custom_domain_status,
    DROP COLUMN IF EXISTS custom_domain;

-- +goose StatementEnd
