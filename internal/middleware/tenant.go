// Package middleware provides HTTP middleware for the Freyja application.
//
// This file contains tenant resolution middleware for multi-tenant subdomain routing.
// It replaces the previous custom_domain.go with a more comprehensive approach.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/hiri/internal/tenant"
)

// TenantConfig holds configuration for tenant resolution middleware.
type TenantConfig struct {
	// BaseDomain is the root domain for subdomain extraction (e.g., "hiri.coffee" or "lvh.me:3000").
	// Tenant subdomains are extracted as: {slug}.BaseDomain
	BaseDomain string

	// AppDomain is the SaaS application domain (e.g., "app.hiri.coffee").
	// Requests to this domain skip tenant resolution entirely.
	AppDomain string

	// Resolver is the tenant resolver for database lookups.
	Resolver tenant.Resolver

	// Logger is the structured logger for middleware operations.
	// If nil, uses slog.Default().
	Logger *slog.Logger
}

// ResolveTenant creates middleware that resolves tenant from request host.
//
// Resolution order:
//  1. Check if host matches AppDomain - if so, skip resolution (SaaS app routes)
//  2. Check if host is BaseDomain (apex) - if so, skip resolution (marketing site)
//  3. Check for "www" subdomain - redirect to BaseDomain
//  4. Check if host is a subdomain of BaseDomain - resolve by slug
//  5. Otherwise treat as custom domain - resolve by custom domain
//
// After resolution, tenant status is checked:
//   - "active": continue normally, tenant added to context
//   - "pending": respond with 404 (storefront doesn't exist yet)
//   - "suspended": respond with 503 (temporarily unavailable)
//   - "cancelled": respond with 404 (storefront no longer exists)
//
// TODO: Implement the resolution logic
//   - Parse host to extract subdomain
//   - Handle AppDomain bypass
//   - Handle www redirect
//   - Call resolver.BySlug or resolver.ByCustomDomain
//   - Check tenant status and respond appropriately
//   - Add tenant to context using tenant.NewContext
func ResolveTenant(cfg TenantConfig) func(http.Handler) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1: Extract host without port
			host := r.Host
			if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
				host = host[:colonIndex]
			}

			// Step 2: Check for AppDomain (skip resolution)
			appDomainHost := stripPort(cfg.AppDomain)
			if host == appDomainHost {
				next.ServeHTTP(w, r)
				return
			}

			// Step 3: Check for BaseDomain apex (marketing site, skip resolution)
			baseDomainHost := stripPort(cfg.BaseDomain)
			if host == baseDomainHost {
				next.ServeHTTP(w, r)
				return
			}

			// Step 4: Extract subdomain
			subdomain := extractSubdomainForTenant(host, baseDomainHost)

			// Step 5: Handle www redirect
			if subdomain == "www" {
				redirectURL := "https://" + cfg.BaseDomain + r.URL.Path
				if r.URL.RawQuery != "" {
					redirectURL += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
				return
			}

			// Step 6: Resolve tenant
			var t *tenant.Tenant
			var err error
			if subdomain != "" {
				// Subdomain routing
				t, err = cfg.Resolver.BySlug(r.Context(), subdomain)
			} else {
				// Custom domain routing
				t, err = cfg.Resolver.ByCustomDomain(r.Context(), host)
			}

			// Step 7: Handle resolution errors
			if err != nil {
				if err == tenant.ErrTenantNotFound {
					respondTenantNotFound(w, r)
					return
				}
				if err == tenant.ErrCustomDomainNotActive {
					respondTenantNotFound(w, r)
					return
				}
				logger.Error("tenant resolution failed", "host", host, "error", err)
				respondTenantInternalError(w, r, err)
				return
			}

			// Step 8: Check tenant status
			switch t.Status {
			case "active":
				// Continue normally
			case "pending":
				respondTenantNotFound(w, r)
				return
			case "suspended":
				respondTenantServiceUnavailable(w, r, "This store is temporarily unavailable")
				return
			case "cancelled":
				respondTenantNotFound(w, r)
				return
			default:
				logger.Warn("unknown tenant status", "tenant_id", t.ID, "status", t.Status)
				respondTenantNotFound(w, r)
				return
			}

			// Step 9: Add tenant to context and continue
			ctx := tenant.NewContext(r.Context(), t)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant is middleware that ensures a tenant is present in context.
// Returns 404 if no tenant is found. Use this on routes that require tenant context.
//
// This should be applied AFTER ResolveTenant middleware.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := tenant.FromContext(r.Context())
		if t == nil {
			respondTenantNotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractSubdomainForTenant extracts the subdomain from a host given a base domain.
// Returns empty string if host doesn't have a subdomain or doesn't match base domain.
//
// NOTE: This duplicates extractSubdomain from custom_domain.go but with a different name
// to avoid conflicts during the migration. After custom_domain.go is removed, this can
// be renamed to extractSubdomain.
//
// Examples:
//
//	extractSubdomainForTenant("acme.hiri.coffee", "hiri.coffee") -> "acme"
//	extractSubdomainForTenant("hiri.coffee", "hiri.coffee") -> ""
//	extractSubdomainForTenant("shop.example.com", "hiri.coffee") -> "" (custom domain)
//	extractSubdomainForTenant("sub.acme.hiri.coffee", "hiri.coffee") -> "" (nested subdomain)
func extractSubdomainForTenant(host, baseDomain string) string {
	suffix := "." + baseDomain
	if !strings.HasSuffix(host, suffix) {
		return ""
	}

	subdomain := strings.TrimSuffix(host, suffix)
	// Reject nested subdomains (e.g., "sub.tenant.hiri.coffee")
	if subdomain == "" || strings.Contains(subdomain, ".") {
		return ""
	}

	return subdomain
}

// stripPort removes the port from a host string.
// Returns the host unchanged if no port is present.
func stripPort(host string) string {
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		return host[:colonIndex]
	}
	return host
}

// respondTenantNotFound sends a 404 Not Found response for tenant resolution.
// NOTE: Uses a different name to avoid conflict with respondNotFound in middleware.go
// After migration, consolidate these functions.
func respondTenantNotFound(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement - possibly render a nice 404 page
	// For now, simple text response
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not Found"))
}

// respondTenantServiceUnavailable sends a 503 Service Unavailable response.
func respondTenantServiceUnavailable(w http.ResponseWriter, r *http.Request, message string) {
	// TODO: Implement - possibly render a nice 503 page with retry-after header
	w.Header().Set("Retry-After", "3600") // Suggest retry in 1 hour
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(message))
}

