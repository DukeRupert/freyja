# Custom Domain Middleware Changes

This document describes the required changes to the tenant resolution and redirect middleware to support custom domains.

---

## Required Middleware Changes

### 1. Tenant Resolution Middleware

**File:** `internal/middleware/tenant.go` (or wherever tenant resolution is implemented)

**Current behavior:**
- Extracts subdomain from `Host` header (e.g., `roastercoffee.freyja.app` → `roastercoffee`)
- Looks up tenant by `slug` column in database
- Adds tenant to request context

**Required changes:**

```go
func ResolveTenant(queries *repository.Queries) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := r.Context()
            host := r.Host

            // Remove port if present (e.g., "example.com:443" → "example.com")
            if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
                host = host[:colonIndex]
            }

            var tenant *repository.Tenant
            var err error

            // Case 1: Request is on a custom domain (no .freyja.app suffix)
            if !strings.HasSuffix(host, ".freyja.app") {
                // Lookup tenant by custom_domain
                tenant, err = queries.GetTenantByCustomDomain(ctx, host)
                if err != nil {
                    if errors.Is(err, sql.ErrNoRows) {
                        http.Error(w, "Store not found", http.StatusNotFound)
                        return
                    }
                    http.Error(w, "Internal server error", http.StatusInternalServerError)
                    return
                }

                // Verify domain is active (should be guaranteed by query, but double-check)
                if tenant.CustomDomainStatus != "active" {
                    http.Error(w, "Custom domain not active", http.StatusNotFound)
                    return
                }
            } else {
                // Case 2: Request is on default subdomain (*.freyja.app)
                subdomain := extractSubdomain(host) // e.g., "roastercoffee"
                if subdomain == "" {
                    http.Error(w, "Invalid subdomain", http.StatusBadRequest)
                    return
                }

                tenant, err = queries.GetTenantBySlug(ctx, subdomain)
                if err != nil {
                    if errors.Is(err, sql.ErrNoRows) {
                        http.Error(w, "Store not found", http.StatusNotFound)
                        return
                    }
                    http.Error(w, "Internal server error", http.StatusInternalServerError)
                    return
                }
            }

            // Add tenant to context
            ctx = context.WithValue(ctx, tenantContextKey, tenant)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Helper function to extract subdomain from *.freyja.app
func extractSubdomain(host string) string {
    // Input: "roastercoffee.freyja.app"
    // Output: "roastercoffee"
    parts := strings.Split(host, ".")
    if len(parts) >= 3 {
        return parts[0]
    }
    return ""
}
```

**Performance notes:**
- Custom domain lookup uses `idx_tenants_custom_domain_active` index (very fast)
- Subdomain lookup uses existing `idx_tenants_slug` index
- Both are indexed lookups, no performance degradation
- Consider adding caching layer if needed (e.g., 1-minute TTL in-memory cache)

---

### 2. Custom Domain Redirect Middleware

**File:** `internal/middleware/custom_domain_redirect.go` (new file)

**Purpose:** Redirect storefront routes from subdomain to custom domain when active.

**Implementation:**

