package storefront

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/hiri/internal/cookie"
)

// Test GetSessionIDFromCookie
func TestGetSessionIDFromCookie(t *testing.T) {
	tests := []struct {
		name           string
		cookieName     string
		cookieValue    string
		expectedResult string
	}{
		{
			name:           "returns session ID when cookie exists",
			cookieName:     "hiri_session",
			cookieValue:    "valid-session-id-12345",
			expectedResult: "valid-session-id-12345",
		},
		{
			name:           "returns empty string when cookie does not exist",
			cookieName:     "other_cookie",
			cookieValue:    "some-value",
			expectedResult: "",
		},
		{
			name:           "returns empty string when no cookies present",
			expectedResult: "",
		},
		{
			name:           "handles UUID-formatted session IDs",
			cookieName:     "hiri_session",
			cookieValue:    "123e4567-e89b-12d3-a456-426614174000",
			expectedResult: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:           "handles empty cookie value",
			cookieName:     "hiri_session",
			cookieValue:    "",
			expectedResult: "",
		},
		{
			name:           "handles long session IDs",
			cookieName:     "hiri_session",
			cookieValue:    "very-long-session-id-with-lots-of-characters-12345678901234567890",
			expectedResult: "very-long-session-id-with-lots-of-characters-12345678901234567890",
		},
		{
			name:           "handles session IDs with special characters",
			cookieName:     "hiri_session",
			cookieValue:    "session_id-with.special+chars",
			expectedResult: "session_id-with.special+chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.cookieName != "" {
				req.AddCookie(&http.Cookie{
					Name:  tt.cookieName,
					Value: tt.cookieValue,
				})
			}

			result := GetSessionIDFromCookie(req)

			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// Test SetSessionCookie
func TestSetSessionCookie(t *testing.T) {
	tests := []struct {
		name             string
		sessionID        string
		secure           bool
		expectedName     string
		expectedValue    string
		expectedPath     string
		expectedMaxAge   int
		expectedHttpOnly bool
		expectedSecure   bool
		expectedSameSite http.SameSite
	}{
		{
			name:             "sets cookie with correct attributes in insecure mode",
			sessionID:        "test-session-id-12345",
			secure:           false,
			expectedName:     "hiri_session",
			expectedValue:    "test-session-id-12345",
			expectedPath:     "/",
			expectedMaxAge:   30 * 24 * 60 * 60, // 30 days
			expectedHttpOnly: true,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "sets cookie with correct attributes in secure mode",
			sessionID:        "test-session-id-67890",
			secure:           true,
			expectedName:     "hiri_session",
			expectedValue:    "test-session-id-67890",
			expectedPath:     "/",
			expectedMaxAge:   30 * 24 * 60 * 60,
			expectedHttpOnly: true,
			expectedSecure:   true,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "handles UUID session IDs",
			sessionID:        "123e4567-e89b-12d3-a456-426614174000",
			secure:           false,
			expectedName:     "hiri_session",
			expectedValue:    "123e4567-e89b-12d3-a456-426614174000",
			expectedPath:     "/",
			expectedMaxAge:   30 * 24 * 60 * 60,
			expectedHttpOnly: true,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "handles empty session ID",
			sessionID:        "",
			secure:           false,
			expectedName:     "hiri_session",
			expectedValue:    "",
			expectedPath:     "/",
			expectedMaxAge:   30 * 24 * 60 * 60,
			expectedHttpOnly: true,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "handles long session IDs",
			sessionID:        "very-long-session-id-with-many-characters-to-test-edge-cases-12345678901234567890",
			secure:           true,
			expectedName:     "hiri_session",
			expectedValue:    "very-long-session-id-with-many-characters-to-test-edge-cases-12345678901234567890",
			expectedPath:     "/",
			expectedMaxAge:   30 * 24 * 60 * 60,
			expectedHttpOnly: true,
			expectedSecure:   true,
			expectedSameSite: http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cookieConfig := cookie.NewConfig("test.local", tt.secure)

			SetSessionCookie(w, tt.sessionID, cookieConfig)

			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			c := cookies[0]

			if c.Name != tt.expectedName {
				t.Errorf("expected Name %q, got %q", tt.expectedName, c.Name)
			}

			if c.Value != tt.expectedValue {
				t.Errorf("expected Value %q, got %q", tt.expectedValue, c.Value)
			}

			if c.Path != tt.expectedPath {
				t.Errorf("expected Path %q, got %q", tt.expectedPath, c.Path)
			}

			if c.MaxAge != tt.expectedMaxAge {
				t.Errorf("expected MaxAge %d, got %d", tt.expectedMaxAge, c.MaxAge)
			}

			if c.HttpOnly != tt.expectedHttpOnly {
				t.Errorf("expected HttpOnly %v, got %v", tt.expectedHttpOnly, c.HttpOnly)
			}

			if c.Secure != tt.expectedSecure {
				t.Errorf("expected Secure %v, got %v", tt.expectedSecure, c.Secure)
			}

			if c.SameSite != tt.expectedSameSite {
				t.Errorf("expected SameSite %v, got %v", tt.expectedSameSite, c.SameSite)
			}
		})
	}
}

// Test cookie expiration calculation
func TestSetSessionCookie_ExpirationTime(t *testing.T) {
	w := httptest.NewRecorder()
	sessionID := "test-session"
	cookieConfig := cookie.NewConfig("test.local", false)

	SetSessionCookie(w, sessionID, cookieConfig)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]

	// MaxAge should be 30 days in seconds
	expectedMaxAge := 30 * 24 * 60 * 60
	if c.MaxAge != expectedMaxAge {
		t.Errorf("expected MaxAge %d seconds (30 days), got %d", expectedMaxAge, c.MaxAge)
	}

	// Note: In httptest.ResponseRecorder, the Expires field is not automatically
	// calculated from MaxAge. This is a limitation of the test infrastructure, not
	// the actual handler. In a real HTTP response, the browser will calculate the
	// expiration time based on MaxAge. We verify MaxAge is set correctly, which is
	// what matters for production use.
}

