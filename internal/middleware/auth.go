package middleware

import (
	"context"
	"net/http"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
)

type contextKey string

const (
	// UserContextKey is the context key for storing the authenticated user
	UserContextKey contextKey = "user"

	sessionCookieName = "freyja_session"
)

// WithUser extracts the user from the session cookie and adds it to the request context
// This middleware is optional - it adds the user if present but doesn't require authentication
func WithUser(userService service.UserService) func(http.Handler) http.Handler {
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
			// Not authenticated, redirect to login
			returnTo := r.URL.Path
			if r.URL.RawQuery != "" {
				returnTo += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, "/login?return_to="+returnTo, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin ensures the user is an admin, returning 403 if not
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if user.AccountType != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user from the request context
// Returns nil if no user is authenticated
func GetUserFromContext(ctx context.Context) *repository.User {
	user, ok := ctx.Value(UserContextKey).(*repository.User)
	if !ok {
		return nil
	}
	return user
}