```go
package middleware

import (
    "net/http"
    "strings"
)

// CustomDomainRedirect redirects storefront requests from subdomain to custom domain
// when a tenant has an active custom domain configured.
//
// Behavior:
// - If tenant has active custom domain AND request is on subdomain:
//   - Storefront routes (/, /products/*, etc.) → Redirect to custom domain (301)
//   - Admin routes (/admin/*, /saas/*) → No redirect (accessible on both)
//   - API routes (/api/*, /webhooks/*) → No redirect (accessible on both)
// - If tenant has no custom domain OR request is on custom domain:
//   - No redirect (serve request normally)
//
// Must be applied AFTER ResolveTenant middleware (requires tenant in context)
func CustomDomainRedirect(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        tenant := GetTenantFromContext(ctx)

        // If no tenant, skip (should be caught by ResolveTenant)
        if tenant == nil {
            next.ServeHTTP(w, r)
            return
        }

        // Only redirect if:
        // 1. Tenant has active custom domain
        // 2. Request is on default subdomain (*.freyja.app)
        // 3. Request is NOT to protected routes (admin/saas/api/webhooks)
        if tenant.CustomDomainStatus == "active" &&
           tenant.CustomDomain != "" &&
           strings.HasSuffix(r.Host, ".freyja.app") &&
           !isProtectedRoute(r.URL.Path) {

            // Build redirect URL
            scheme := "https" // Always HTTPS for custom domains
            redirectURL := scheme + "://" + tenant.CustomDomain + r.URL.Path
            if r.URL.RawQuery != "" {
                redirectURL += "?" + r.URL.RawQuery
            }

            // 301 Permanent Redirect
            // - SEO: Search engines will index custom domain
            // - Performance: Browsers cache the redirect
            http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
            return
        }

        next.ServeHTTP(w, r)
    })
}

// isProtectedRoute returns true if the path should NOT redirect
// Protected routes remain accessible on both subdomain and custom domain
func isProtectedRoute(path string) bool {
    protectedPrefixes := []string{
        "/admin/",    // Admin dashboard
        "/saas/",     // SaaS onboarding and billing
        "/api/",      // API endpoints (including Caddy validation)
        "/webhooks/", // Webhook handlers (Stripe, etc.)
    }

    for _, prefix := range protectedPrefixes {
        if strings.HasPrefix(path, prefix) {
            return true
        }
    }

    return false
}
```

**Why 301 Permanent Redirect?**
- SEO: Search engines will index the custom domain as the canonical URL
- Performance: Browsers cache permanent redirects (fewer round-trips)
- User experience: Custom domain becomes the primary URL

**Why keep admin/saas accessible on both domains?**
- Reliability: If custom domain DNS fails, admin can still access via subdomain
- Support: Freyja support staff can access tenant admin via subdomain
- Testing: Operators can test custom domain without losing admin access

---

### 3. Middleware Application Order

**File:** `main.go` or `cmd/server/main.go`

The middleware must be applied in this order:

```go
// Global middleware (applies to all routes)
r.Use(
    middleware.Logger(),                    // 1. Logging
    middleware.Recover(),                   // 2. Panic recovery
    middleware.ResolveTenant(queries),      // 3. Tenant resolution (REQUIRED FIRST)
    middleware.CustomDomainRedirect,        // 4. Custom domain redirect (AFTER tenant resolution)
    middleware.SentryContext(),             // 5. Error tracking context
    // ... other middleware
)
```

**Critical:** `CustomDomainRedirect` MUST come after `ResolveTenant` because it needs tenant data from context.

---

## Testing the Middleware

### Unit Tests

**File:** `internal/middleware/custom_domain_redirect_test.go`

```go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestCustomDomainRedirect(t *testing.T) {
    tests := []struct {
        name               string
        tenant             *repository.Tenant
        requestHost        string
        requestPath        string
        expectedStatus     int
        expectedLocation   string
        expectRedirect     bool
    }{
        {
            name: "redirect storefront from subdomain to custom domain",
            tenant: &repository.Tenant{
                CustomDomain:       "shop.example.com",
                CustomDomainStatus: "active",
            },
            requestHost:      "roastercoffee.freyja.app",
            requestPath:      "/products/ethiopian-yirgacheffe",
            expectedStatus:   http.StatusMovedPermanently,
            expectedLocation: "https://shop.example.com/products/ethiopian-yirgacheffe",
            expectRedirect:   true,
        },
        {
            name: "no redirect for admin routes",
            tenant: &repository.Tenant{
                CustomDomain:       "shop.example.com",
                CustomDomainStatus: "active",
            },
            requestHost:    "roastercoffee.freyja.app",
            requestPath:    "/admin/products",
            expectedStatus: http.StatusOK,
            expectRedirect: false,
        },
        {
            name: "no redirect if request is already on custom domain",
            tenant: &repository.Tenant{
                CustomDomain:       "shop.example.com",
                CustomDomainStatus: "active",
            },
            requestHost:    "shop.example.com",
            requestPath:    "/products",
            expectedStatus: http.StatusOK,
            expectRedirect: false,
        },
        {
            name: "no redirect if domain status is not active",
            tenant: &repository.Tenant{
                CustomDomain:       "shop.example.com",
                CustomDomainStatus: "pending",
            },
            requestHost:    "roastercoffee.freyja.app",
            requestPath:    "/products",
            expectedStatus: http.StatusOK,
            expectRedirect: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // TODO: Implement test
            // 1. Create request with tt.requestHost and tt.requestPath
            // 2. Add tenant to context
            // 3. Apply CustomDomainRedirect middleware
            // 4. Check response status matches tt.expectedStatus
            // 5. If expectRedirect, check Location header matches tt.expectedLocation
        })
    }
}
```

