package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryConfig holds configuration for Sentry error tracking
type SentryConfig struct {
	// DSN is the Sentry Data Source Name (required if Enabled is true)
	DSN string

	// Enabled controls whether Sentry is active
	// Set to false to disable during development or when DSN is not configured
	Enabled bool

	// Environment identifies the deployment environment (dev, staging, prod)
	Environment string

	// Release is the application version/release identifier
	Release string

	// SampleRate controls the percentage of errors to capture (0.0 to 1.0)
	// Default: 1.0 (capture all errors)
	SampleRate float64

	// TracesSampleRate controls the percentage of transactions to trace (0.0 to 1.0)
	// Set to 0 to disable performance monitoring
	TracesSampleRate float64

	// Debug enables Sentry SDK debug logging
	Debug bool
}

// SentryClient wraps Sentry functionality with enable/disable support
type SentryClient struct {
	enabled bool
	config  SentryConfig
}

// sentryInstance is the global Sentry client
var sentryInstance *SentryClient

// InitSentry initializes the Sentry client
// Returns a cleanup function that should be called on application shutdown
func InitSentry(cfg SentryConfig, logger *slog.Logger) (func(), error) {
	sentryInstance = &SentryClient{
		enabled: cfg.Enabled,
		config:  cfg,
	}

	if !cfg.Enabled {
		logger.Info("Sentry disabled (SENTRY_ENABLED=false or DSN not configured)")
		return func() {}, nil
	}

	if cfg.DSN == "" {
		logger.Warn("Sentry DSN not configured, disabling error tracking")
		sentryInstance.enabled = false
		return func() {}, nil
	}

	// Set defaults
	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 1.0
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       sampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// You can filter or modify events here
			// For example, filter out specific errors in development
			return event
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	logger.Info("Sentry initialized",
		"environment", cfg.Environment,
		"release", cfg.Release,
		"sample_rate", sampleRate,
		"traces_sample_rate", cfg.TracesSampleRate,
	)

	// Return cleanup function
	cleanup := func() {
		sentry.Flush(2 * time.Second)
	}

	return cleanup, nil
}

// IsEnabled returns whether Sentry is currently enabled
func IsEnabled() bool {
	if sentryInstance == nil {
		return false
	}
	return sentryInstance.enabled
}

// CaptureError captures an error with optional context
// Safe to call even when Sentry is disabled
func CaptureError(err error, ctx ...map[string]interface{}) {
	if !IsEnabled() || err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		if len(ctx) > 0 {
			for key, value := range ctx[0] {
				scope.SetExtra(key, value)
			}
		}
		sentry.CaptureException(err)
	})
}

// CaptureErrorWithTenant captures an error with tenant context
// This is the primary method for capturing errors in handlers
func CaptureErrorWithTenant(err error, tenantID string, extras map[string]interface{}) {
	if !IsEnabled() || err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("tenant_id", tenantID)
		for key, value := range extras {
			scope.SetExtra(key, value)
		}
		sentry.CaptureException(err)
	})
}

// CaptureMessage captures a message (non-error event)
func CaptureMessage(message string, level sentry.Level, ctx ...map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		if len(ctx) > 0 {
			for key, value := range ctx[0] {
				scope.SetExtra(key, value)
			}
		}
		sentry.CaptureMessage(message)
	})
}

// SetUser sets user context for subsequent error captures
func SetUser(id, email string) {
	if !IsEnabled() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:    id,
			Email: email,
		})
	})
}

// SetTenant sets tenant context for subsequent error captures
func SetTenant(tenantID string) {
	if !IsEnabled() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("tenant_id", tenantID)
	})
}

// AddBreadcrumb adds a breadcrumb for debugging
func AddBreadcrumb(category, message string, data map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: category,
		Message:  message,
		Data:     data,
		Level:    sentry.LevelInfo,
	})
}

// StartSpan starts a performance monitoring span
// Returns a finish function and a context with the span
func StartSpan(ctx context.Context, operation, description string) (context.Context, func()) {
	if !IsEnabled() {
		return ctx, func() {}
	}

	span := sentry.StartSpan(ctx, operation)
	span.Description = description

	return span.Context(), func() {
		span.Finish()
	}
}

// RecoverWithSentry recovers from panics and reports to Sentry
// Use: defer telemetry.RecoverWithSentry()
func RecoverWithSentry() {
	if r := recover(); r != nil {
		if IsEnabled() {
			sentry.CurrentHub().Recover(r)
			sentry.Flush(2 * time.Second)
		}
		// Re-panic after reporting
		panic(r)
	}
}

// SentryMiddleware returns an HTTP middleware that captures panics and adds request context
func SentryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			hub := sentry.GetHubFromContext(r.Context())
			if hub == nil {
				hub = sentry.CurrentHub().Clone()
			}

			hub.Scope().SetRequest(r)
			ctx := sentry.SetHubOnContext(r.Context(), hub)

			defer func() {
				if err := recover(); err != nil {
					hub.RecoverWithContext(ctx, err)
					sentry.Flush(2 * time.Second)
					// Return 500 after capturing the panic
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserInfo represents user information for Sentry context
type UserInfo struct {
	ID    string
	Email string
}

// UserContextExtractor is a function that extracts user info from a request context
type UserContextExtractor func(ctx context.Context) *UserInfo

// SentryContextMiddleware returns an HTTP middleware that sets tenant and user context
// on the Sentry hub for all errors captured during the request.
// This middleware should be applied after authentication middleware.
//
// Parameters:
//   - tenantID: The tenant ID to tag all errors with
//   - userExtractor: Optional function to extract user info from context (can be nil)
func SentryContextMiddleware(tenantID string, userExtractor UserContextExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			hub := sentry.GetHubFromContext(r.Context())
			if hub == nil {
				hub = sentry.CurrentHub().Clone()
			}

			// Set tenant context
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("tenant_id", tenantID)
				scope.SetContext("request", map[string]interface{}{
					"url":    r.URL.String(),
					"method": r.Method,
					"path":   r.URL.Path,
				})
			})

			// Set user context if extractor provided and user is authenticated
			if userExtractor != nil {
				if user := userExtractor(r.Context()); user != nil {
					hub.ConfigureScope(func(scope *sentry.Scope) {
						scope.SetUser(sentry.User{
							ID:    user.ID,
							Email: user.Email,
						})
					})
				}
			}

			ctx := sentry.SetHubOnContext(r.Context(), hub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CaptureErrorFromContext captures an error using the Sentry hub from the request context.
// This automatically includes tenant/user context set by SentryContextMiddleware.
// Use this in HTTP handlers instead of CaptureErrorWithTenant.
func CaptureErrorFromContext(ctx context.Context, err error, extras map[string]interface{}) {
	if !IsEnabled() || err == nil {
		return
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		// Fallback to global hub if context doesn't have one
		hub = sentry.CurrentHub()
	}

	hub.WithScope(func(scope *sentry.Scope) {
		for key, value := range extras {
			scope.SetExtra(key, value)
		}
		hub.CaptureException(err)
	})
}

// HTTPTransport wraps an http.RoundTripper to add Sentry tracing
type HTTPTransport struct {
	Transport http.RoundTripper
}

func (t *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !IsEnabled() {
		return t.Transport.RoundTrip(req)
	}

	span := sentry.StartSpan(req.Context(), "http.client")
	span.Description = fmt.Sprintf("%s %s", req.Method, req.URL.Host)
	defer span.Finish()

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
	} else {
		span.SetData("http.status_code", resp.StatusCode)
	}

	return resp, err
}
