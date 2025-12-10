package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/hiri/internal/cookie"
)

const (
	// CSRFTokenLength is the length of the CSRF token in bytes
	CSRFTokenLength = 32

	// CSRFCookieName is the name of the CSRF cookie
	CSRFCookieName = "csrf_token"

	// CSRFHeaderName is the header name for CSRF token
	CSRFHeaderName = "X-CSRF-Token"

	// CSRFFormFieldName is the form field name for CSRF token
	CSRFFormFieldName = "csrf_token"

	// CSRFContextKey is the context key for the CSRF token
	CSRFContextKey contextKey = "csrf_token"
)

// CSRFConfig configures CSRF protection
type CSRFConfig struct {
	// CookieConfig is the cookie configuration for domain-scoped cookies
	CookieConfig *cookie.Config

	// CookieName is the name of the CSRF cookie
	// Default: "csrf_token"
	CookieName string

	// CookieMaxAge is the max age of the CSRF cookie in seconds
	// Default: 86400 (24 hours)
	CookieMaxAge int

	// SkipPaths are paths that should skip CSRF validation
	// Useful for webhooks that have their own authentication
	SkipPaths []string

	// ErrorHandler is called when CSRF validation fails
	// Default: returns 403 Forbidden
	ErrorHandler func(w http.ResponseWriter, r *http.Request)
}

// DefaultCSRFConfig returns sensible defaults.
// Requires a cookie.Config to be provided for domain scoping.
func DefaultCSRFConfig(cookieConfig *cookie.Config) CSRFConfig {
	return CSRFConfig{
		CookieConfig: cookieConfig,
		CookieName:   CSRFCookieName,
		CookieMaxAge: 86400, // 24 hours
		SkipPaths:    []string{"/webhooks/"},
		ErrorHandler: nil,
	}
}

// CSRF provides CSRF protection middleware.
// Requires CSRFConfig with a valid cookie.Config.
func CSRF(cfg CSRFConfig) func(http.Handler) http.Handler {
	// Validate required config
	if cfg.CookieConfig == nil {
		panic("csrf: CookieConfig is required")
	}

	// Fill in missing values with defaults
	if cfg.CookieName == "" {
		cfg.CookieName = CSRFCookieName
	}
	if cfg.CookieMaxAge == 0 {
		cfg.CookieMaxAge = 86400 // 24 hours
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip CSRF validation
			// SECURITY: Use proper path boundary matching to prevent bypass
			// e.g., /webhooks/ should not match /webhooks-evil/
			for _, skipPath := range cfg.SkipPaths {
				if matchesPathPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get or create CSRF token
			token := getCSRFTokenFromCookie(r, cfg.CookieName)
			if token == "" {
				var err error
				token, err = generateCSRFToken()
				if err != nil {
					// SECURITY: Fail closed if we can't generate secure token
					slog.Error("csrf: failed to generate secure token", "error", err)
					respondInternalError(w, r, err)
					return
				}
				setCSRFCookie(w, token, cfg)
			}

			// Add token to context for templates
			ctx := context.WithValue(r.Context(), CSRFContextKey, token)
			r = r.WithContext(ctx)

			// For safe methods, just continue
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// For unsafe methods, validate the token
			submittedToken := getSubmittedCSRFToken(r)
			if !validateCSRFToken(token, submittedToken) {
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(w, r)
				} else {
					respondForbidden(w, r)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetCSRFToken retrieves the CSRF token from the request context
// Use this in templates: <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
func GetCSRFToken(ctx context.Context) string {
	if token, ok := ctx.Value(CSRFContextKey).(string); ok {
		return token
	}
	return ""
}

// generateCSRFToken creates a new random CSRF token.
// Returns an error if secure random generation fails - we fail closed
// rather than using a weak fallback that could be exploited.
func generateCSRFToken() (string, error) {
	b := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(b); err != nil {
		// SECURITY: Fail closed - don't use predictable fallback
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// getCSRFTokenFromCookie retrieves the CSRF token from the cookie
func getCSRFTokenFromCookie(r *http.Request, cookieName string) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// setCSRFCookie sets the CSRF token cookie using the cookie config.
// Note: CSRF tokens need to be readable by JavaScript for htmx, so we use a custom cookie
// instead of the cookie.Config.SetSession which sets HttpOnly=true.
func setCSRFCookie(w http.ResponseWriter, token string, config CSRFConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Domain:   "." + config.CookieConfig.BaseDomain,
		Path:     "/",
		MaxAge:   config.CookieMaxAge,
		Secure:   config.CookieConfig.Secure,
		HttpOnly: false, // JS needs to read for htmx
		SameSite: http.SameSiteLaxMode,
	})
}

// getSubmittedCSRFToken retrieves the submitted CSRF token from header or form
func getSubmittedCSRFToken(r *http.Request) string {
	// Check header first (for AJAX/htmx requests)
	if token := r.Header.Get(CSRFHeaderName); token != "" {
		return token
	}

	// Check content type to determine how to parse the form
	contentType := r.Header.Get("Content-Type")

	// Handle multipart forms (file uploads)
	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		// Parse multipart form with 32MB max memory
		if err := r.ParseMultipartForm(32 << 20); err == nil {
			if token := r.FormValue(CSRFFormFieldName); token != "" {
				return token
			}
		}
		return ""
	}

	// Handle regular form submissions
	if err := r.ParseForm(); err == nil {
		if token := r.FormValue(CSRFFormFieldName); token != "" {
			return token
		}
	}

	return ""
}

// validateCSRFToken validates the submitted token against the cookie token
func validateCSRFToken(cookieToken, submittedToken string) bool {
	if cookieToken == "" || submittedToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(submittedToken)) == 1
}

// isSafeMethod returns true for HTTP methods that don't change state
func isSafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions ||
		method == http.MethodTrace
}

// matchesPathPrefix checks if requestPath matches the skipPath with proper boundary checking.
// This prevents bypass attacks where /webhooks/ would incorrectly match /webhooks-evil/.
// The skipPath must be an exact prefix with a path boundary (/ or end of string).
func matchesPathPrefix(requestPath, skipPath string) bool {
	if !strings.HasPrefix(requestPath, skipPath) {
		return false
	}

	// If skipPath ends with /, it already has a proper boundary
	if strings.HasSuffix(skipPath, "/") {
		return true
	}

	// For paths without trailing slash, check boundary
	// requestPath must be exactly skipPath, or have / after it
	if len(requestPath) == len(skipPath) {
		return true // Exact match
	}

	// Check that the character after skipPath is a path separator
	return requestPath[len(skipPath)] == '/'
}
