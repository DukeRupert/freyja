# Custom Domains for Freyja

## Overview

Custom domains allow tenants to serve their storefront on their own domain (e.g., `shop.roastercoffee.com`) instead of the default Freyja subdomain (`roastercoffee.freyja.app`). This document outlines the complete architecture for custom domain support.

**Key Design Decisions:**
- **Caddy On-Demand TLS** - Automatic Let's Encrypt certificate provisioning
- **DNS verification required** - Tenants must prove domain ownership via TXT record before activation
- **Subdomain redirect** - After custom domain is enabled, storefront routes redirect from `*.freyja.app` to custom domain
- **Admin routes accessible on both domains** - `/admin/*` and `/saas/*` remain accessible on both domains for reliability
- **SSL-only** - All custom domains enforce HTTPS
- **No apex domain support initially** - Require subdomain (e.g., `www.example.com` or `shop.example.com`) with CNAME pointing to `custom.freyja.app`

---

## Architecture

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Tenant initiates custom domain setup                         │
│    → Enters domain: shop.roastercoffee.com                      │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│ 2. System generates verification token                          │
│    → TXT record: _freyja-verify.shop.roastercoffee.com         │
│    → Value: freyja-verify=a7b2c3d4e5f6g7h8...                   │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│ 3. Tenant adds DNS records                                      │
│    → CNAME: shop.roastercoffee.com → custom.freyja.app         │
│    → TXT: _freyja-verify.shop.roastercoffee.com → token        │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│ 4. Tenant clicks "Verify Domain"                                │
│    → System performs DNS TXT lookup                             │
│    → If token matches, domain marked "verified"                 │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│ 5. Tenant clicks "Activate Domain"                              │
│    → Domain status: verified → active                           │
│    → Caddy on-demand TLS provisions certificate                 │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│ 6. Storefront accessible on custom domain                       │
│    → shop.roastercoffee.com serves storefront                   │
│    → roastercoffee.freyja.app redirects to custom domain        │
│    → /admin/* still accessible on both domains                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Database Schema

### Columns Added to `tenants` Table

```sql
-- Custom domain fields
custom_domain VARCHAR(255) UNIQUE,              -- e.g., "shop.roastercoffee.com"
custom_domain_status VARCHAR(50) DEFAULT 'none'
    CHECK (custom_domain_status IN ('none', 'pending', 'verifying', 'verified', 'active', 'failed')),
custom_domain_verification_token VARCHAR(64),   -- SHA-256 hash of verification token
custom_domain_verified_at TIMESTAMPTZ,
custom_domain_activated_at TIMESTAMPTZ,
custom_domain_last_checked_at TIMESTAMPTZ,      -- Last DNS verification attempt
custom_domain_error_message TEXT,               -- Error message if verification failed

-- Index for custom domain lookup (critical path - used by Caddy validation)
CREATE UNIQUE INDEX idx_tenants_custom_domain ON tenants(custom_domain)
    WHERE custom_domain IS NOT NULL AND custom_domain_status = 'active';

-- Index for verification token lookup
CREATE INDEX idx_tenants_custom_domain_verification
    ON tenants(custom_domain_verification_token)
    WHERE custom_domain_verification_token IS NOT NULL;
```

### Domain Status State Machine

```
none → pending → verifying → verified → active
         ↓          ↓           ↓
       failed ← failed ← failed

From active:
  → none (domain removed)
  → failed (DNS check fails, e.g., CNAME removed)
```

**Status Descriptions:**
- `none` - No custom domain configured
- `pending` - Tenant has entered domain, waiting for DNS records to be added
- `verifying` - System is checking DNS records (during verification attempt)
- `verified` - DNS verified, domain ready to activate
- `active` - Domain is live, serving traffic, TLS certificate provisioned
- `failed` - Verification or ongoing DNS checks failed

### Why Not a Separate Table?

Decision: Store custom domain data directly on `tenants` table rather than a separate `custom_domains` table.

**Rationale:**
- One-to-one relationship: Each tenant has at most one custom domain
- Simpler queries: No joins required for tenant resolution middleware
- Fewer round-trips: Domain lookup happens on every request
- Atomic updates: Domain status changes with tenant status changes
- Performance: UNIQUE index on `custom_domain` provides fast lookups

**Future Consideration:** If we later support multiple custom domains per tenant (e.g., `shop.example.com` + `cafe.example.com`), migrate to a separate table. Current design optimizes for 99% use case (one domain).

---

## DNS Verification

### Verification Token Generation

```go
// Generate a cryptographically secure 32-byte verification token
func generateVerificationToken() (string, string) {
    rawToken := generateSecureRandomBytes(32) // 64-char hex string
    tokenHash := sha256Hash(rawToken)         // Store hash in DB
    return rawToken, tokenHash
}
```

**Storage:**
- Raw token: Shown to tenant in UI (used to create TXT record)
- Token hash: Stored in `custom_domain_verification_token` (prevents DB compromise from exposing valid tokens)

### DNS Records Required

**1. CNAME Record (required for traffic routing):**
```
Type: CNAME
Name: shop.roastercoffee.com
Value: custom.freyja.app
TTL: 3600 (1 hour recommended)
```

**2. TXT Record (required for verification):**
```
Type: TXT
Name: _freyja-verify.shop.roastercoffee.com
Value: freyja-verify=<raw-token>
TTL: 3600
```

**Why prefix with `_freyja-verify`?**
- Convention: Underscore prefix indicates service-specific record (similar to `_dmarc`, `_domainkey`)
- Avoids conflicts: Won't collide with other TXT records on the domain
- Clear purpose: Easy to identify in DNS panel

### DNS Verification Logic

```go
// Pseudocode for verification process
func VerifyDomain(tenantID uuid.UUID) error {
    tenant := GetTenant(tenantID)
    domain := tenant.CustomDomain
    expectedToken := GetRawVerificationToken(tenant.CustomDomainVerificationToken)

    // Step 1: Verify CNAME points to custom.freyja.app
    cnameTarget := lookupCNAME(domain)
    if !cnameTarget.EndsWith("custom.freyja.app") {
        return ErrCNAMENotConfigured
    }

    // Step 2: Verify TXT record contains verification token
    txtRecords := lookupTXT("_freyja-verify." + domain)
    found := false
    for _, record := range txtRecords {
        if record == "freyja-verify=" + expectedToken {
            found = true
            break
        }
    }
    if !found {
        return ErrVerificationTokenNotFound
    }

    // Step 3: Mark domain as verified
    UpdateTenantDomain(tenantID, DomainStatusVerified)
    return nil
}
```

**DNS Lookup Package:** Use `net.LookupCNAME` and `net.LookupTXT` from Go standard library.

**Timeout:** Set 10-second timeout on DNS lookups to prevent hanging.

**Retries:** Allow tenant to retry verification immediately (no rate limiting on verification checks).

---

## Caddy On-Demand TLS Configuration

### Caddy Configuration

```caddyfile
# Caddyfile

# Default subdomain handling (*.freyja.app)
*.freyja.app {
    reverse_proxy freyja-app:3000
}

# Custom domain handling with on-demand TLS
https:// {
    # Ask Freyja if this domain is valid before issuing certificate
    tls {
        on_demand
    }

    reverse_proxy freyja-app:3000
}

# On-demand TLS validation endpoint
# Caddy will call this before issuing a certificate
# Must return 200 if domain is valid, 404 otherwise
(on_demand_tls_ask) {
    ask http://freyja-app:3000/api/validate-domain
}
```

### Caddy On-Demand TLS Flow

```
┌──────────────────────────────────────────────────────────────┐
│ Browser: https://shop.roastercoffee.com                      │
└────────────────────┬─────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Caddy: Do I have a certificate for shop.roastercoffee.com?  │
│        → No, trigger on-demand TLS                           │
└────────────────────┬─────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Caddy → GET /api/validate-domain?domain=shop.roastercoffee  │
│         .com                                                 │
└────────────────────┬─────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Freyja Validation Endpoint:                                  │
│   1. Lookup tenant by custom_domain                          │
│   2. Check custom_domain_status = 'active'                   │
│   3. Return 200 if valid, 404 if not                         │
└────────────────────┬─────────────────────────────────────────┘
                     │
                 200 OK
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Caddy: Request Let's Encrypt certificate via ACME            │
│        → Performs HTTP-01 or TLS-ALPN-01 challenge           │
└────────────────────┬─────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Let's Encrypt: Issues certificate (valid 90 days)            │
└────────────────────┬─────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────┐
│ Caddy: Caches certificate, serves HTTPS request              │
└──────────────────────────────────────────────────────────────┘
```

**Certificate Storage:** Caddy stores certificates in its internal storage (default: `/data/caddy`). Auto-renews before expiration.

**Security:** The `ask` endpoint prevents certificate issuance for unauthorized domains. This is critical to prevent abuse (e.g., someone trying to issue a cert for `google.com.freyja.app`).

---

## Tenant Resolution Middleware Changes

### Current Middleware Behavior

The existing tenant resolution middleware (likely in `/internal/middleware/tenant.go` or similar) currently:
1. Extracts subdomain from `Host` header (e.g., `roastercoffee.freyja.app` → `roastercoffee`)
2. Looks up tenant by `slug` column
3. Adds tenant to request context

### Required Changes

**New logic:**

```go
func ResolveTenant(r *http.Request) (*Tenant, error) {
    host := r.Host

    // Case 1: Request is on a custom domain (no .freyja.app suffix)
    if !strings.HasSuffix(host, ".freyja.app") {
        // Lookup tenant by custom_domain
        tenant, err := queries.GetTenantByCustomDomain(ctx, host)
        if err != nil {
            return nil, ErrTenantNotFound
        }

        // Verify domain is active
        if tenant.CustomDomainStatus != "active" {
            return nil, ErrCustomDomainNotActive
        }

        return tenant, nil
    }

    // Case 2: Request is on default subdomain (*.freyja.app)
    subdomain := extractSubdomain(host) // "roastercoffee" from "roastercoffee.freyja.app"
    tenant, err := queries.GetTenantBySlug(ctx, subdomain)
    if err != nil {
        return nil, ErrTenantNotFound
    }

    return tenant, nil
}
```

**Performance Consideration:**
- Custom domain lookup uses `idx_tenants_custom_domain` index (UNIQUE, WHERE custom_domain_status = 'active')
- Subdomain lookup uses existing `idx_tenants_slug` index
- Both are indexed lookups, no performance degradation

---

## Subdomain Redirect Logic

### Requirements

When a tenant has an active custom domain:
- **Storefront routes** (`/`, `/products/*`, `/cart`, `/checkout`, etc.) → Redirect to custom domain
- **Admin routes** (`/admin/*`, `/saas/*`) → Accessible on both domains (no redirect)
- **API/webhook routes** (`/api/*`, `/webhooks/*`) → Accessible on both domains (no redirect)

### Implementation Strategy

**Option A: Middleware-based redirect** (Recommended)

```go
func CustomDomainRedirect(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        tenant := GetTenantFromContext(ctx)

        // Only redirect if:
        // 1. Tenant has active custom domain
        // 2. Request is on default subdomain
        // 3. Request is NOT to admin/saas/api/webhooks
        if tenant.CustomDomainStatus == "active" &&
           strings.HasSuffix(r.Host, ".freyja.app") &&
           !isProtectedRoute(r.URL.Path) {

            // Build redirect URL
            redirectURL := "https://" + tenant.CustomDomain + r.URL.Path
            if r.URL.RawQuery != "" {
                redirectURL += "?" + r.URL.RawQuery
            }

            http.Redirect(w, r, redirectURL, http.StatusMovedPermanently) // 301
            return
        }

        next.ServeHTTP(w, r)
    })
}

func isProtectedRoute(path string) bool {
    prefixes := []string{"/admin/", "/saas/", "/api/", "/webhooks/"}
    for _, prefix := range prefixes {
        if strings.HasPrefix(path, prefix) {
            return true
        }
    }
    return false
}
```

**Why 301 Permanent Redirect?**
- Search engines will index the custom domain (not the .freyja.app subdomain)
- Browser caching improves performance for repeat visitors
- Signals to users that the custom domain is the canonical URL

**Application in Router:**

```go
// main.go or wherever router is configured
r.Use(
    middleware.ResolveTenant(queries),
    middleware.CustomDomainRedirect, // Add after tenant resolution
    // ... other middleware
)
```

---

## Admin UI Flow

### Page: `/admin/settings/domain`

**UI States:**

**State 1: No Custom Domain (status = 'none')**

```
┌─────────────────────────────────────────────────────────────┐
│ Custom Domain                                                │
├─────────────────────────────────────────────────────────────┤
│ Use your own domain for your storefront                     │
│                                                              │
│ Domain: [____________________________] (e.g., shop.example) │
│         .com or www.example.com)                            │
│                                                              │
│ [Set Up Custom Domain]                                      │
└─────────────────────────────────────────────────────────────┘
```

**State 2: Pending Verification (status = 'pending')**

```
┌─────────────────────────────────────────────────────────────┐
│ Custom Domain: shop.roastercoffee.com                       │
├─────────────────────────────────────────────────────────────┤
│ ⚠ Verification Required                                     │
│                                                              │
│ Add these DNS records to your domain:                       │
│                                                              │
│ 1. CNAME Record                                             │
│    Name:  shop.roastercoffee.com                            │
│    Value: custom.freyja.app                                 │
│    TTL:   3600                                              │
│                                                              │
│ 2. TXT Record                                               │
│    Name:  _freyja-verify.shop.roastercoffee.com            │
│    Value: freyja-verify=a7b2c3d4e5f6g7h8i9j0k1l2m3n4       │
│    TTL:   3600                                              │
│                                                              │
│ [Verify Domain]  [Cancel]                                   │
│                                                              │
│ DNS changes can take up to 48 hours to propagate.           │
└─────────────────────────────────────────────────────────────┘
```

**State 3: Verified (status = 'verified')**

```
┌─────────────────────────────────────────────────────────────┐
│ Custom Domain: shop.roastercoffee.com                       │
├─────────────────────────────────────────────────────────────┤
│ ✓ Domain Verified                                           │
│                                                              │
│ Your domain has been verified. Activate it to start serving │
│ traffic on your custom domain.                              │
│                                                              │
│ [Activate Domain]  [Remove Domain]                          │
└─────────────────────────────────────────────────────────────┘
```

**State 4: Active (status = 'active')**

```
┌─────────────────────────────────────────────────────────────┐
│ Custom Domain: shop.roastercoffee.com                       │
├─────────────────────────────────────────────────────────────┤
│ ✓ Active                                                    │
│                                                              │
│ Your storefront is live at:                                 │
│ https://shop.roastercoffee.com                              │
│                                                              │
│ SSL Certificate: Valid until Jan 15, 2025                   │
│                                                              │
│ [Remove Domain]                                             │
└─────────────────────────────────────────────────────────────┘
```

**State 5: Failed (status = 'failed')**

```
┌─────────────────────────────────────────────────────────────┐
│ Custom Domain: shop.roastercoffee.com                       │
├─────────────────────────────────────────────────────────────┤
│ ✗ Verification Failed                                       │
│                                                              │
│ Error: CNAME record not found. Please verify your DNS       │
│ settings and try again.                                     │
│                                                              │
│ [Retry Verification]  [Remove Domain]                       │
└─────────────────────────────────────────────────────────────┘
```

### Routes

```
GET    /admin/settings/domain          - Show domain settings page
POST   /admin/settings/domain          - Initiate domain setup (status: none → pending)
POST   /admin/settings/domain/verify   - Verify DNS records (status: pending → verified or failed)
POST   /admin/settings/domain/activate - Activate domain (status: verified → active)
DELETE /admin/settings/domain          - Remove custom domain (any status → none)
```

---

## Error Handling and Edge Cases

### Edge Case 1: Domain Already in Use

**Scenario:** Tenant tries to add `shop.example.com`, but another tenant already has it.

**Handling:**
- UNIQUE constraint on `tenants.custom_domain` prevents duplicate entries
- Return error: "This domain is already in use by another store."

### Edge Case 2: DNS Propagation Delay

**Scenario:** Tenant adds DNS records but clicks "Verify" before propagation completes.

**Handling:**
- Show friendly error: "DNS records not found yet. Changes can take up to 48 hours to propagate. Please try again later."
- Allow unlimited retry attempts (no rate limiting)

### Edge Case 3: CNAME Removed After Activation

**Scenario:** Domain is active, but tenant later removes the CNAME record.

**Handling:**
- **Periodic health check job:** Background job runs daily, checks all active custom domains
- If CNAME no longer points to `custom.freyja.app`, mark domain as `failed`
- Send email notification to tenant: "Your custom domain configuration has an issue."
- Storefront automatically falls back to `*.freyja.app` subdomain (no downtime)

### Edge Case 4: TXT Record Removed After Verification

**Scenario:** Tenant removes TXT record after verification.

**Handling:**
- Not a problem: TXT record is only required for initial verification
- Once domain is `verified` or `active`, TXT record can be safely removed

### Edge Case 5: Invalid Domain Format

**Scenario:** Tenant enters `example.com` (apex domain) or `http://example.com` or `example.com/path`.

**Validation Rules:**
- Must be a valid hostname (no protocol, no path)
- Must be a subdomain (not apex domain): Reject `example.com`, accept `www.example.com` or `shop.example.com`
- Use `net.ParseRequestURI` to validate format
- Return error: "Please enter a subdomain (e.g., shop.example.com or www.example.com). Apex domains (example.com) are not supported yet."

**Future:** Support apex domains by requiring two DNS records (A + AAAA pointing to Freyja IPs).

### Edge Case 6: Let's Encrypt Rate Limits

**Scenario:** Caddy fails to obtain certificate due to Let's Encrypt rate limits.

**Let's Encrypt Limits:**
- 50 certificates per registered domain per week
- 5 duplicate certificates per week

**Handling:**
- Unlikely to hit limits (each custom domain is unique)
- If hit: Caddy will return error, tenant's domain won't serve traffic
- Mitigation: Use Let's Encrypt staging environment for development/testing

### Edge Case 7: User Changes Domain While Active

**Scenario:** Tenant wants to change from `shop.example.com` to `store.example.com`.

**Handling:**
- Require removal of current domain first (DELETE → POST flow)
- Prevents orphaned certificates and simplifies state management

---

## Security Considerations

### 1. Verification Token Security

**Threat:** Attacker gains database access and sees verification tokens in plaintext.

**Mitigation:**
- Store SHA-256 hash of token in `custom_domain_verification_token`
- Raw token only shown in UI once (never logged)
- Tokens expire after 7 days (implicit: if not verified within 7 days, tenant must restart flow)

### 2. Subdomain Takeover

**Threat:** Tenant removes domain from Freyja but forgets to remove CNAME record. Attacker signs up for Freyja and adds the same domain.

**Mitigation:**
- DNS verification required: Attacker would need access to victim's DNS to add TXT record
- UNIQUE constraint prevents two tenants from claiming the same domain simultaneously

### 3. Certificate Abuse

**Threat:** Attacker tries to issue Let's Encrypt certificate for arbitrary domains via Caddy.

**Mitigation:**
- Caddy's `ask` endpoint (`/api/validate-domain`) only returns 200 for verified, active domains
- Without 200 response, Caddy rejects certificate issuance
- Rate limiting on `/api/validate-domain` (100 req/min per IP) prevents brute-force

### 4. SSRF via DNS Lookup

**Threat:** Attacker provides malicious domain to trigger DNS lookups to internal IPs.

**Mitigation:**
- DNS lookups only to external resolvers (use `8.8.8.8` or Cloudflare `1.1.1.1`)
- 10-second timeout on DNS queries
- Domain format validation (must be valid hostname)

### 5. Redirect Loop

**Threat:** Misconfiguration causes infinite redirect between custom domain and subdomain.

**Mitigation:**
- Redirect logic only triggers when:
  1. Request is on `.freyja.app` subdomain
  2. Custom domain status is `active`
  3. Path is not protected (`/admin/`, `/saas/`, etc.)
- Custom domain requests never redirect (no `CustomDomainRedirect` on custom domain traffic)

---

## Performance Considerations

### 1. Tenant Resolution Latency

**Critical Path:** Every request requires tenant lookup.

**Optimization:**
- Index lookups: Both `idx_tenants_custom_domain` and `idx_tenants_slug` are indexed
- Partial index: `WHERE custom_domain_status = 'active'` reduces index size
- Database connection pooling: Reuse connections for tenant lookups

**Benchmark Target:** < 5ms for tenant resolution on 99th percentile.

### 2. DNS Verification Performance

**Non-Critical Path:** Verification happens manually via UI, not on request path.

**Optimization:**
- Use goroutine for DNS lookups (don't block HTTP handler)
- Cache negative results for 5 minutes (prevent repeated failed lookups)

### 3. Caddy Validation Endpoint

**Critical Path:** Caddy calls `/api/validate-domain` before issuing certificates.

**Optimization:**
- Single database query: `SELECT 1 FROM tenants WHERE custom_domain = $1 AND custom_domain_status = 'active'`
- Use prepared statement (sqlc generates this)
- Cache results for 1 minute (certificate issuance doesn't happen frequently)

---

## Background Jobs

### Job 1: Daily Custom Domain Health Check

**Purpose:** Verify all active custom domains still have valid CNAME records.

**Schedule:** Daily at 3:00 AM UTC.

**Logic:**

```go
func HealthCheckCustomDomains(ctx context.Context) error {
    activeDomains := queries.GetTenantsWithActiveCustomDomains(ctx)

    for _, tenant := range activeDomains {
        // Check CNAME still points to custom.freyja.app
        cnameTarget := lookupCNAME(tenant.CustomDomain)

        if !strings.HasSuffix(cnameTarget, "custom.freyja.app") {
            // Mark domain as failed
            queries.UpdateCustomDomainStatus(ctx, UpdateParams{
                TenantID: tenant.ID,
                Status:   "failed",
                ErrorMessage: "CNAME record no longer points to custom.freyja.app",
            })

            // Send email notification
            emailService.SendCustomDomainFailureEmail(tenant)
        } else {
            // Update last_checked_at timestamp
            queries.UpdateCustomDomainLastChecked(ctx, tenant.ID)
        }
    }

    return nil
}
```

**Email Template:** `web/templates/email/custom_domain_failure.html`

### Job 2: Cleanup Stale Pending Domains

**Purpose:** Remove domains stuck in `pending` status for > 30 days.

**Schedule:** Weekly on Sundays at 4:00 AM UTC.

**Logic:**

```sql
-- Reset domains pending for > 30 days
UPDATE tenants
SET custom_domain = NULL,
    custom_domain_status = 'none',
    custom_domain_verification_token = NULL
WHERE custom_domain_status = 'pending'
  AND updated_at < NOW() - INTERVAL '30 days';
```

---

## Telemetry and Metrics

### Prometheus Metrics

```go
// Custom domain setup funnel
customDomainInitiated       = prometheus.NewCounterVec(...)  // status: none → pending
customDomainVerified        = prometheus.NewCounterVec(...)  // status: pending → verified
customDomainActivated       = prometheus.NewCounterVec(...)  // status: verified → active
customDomainFailed          = prometheus.NewCounterVec(...)  // Any status → failed
customDomainRemoved         = prometheus.NewCounterVec(...)  // Any status → none

// DNS verification latency
dnsVerificationDuration     = prometheus.NewHistogramVec(...)  // Time to verify DNS

// Caddy validation endpoint
caddyValidationRequests     = prometheus.NewCounterVec(...)    // Total requests
caddyValidationLatency      = prometheus.NewHistogramVec(...)  // Query latency
```

**Dashboard Queries:**
- Custom domain adoption rate: `custom_domain_activated_total / total_tenants`
- Verification failure rate: `custom_domain_failed_total / custom_domain_initiated_total`
- Caddy validation P99 latency: `histogram_quantile(0.99, caddy_validation_latency)`

---

## Migration Path (No Downtime)

### Phase 1: Schema Changes

1. Add columns to `tenants` table (migration `00028_add_custom_domains.sql`)
2. Create indexes
3. Deploy migration (non-breaking, columns are nullable)

### Phase 2: Application Changes

1. Add validation endpoint (`/api/validate-domain`)
2. Add service layer (`CustomDomainService`)
3. Add admin UI routes and templates
4. Deploy application (feature disabled for now)

### Phase 3: Caddy Configuration

1. Update Caddyfile with on-demand TLS config
2. Reload Caddy configuration (`caddy reload`)

### Phase 4: Enable Feature

1. Update feature flag (`CUSTOM_DOMAINS_ENABLED=true`)
2. Restart application
3. Feature now visible in admin UI

### Rollback Plan

1. Set feature flag `CUSTOM_DOMAINS_ENABLED=false`
2. Revert Caddyfile changes (`caddy reload`)
3. Custom domains remain in database (can re-enable later)

---

## Testing Strategy

### Unit Tests

- `CustomDomainService.InitiateVerification()` - Token generation, domain validation
- `CustomDomainService.CheckVerification()` - DNS lookup, token matching
- `CustomDomainService.ActivateDomain()` - Status transitions
- Tenant resolution middleware - Custom domain vs. subdomain routing

### Integration Tests

- End-to-end flow: Initiate → Verify → Activate
- DNS lookup with mocked responses
- Caddy validation endpoint (200 for active, 404 for inactive)

### Manual Testing Checklist

- [ ] Add custom domain in UI
- [ ] Verify DNS instructions are shown correctly
- [ ] Add CNAME and TXT records to test domain
- [ ] Click "Verify Domain" - should succeed
- [ ] Click "Activate Domain" - should mark as active
- [ ] Visit custom domain - should serve storefront
- [ ] Visit `/admin/*` on custom domain - should work
- [ ] Visit subdomain storefront - should redirect to custom domain
- [ ] Visit `/admin/*` on subdomain - should NOT redirect
- [ ] Remove CNAME record - health check should detect failure within 24 hours
- [ ] Remove custom domain - should revert to subdomain

---

## Future Enhancements (Not in Initial Scope)

### Apex Domain Support

**Challenge:** Apex domains (e.g., `example.com`) cannot use CNAME records (DNS spec limitation).

**Solution:**
- Require A and AAAA records pointing to Freyja's IP addresses
- Need static IP addresses for infrastructure
- More complex setup for tenants

**Decision:** Defer to post-MVP. Most storefronts use subdomains (`www`, `shop`, `store`).

### Multiple Custom Domains per Tenant

**Use Case:** Tenant wants `shop.example.com` AND `cafe.example.com`.

**Solution:**
- Migrate to `custom_domains` table (one-to-many relationship)
- Update tenant resolution middleware to check all domains

**Decision:** Defer until user demand is validated. 99% of tenants will use one domain.

### Automatic SSL Certificate Monitoring

**Enhancement:** Show SSL certificate expiration date in admin UI.

**Implementation:**
- Query Caddy's certificate storage
- Display "Valid until [date]" in UI
- Caddy auto-renews, so this is informational only

**Decision:** Nice-to-have, not critical for MVP.

### Custom Domain Analytics

**Enhancement:** Track traffic by domain (custom vs. subdomain).

**Implementation:**
- Add `domain_type` label to Prometheus metrics
- Dashboard shows % of traffic on custom domain

**Decision:** Add after feature adoption is proven.

---

## Open Questions and Assumptions

### Assumptions

1. **Caddy is the reverse proxy** - All traffic flows through Caddy, which handles TLS termination
2. **Single Freyja instance** - No multi-region or load balancing complexity initially
3. **PostgreSQL is primary datastore** - No Redis or caching layer for tenant lookups
4. **Target users are non-technical** - UI must be extremely clear with step-by-step instructions
5. **Subdomain-only initially** - Apex domain support deferred to future iteration

### Questions for Validation

1. **Q:** Should we allow email addresses on custom domains (e.g., `order-confirmation@shop.example.com`)?
   **A:** No, initial implementation uses platform email. Custom email domains deferred to future.

2. **Q:** What happens if tenant changes custom domain while active?
   **A:** Require removal first. Prevents edge cases with certificate orphaning.

3. **Q:** Should we rate-limit verification attempts?
   **A:** No. DNS propagation is unpredictable; allow unlimited retries.

4. **Q:** How often should health checks run?
   **A:** Daily at 3 AM UTC. If domain is business-critical, tenant will notice and contact support within 24h.

5. **Q:** Should we support internationalized domain names (IDN)?
   **A:** Yes, but convert to Punycode before storage. Go's `net` package handles this automatically.

---

## Summary

This custom domain system provides tenants with professional branding while maintaining operational simplicity. The design prioritizes:

- **Security:** DNS verification prevents domain takeover
- **Reliability:** Subdomain fallback ensures zero downtime
- **Simplicity:** One domain per tenant, clear UI flow
- **Performance:** Indexed lookups, no caching required initially
- **Maintainability:** Standard library DNS lookups, Caddy handles TLS complexity

The architecture is reversible (domains can be removed), extensible (can add apex domain support later), and aligns with existing Freyja patterns (interface-based services, sqlc queries, operator middleware).
