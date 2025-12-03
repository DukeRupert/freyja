package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

const (
	// LoggerContextKey is the context key for storing the request-scoped logger
	LoggerContextKey contextKey = "logger"
)

// WithRequestLogger creates middleware that injects a request-scoped logger into the context.
// The logger includes request metadata (request_id, method, path) and user info if available.
// This middleware should be placed after RequestID and WithUser in the middleware chain.
func WithRequestLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Build logger with request context
			requestLogger := baseLogger.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)

			// Add request ID if available
			if requestID := GetRequestID(r.Context()); requestID != "" {
				requestLogger = requestLogger.With(slog.String("request_id", requestID))
			}

			// Add user ID if authenticated
			if user := GetUserFromContext(r.Context()); user != nil {
				requestLogger = requestLogger.With(slog.String("user_id", user.ID.String()))
			}

			// Store logger in context
			ctx := context.WithValue(r.Context(), LoggerContextKey, requestLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLogger retrieves the request-scoped logger from the context.
// If no logger is found, returns the provided fallback logger.
// If no fallback is provided, returns slog.Default().
func GetLogger(ctx context.Context, fallback ...*slog.Logger) *slog.Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*slog.Logger); ok {
		return logger
	}
	if len(fallback) > 0 && fallback[0] != nil {
		return fallback[0]
	}
	return slog.Default()
}
