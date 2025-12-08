package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// BaseDomain is the platform domain (e.g., "freyja.app")
// In production, this should come from configuration
const BaseDomain = "freyja.app"

// ResolveTenantByHost resolves the tenant from the request host.
// It handles both custom domains and subdomains (*.freyja.app).
//
// This middleware should be applied to routes that need tenant context
// based on hostname (primarily storefront routes in multi-tenant mode).
//
// For admin routes, tenant resolution happens through operator authentication.
func ResolveTenantByHost(queries *repository.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			host := r.Host

			// Remove port if present (e.g., "example.com:443" -> "example.com")
			if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
				host = host[:colonIndex]
			}

			var tenant *repository.Tenant

			// Case 1: Request is on a custom domain (no .freyja.app suffix)
			if !strings.HasSuffix(host, "."+BaseDomain) && host != BaseDomain {
				// Lookup tenant by custom_domain
				tenantRow, lookupErr := queries.GetTenantByCustomDomain(ctx, pgtype.Text{String: host, Valid: true})
				if lookupErr != nil {
					if lookupErr == pgx.ErrNoRows {
						slog.Debug("tenant resolution: custom domain not found",
							"host", host,
						)
						respondNotFound(w, r)
						return
					}
					slog.Error("tenant resolution: database error",
						"host", host,
						"error", lookupErr,
					)
					respondInternalError(w, r, lookupErr)
					return
				}

				// Convert row to tenant
				tenant = &repository.Tenant{
					ID:                 tenantRow.ID,
					Name:               tenantRow.Name,
					Slug:               tenantRow.Slug,
					CustomDomain:       tenantRow.CustomDomain,
					CustomDomainStatus: tenantRow.CustomDomainStatus,
					Status:             tenantRow.Status,
				}

				// Verify domain is active
				if tenant.CustomDomainStatus != "active" {
					slog.Warn("tenant resolution: custom domain not active",
						"host", host,
						"status", tenant.CustomDomainStatus,
					)
					respondWithError(w, r, domain.Errorf(domain.ENOTFOUND, "", "Custom domain not active"))
					return
				}
			} else {
				// Case 2: Request is on default subdomain (*.freyja.app)
				subdomain := extractSubdomain(host, BaseDomain)
				if subdomain == "" {
					// Direct access to freyja.app (no subdomain) - this is the marketing site
					// Let the request proceed without tenant context
					next.ServeHTTP(w, r)
					return
				}

				tenantRow, lookupErr := queries.GetTenantBySlug(ctx, subdomain)
				if lookupErr != nil {
					if lookupErr == pgx.ErrNoRows {
						slog.Debug("tenant resolution: subdomain not found",
							"subdomain", subdomain,
						)
						respondNotFound(w, r)
						return
					}
					slog.Error("tenant resolution: database error",
						"subdomain", subdomain,
						"error", lookupErr,
					)
					respondInternalError(w, r, lookupErr)
					return
				}

				tenant = &tenantRow
			}

			// Add tenant to context
			ctx = context.WithValue(ctx, TenantContextKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CustomDomainRedirect redirects storefront requests from subdomain to custom domain
// when a tenant has an active custom domain configured.
//
// Behavior:
// - If tenant has active custom domain AND request is on subdomain:
//   - Storefront routes (/, /products/*, etc.) -> Redirect to custom domain (301)
//   - Admin routes (/admin/*, /saas/*) -> No redirect (accessible on both)
//   - API routes (/api/*, /webhooks/*) -> No redirect (accessible on both)
// - If tenant has no custom domain OR request is on custom domain:
//   - No redirect (serve request normally)
//
// Must be applied AFTER ResolveTenantByHost middleware (requires tenant in context).
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
		host := r.Host
		if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}

		hasActiveDomain := tenant.CustomDomainStatus == "active" &&
			tenant.CustomDomain.Valid &&
			tenant.CustomDomain.String != ""

		isOnSubdomain := strings.HasSuffix(host, "."+BaseDomain)

		if hasActiveDomain && isOnSubdomain && !isProtectedRoute(r.URL.Path) {
			// Build redirect URL
			scheme := "https" // Always HTTPS for custom domains
			redirectURL := scheme + "://" + tenant.CustomDomain.String + r.URL.Path
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

// extractSubdomain extracts the subdomain from a host.
// Input: "roastercoffee.freyja.app", "freyja.app"
// Output: "roastercoffee" or "" if no subdomain
func extractSubdomain(host, baseDomain string) string {
	// Check if host ends with .baseDomain
	suffix := "." + baseDomain
	if !strings.HasSuffix(host, suffix) {
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, suffix)
	if subdomain == "" || strings.Contains(subdomain, ".") {
		// Empty or nested subdomain (e.g., "sub.tenant.freyja.app")
		return ""
	}

	return subdomain
}

// isProtectedRoute returns true if the path should NOT redirect.
// Protected routes remain accessible on both subdomain and custom domain.
func isProtectedRoute(path string) bool {
	protectedPrefixes := []string{
		"/admin",     // Admin dashboard (exact match or /admin/*)
		"/saas/",     // SaaS onboarding and billing
		"/api/",      // API endpoints (including Caddy validation)
		"/webhooks/", // Webhook handlers (Stripe, etc.)
		"/login",     // Auth routes
		"/logout",
		"/register",
		"/forgot-password",
		"/reset-password",
	}

	for _, prefix := range protectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
