package storefront

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/cookie"
)

// GetSessionIDFromCookie retrieves the session ID from the hiri_session cookie.
// Returns empty string if cookie is not present.
func GetSessionIDFromCookie(r *http.Request) string {
	return cookie.Get(r, "hiri_session")
}

// SetSessionCookie sets the hiri_session cookie with appropriate security settings.
// This function is deprecated - prefer using cookie.Config.SetSession directly.
func SetSessionCookie(w http.ResponseWriter, sessionID string, cookieConfig *cookie.Config) {
	cookieConfig.SetSession(w, "hiri_session", sessionID, 30*24*60*60)
}
