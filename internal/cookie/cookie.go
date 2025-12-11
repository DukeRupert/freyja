// Package cookie provides domain-aware cookie helpers for multi-tenant subdomain routing.
// All session and authentication cookies should use this package to ensure proper
// domain scoping across subdomains (e.g., acme.hiri.coffee, app.hiri.coffee).
package cookie

import (
	"net/http"
	"time"
)

// Config holds cookie configuration for domain-aware cookie operations.
type Config struct {
	// BaseDomain is the root domain for cookie scoping (e.g., "hiri.coffee" or "lvh.me").
	// Cookies will be set with Domain="." + BaseDomain to share across subdomains.
	BaseDomain string

	// Secure determines whether cookies require HTTPS.
	// Should be true in production, false in development.
	Secure bool
}

// NewConfig creates a new cookie configuration.
//
// Example:
//
//	cfg := cookie.NewConfig("hiri.coffee", true)  // production
//	cfg := cookie.NewConfig("lvh.me", false)      // development
func NewConfig(baseDomain string, secure bool) *Config {
	return &Config{
		BaseDomain: baseDomain,
		Secure:     secure,
	}
}

// SetSession sets a session cookie that is shared across all subdomains.
//
// The cookie will be set with:
//   - Domain: "." + BaseDomain (shared across subdomains)
//   - Path: "/" (available on all paths)
//   - HttpOnly: true (not accessible via JavaScript)
//   - SameSite: Lax (sent on top-level navigations and GET from third-party)
//   - Secure: based on config
func (c *Config) SetSession(w http.ResponseWriter, name, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   "." + c.BaseDomain,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSession removes a session cookie by setting MaxAge to -1.
func (c *Config) ClearSession(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Domain:   "." + c.BaseDomain,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// SetSessionWithExpiry sets a session cookie with an explicit expiration time.
// Use this when you need precise expiration control (e.g., "remember me" functionality).
func (c *Config) SetSessionWithExpiry(w http.ResponseWriter, name, value string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   "." + c.BaseDomain,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// Get retrieves a cookie value from the request.
// Returns empty string if cookie not found.
//
// This is a convenience wrapper around r.Cookie() that handles errors.
func Get(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Common cookie names used throughout the application.
// Using constants ensures consistency and makes refactoring easier.
const (
	// SessionCookieName is the main session cookie for authenticated users.
	SessionCookieName = "hiri_session"

	// CSRFCookieName stores the CSRF token for form protection.
	CSRFCookieName = "hiri_csrf"

	// CartCookieName stores the anonymous cart ID for guest users.
	CartCookieName = "hiri_cart"

	// FlashCookieName stores flash messages between redirects.
	FlashCookieName = "hiri_flash"
)