// respondTenantInternalError sends a 500 Internal Server Error response.
// NOTE: Uses a different name to avoid conflict with respondInternalError in middleware.go
func respondTenantInternalError(w http.ResponseWriter, r *http.Request, err error) {
	// TODO: Log error, possibly report to Sentry
	// For now, generic error message (don't leak internal details)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

// MIGRATION NOTES:
//
// There are two sources of tenant context in the application:
//
// 1. Storefront routes (subdomain/custom domain routing):
//    - Set by ResolveTenant middleware
//    - Uses tenant.NewContext() / tenant.FromContext()
//    - Tenant resolved from host header
//
// 2. Admin routes (operator session):
//    - Set by RequireActiveTenant middleware in operator.go
//    - Uses TenantContextKey from operator.go
//    - Tenant resolved from operator's TenantID
//
// After migration, both should use the tenant package consistently.
// The operator middleware should be updated to use tenant.NewContext()
// instead of context.WithValue(ctx, TenantContextKey, &tenant).
//
// TODO: Update operator.go to use tenant package:
//
// BEFORE (current):
//   ctx = context.WithValue(r.Context(), TenantContextKey, &tenant)
//
// AFTER (migrated):
//   t := &tenant.Tenant{
//       ID:     repoTenant.ID,
//       Slug:   repoTenant.Slug,
//       Name:   repoTenant.Name,
//       Status: repoTenant.Status,
//   }
//   ctx = tenant.NewContext(r.Context(), t)
//
// This ensures services can use tenant.IDFromContext() regardless of
// whether the request came via storefront or admin routes.

// bridgeTenantContext creates a tenant.Tenant from the repository.Tenant
// stored by operator middleware. This is a temporary bridge during migration.
//
// TODO: Remove after operator.go is updated to use tenant package directly.
func bridgeTenantContext(ctx context.Context) *tenant.Tenant {
	// Check for tenant from new tenant package first
	if t := tenant.FromContext(ctx); t != nil {
		return t
	}

	// Fall back to operator middleware's tenant context
	// This requires importing repository package which creates a dependency cycle
	// For now, this is documented as a TODO for the implementer

	return nil
}
