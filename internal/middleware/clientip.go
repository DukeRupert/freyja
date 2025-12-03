package middleware

import (
	"context"
	"net/http"
)

const (
	// ClientIPContextKey is the context key for storing the client IP address
	ClientIPContextKey contextKey = "client_ip"
)

// WithClientIP returns middleware that extracts the real client IP address from the request
// and stores it in the context. It uses the GetClientIP function from ratelimit.go which
// checks proxy headers (X-Forwarded-For, X-Real-IP) before falling back to RemoteAddr.
//
// This middleware should be placed early in the middleware chain so that handlers
// can access the client IP via GetClientIPFromContext.
//
// Note: In production, ensure your reverse proxy is configured to set these headers
// and that direct access to the application is not possible, as these headers can be spoofed.
func WithClientIP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use existing GetClientIP function from ratelimit.go
			clientIP := GetClientIP(r)

			// Store in context
			ctx := context.WithValue(r.Context(), ClientIPContextKey, clientIP)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClientIPFromContext retrieves the client IP address from the context.
// Returns an empty string if not found (middleware not applied).
// For direct access from request, use GetClientIP(r) instead.
func GetClientIPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value(ClientIPContextKey).(string); ok {
		return ip
	}
	return ""
}
