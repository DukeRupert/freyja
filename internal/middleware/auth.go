package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dukerupert/hiri/internal/domain"
)

type contextKey string

const (
	// UserContextKey is the context key for storing the authenticated user
	UserContextKey contextKey = "user"

	sessionCookieName = "hiri_session"
)

// WithUser extracts the user from the session cookie and adds it to the request context
// This middleware is optional - it adds the user if present but doesn't require authentication
func WithUser(userService domain.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session cookie
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				// No session cookie, continue without user
				next.ServeHTTP(w, r)
				return
			}

			// Get user from session
			user, err := userService.GetUserBySessionToken(r.Context(), cookie.Value)
			if err != nil {
				// Log auth errors for debugging (except expected ones like expired sessions)
				if domain.ErrorCode(err) != domain.EUNAUTHORIZED && domain.ErrorCode(err) != domain.ENOTFOUND {
					slog.Warn("auth: failed to get user from session",
						"error", err,
						"path", r.URL.Path,
					)
				}
				// Invalid session, continue without user
				next.ServeHTTP(w, r)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth ensures the user is authenticated, redirecting to login if not
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			// Not authenticated, redirect to login with return URL
			returnTo := r.URL.Path
			if r.URL.RawQuery != "" {
				returnTo += "?" + r.URL.RawQuery
			}
			// URL-encode to handle special characters safely
			http.Redirect(w, r, "/login?return_to="+url.QueryEscape(returnTo), http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin ensures the user is an admin, redirecting to admin login if not
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		if user.AccountType != domain.UserAccountTypeAdmin {
			// User is authenticated but not an admin - redirect to storefront
			// This is friendlier than showing a 403 error
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user from the request context
// Returns nil if no user is authenticated
func GetUserFromContext(ctx context.Context) *domain.Customer {
	user, ok := ctx.Value(UserContextKey).(*domain.Customer)
	if !ok {
		return nil
	}
	return user
}
