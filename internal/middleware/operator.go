package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/hiri/internal/cookie"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/service"
)

const (
	// OperatorContextKey is the context key for storing the authenticated operator
	OperatorContextKey contextKey = "operator"

	// TenantContextKey is the context key for storing the operator's tenant
	TenantContextKey contextKey = "tenant"

	// TenantIDContextKey is the context key for storing the tenant's UUID
	TenantIDContextKey contextKey = "tenant_id"

	// operatorCookieName matches the constant in handler/saas/auth.go
	operatorCookieName = "freyja_operator"
)

// WithOperator extracts the operator from the session cookie and adds them to the request context.
// This middleware is optional - it adds the operator if present but doesn't require authentication.
func WithOperator(operatorService service.OperatorService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session cookie
			cookie, err := r.Cookie(operatorCookieName)
			if err != nil || cookie.Value == "" {
				// No session cookie, continue without operator
				next.ServeHTTP(w, r)
				return
			}

			// Get operator from session
			operator, err := operatorService.GetOperatorBySessionToken(r.Context(), cookie.Value)
			if err != nil {
				// Log auth errors for debugging (except expected ones)
				if !errors.Is(err, service.ErrOperatorNotFound) &&
					!errors.Is(err, service.ErrOperatorInvalidToken) {
					slog.Warn("operator auth: failed to get operator from session",
						"error", err,
						"path", r.URL.Path,
					)
				}
				// Invalid session, continue without operator
				next.ServeHTTP(w, r)
				return
			}

			// Add operator to context
			ctx := context.WithValue(r.Context(), OperatorContextKey, operator)

			// Also add tenant_id to context for convenience
			tenantID := convertPgUUIDToUUID(operator.TenantID)
			ctx = context.WithValue(ctx, TenantIDContextKey, tenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireOperator ensures the operator is authenticated, redirecting to login if not.
// Must be used after WithOperator middleware.
func RequireOperator(cookieConfig *cookie.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			operator := GetOperatorFromContext(r.Context())
			if operator == nil {
				// Not authenticated, redirect to login with return URL
				returnTo := r.URL.Path
				if r.URL.RawQuery != "" {
					returnTo += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, "/login?return_to="+url.QueryEscape(returnTo), http.StatusSeeOther)
				return
			}

			// Check if operator is active
			if operator.Status != "active" {
				slog.Warn("operator auth: operator not active",
					"operator_id", operator.ID,
					"status", operator.Status,
				)
				// Clear cookie and redirect to login
				cookieConfig.ClearSession(w, operatorCookieName)
				http.Redirect(w, r, "/login?error=account_inactive", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireActiveTenant ensures the operator's tenant subscription is active.
// Must be used after RequireOperator middleware.
func RequireActiveTenant(queries *repository.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			operator := GetOperatorFromContext(r.Context())
			if operator == nil {
				// Should not happen if RequireOperator was used first
				respondUnauthorized(w, r)
				return
			}

			// Get tenant to check subscription status
			tenant, err := queries.GetTenantByID(r.Context(), operator.TenantID)
			if err != nil {
				slog.Error("operator auth: failed to get tenant",
					"tenant_id", operator.TenantID,
					"error", err,
				)
				respondInternalError(w, r, err)
				return
			}

			// Add tenant to context
			ctx := context.WithValue(r.Context(), TenantContextKey, &tenant)

			// Check tenant status
			switch tenant.Status {
			case "active", "pending":
				// Active or setting up - allow access
				next.ServeHTTP(w, r.WithContext(ctx))

			case "past_due":
				// Payment failed but in grace period - allow access with warning
				// The UI should show a banner about payment issues
				next.ServeHTTP(w, r.WithContext(ctx))

			case "suspended":
				// Grace period expired - redirect to billing page
				slog.Warn("operator auth: tenant suspended",
					"tenant_id", operator.TenantID,
				)
				http.Redirect(w, r, "/admin/billing?status=suspended", http.StatusSeeOther)
				return

			case "cancelled":
				// Subscription cancelled - redirect to reactivation page
				slog.Info("operator auth: tenant cancelled",
					"tenant_id", operator.TenantID,
				)
				http.Redirect(w, r, "/admin/billing?status=cancelled", http.StatusSeeOther)
				return

			default:
				// Unknown status - log and allow (fail open for unknown statuses)
				slog.Warn("operator auth: unknown tenant status",
					"tenant_id", operator.TenantID,
					"status", tenant.Status,
				)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		})
	}
}

// RequireOwner ensures the operator has the owner role.
// Must be used after RequireOperator middleware.
func RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operator := GetOperatorFromContext(r.Context())
		if operator == nil {
			respondUnauthorized(w, r)
			return
		}

		if operator.Role != "owner" {
			slog.Warn("operator auth: owner required",
				"operator_id", operator.ID,
				"role", operator.Role,
				"path", r.URL.Path,
			)
			respondForbidden(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetOperatorFromContext retrieves the operator from the request context.
// Returns nil if no operator is authenticated.
func GetOperatorFromContext(ctx context.Context) *repository.TenantOperator {
	operator, ok := ctx.Value(OperatorContextKey).(*repository.TenantOperator)
	if !ok {
		return nil
	}
	return operator
}

// GetOperatorID retrieves the operator ID from the request context.
// Returns uuid.Nil if no operator is authenticated.
func GetOperatorID(ctx context.Context) uuid.UUID {
	operator := GetOperatorFromContext(ctx)
	if operator == nil {
		return uuid.Nil
	}
	return convertPgUUIDToUUID(operator.ID)
}

// GetTenantIDFromOperator retrieves the tenant ID from the operator in the request context.
// Returns uuid.Nil if no operator is authenticated.
func GetTenantIDFromOperator(ctx context.Context) uuid.UUID {
	operator := GetOperatorFromContext(ctx)
	if operator == nil {
		return uuid.Nil
	}
	return convertPgUUIDToUUID(operator.TenantID)
}

// GetTenantFromContext retrieves the tenant from the request context.
// Only available after RequireActiveTenant middleware.
// Returns nil if no tenant is in context.
func GetTenantFromContext(ctx context.Context) *repository.Tenant {
	tenant, ok := ctx.Value(TenantContextKey).(*repository.Tenant)
	if !ok {
		return nil
	}
	return tenant
}

// convertPgUUIDToUUID converts a pgtype.UUID Bytes to uuid.UUID
func convertPgUUIDToUUID(pgUUID pgtype.UUID) uuid.UUID {
	return uuid.UUID(pgUUID.Bytes)
}