### Integration Tests

**File:** `internal/middleware/tenant_resolution_test.go`

```go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestTenantResolution(t *testing.T) {
    tests := []struct {
        name           string
        requestHost    string
        expectedTenant string
        expectedError  bool
    }{
        {
            name:           "resolve tenant by subdomain",
            requestHost:    "roastercoffee.freyja.app",
            expectedTenant: "roastercoffee",
            expectedError:  false,
        },
        {
            name:           "resolve tenant by custom domain",
            requestHost:    "shop.example.com",
            expectedTenant: "tenant-with-custom-domain",
            expectedError:  false,
        },
        {
            name:          "error on invalid subdomain",
            requestHost:   "nonexistent.freyja.app",
            expectedError: true,
        },
        {
            name:          "error on invalid custom domain",
            requestHost:   "invalid-domain.com",
            expectedError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // TODO: Implement test
            // 1. Create request with tt.requestHost
            // 2. Apply ResolveTenant middleware
            // 3. Check tenant was added to context
            // 4. Verify tenant matches expected
        })
    }
}
```

---

## Deployment Checklist

Before deploying custom domain support:

1. [ ] Add migration `00028_add_custom_domains.sql`
2. [ ] Update `ResolveTenant` middleware to handle custom domains
3. [ ] Create `CustomDomainRedirect` middleware
4. [ ] Apply middleware in correct order in `main.go`
5. [ ] Add sqlc queries (`custom_domains.sql`)
6. [ ] Implement `CustomDomainService`
7. [ ] Implement `CustomDomainHandler` (admin UI)
8. [ ] Implement `DomainValidationHandler` (Caddy endpoint)
9. [ ] Update Caddyfile with on-demand TLS configuration
10. [ ] Add unit tests for middleware
11. [ ] Add integration tests for tenant resolution
12. [ ] Test manually with a real domain

---

## Troubleshooting

### Issue: Redirect loop

**Symptom:** Browser shows "Too many redirects" error

**Cause:** Redirect logic triggers on custom domain requests

**Fix:** Ensure redirect only happens when:
- Request is on `.freyja.app` subdomain (not custom domain)
- Custom domain status is `active`
- Path is not protected (`/admin/`, `/saas/`, etc.)

### Issue: Admin panel not accessible after custom domain activation

**Symptom:** `/admin/*` redirects to custom domain and breaks

**Cause:** `isProtectedRoute` not correctly identifying admin routes

**Fix:** Check `isProtectedRoute` function includes `/admin/` prefix

### Issue: Tenant not found on custom domain

**Symptom:** 404 error when visiting custom domain

**Cause:**
- Domain status is not `active` in database
- DNS records not configured correctly
- Index `idx_tenants_custom_domain_active` not created

**Fix:**
- Check `custom_domain_status` in database
- Verify CNAME points to `custom.freyja.app`
- Run migration to create index