// Test round-trip: set cookie and then read it
func TestSessionCookie_RoundTrip(t *testing.T) {
	sessionID := "round-trip-session-12345"
	cookieConfig := cookie.NewConfig("test.local", false)

	// Set the cookie
	w := httptest.NewRecorder()
	SetSessionCookie(w, sessionID, cookieConfig)

	// Extract the Set-Cookie header
	setCookieHeader := w.Header().Get("Set-Cookie")
	if setCookieHeader == "" {
		t.Fatal("expected Set-Cookie header to be set")
	}

	// Create a new request with the cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", setCookieHeader)

	// Read the cookie back
	retrievedSessionID := GetSessionIDFromCookie(req)

	if retrievedSessionID != sessionID {
		t.Errorf("expected session ID %q, got %q", sessionID, retrievedSessionID)
	}
}

// Test cookie security attributes for production use
func TestSetSessionCookie_SecurityAttributes(t *testing.T) {
	tests := []struct {
		name        string
		secure      bool
		description string
	}{
		{
			name:        "development mode (HTTP)",
			secure:      false,
			description: "allows testing over HTTP without HTTPS",
		},
		{
			name:        "production mode (HTTPS)",
			secure:      true,
			description: "enforces HTTPS-only cookie transmission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cookieConfig := cookie.NewConfig("test.local", tt.secure)
			SetSessionCookie(w, "test-session", cookieConfig)

			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			c := cookies[0]

			// HttpOnly must ALWAYS be true (prevents XSS attacks)
			if !c.HttpOnly {
				t.Error("HttpOnly must be true to prevent XSS attacks")
			}

			// SameSite must be Lax or Strict (prevents CSRF attacks)
			if c.SameSite != http.SameSiteLaxMode && c.SameSite != http.SameSiteStrictMode {
				t.Errorf("SameSite should be Lax or Strict for CSRF protection, got %v", c.SameSite)
			}

			// Secure should match the parameter
			if c.Secure != tt.secure {
				t.Errorf("expected Secure=%v, got %v", tt.secure, c.Secure)
			}

			// Path should be root to apply to entire application
			if c.Path != "/" {
				t.Error("Path should be '/' to apply cookie to entire application")
			}

			// MaxAge should be set (not relying solely on Expires)
			if c.MaxAge <= 0 {
				t.Error("MaxAge should be positive for persistent session")
			}
		})
	}
}

// Test handling multiple cookies in request
func TestGetSessionIDFromCookie_MultipleCookies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Add multiple cookies, including the session cookie
	req.AddCookie(&http.Cookie{
		Name:  "other_cookie_1",
		Value: "other_value_1",
	})
	req.AddCookie(&http.Cookie{
		Name:  "hiri_session",
		Value: "correct-session-id",
	})
	req.AddCookie(&http.Cookie{
		Name:  "other_cookie_2",
		Value: "other_value_2",
	})

	result := GetSessionIDFromCookie(req)

	if result != "correct-session-id" {
		t.Errorf("expected 'correct-session-id', got %q", result)
	}
}
