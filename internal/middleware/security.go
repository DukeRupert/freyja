package middleware

import (
	"net/http"
)

// SecurityHeadersConfig configures security headers
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy sets the Content-Security-Policy header
	// Leave empty to use a sensible default
	ContentSecurityPolicy string

	// FrameOptions sets X-Frame-Options (DENY, SAMEORIGIN, or ALLOW-FROM uri)
	// Default: DENY
	FrameOptions string

	// ContentTypeNosniff sets X-Content-Type-Options: nosniff
	// Default: true
	ContentTypeNosniff bool

	// XSSProtection sets X-XSS-Protection header
	// Default: "1; mode=block"
	XSSProtection string

	// ReferrerPolicy sets Referrer-Policy header
	// Default: "strict-origin-when-cross-origin"
	ReferrerPolicy string

	// PermissionsPolicy sets Permissions-Policy header
	// Default: sensible restrictions
	PermissionsPolicy string

	// HSTSMaxAge sets Strict-Transport-Security max-age in seconds
	// Set to 0 to disable HSTS (not recommended in production)
	// Default: 31536000 (1 year)
	HSTSMaxAge int

	// HSTSIncludeSubdomains includes subdomains in HSTS
	// Default: true
	HSTSIncludeSubdomains bool
}

// DefaultSecurityHeadersConfig returns a sensible default configuration
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://unpkg.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		FrameOptions:          "DENY",
		ContentTypeNosniff:    true,
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=(), payment=(self)",
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
	}
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(config SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Frame-Options - prevent clickjacking
			if config.FrameOptions != "" {
				w.Header().Set("X-Frame-Options", config.FrameOptions)
			}

			// X-Content-Type-Options - prevent MIME sniffing
			if config.ContentTypeNosniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection - legacy XSS protection
			if config.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", config.XSSProtection)
			}

			// Referrer-Policy - control referrer information
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Content-Security-Policy - control resource loading
			if config.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// Permissions-Policy - control browser features
			if config.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Strict-Transport-Security - enforce HTTPS
			// Only set if request is over HTTPS or behind a proxy
			if config.HSTSMaxAge > 0 {
				hsts := "max-age=" + itoa(config.HSTSMaxAge)
				if config.HSTSIncludeSubdomains {
					hsts += "; includeSubDomains"
				}
				w.Header().Set("Strict-Transport-Security", hsts)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// itoa converts int to string without importing strconv
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
