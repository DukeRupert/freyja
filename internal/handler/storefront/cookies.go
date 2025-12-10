package storefront

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/cookie"
)

// GetSessionIDFromCookie retrieves the session ID from the freyja_session cookie.
// Returns empty string if cookie is not present.
func GetSessionIDFromCookie(r *http.Request) string {
	return cookie.Get(r, "freyja_session")
}

// SetSessionCookie sets the freyja_session cookie with appropriate security settings.
// This function is deprecated - prefer using cookie.Config.SetSession directly.
func SetSessionCookie(w http.ResponseWriter, sessionID string, cookieConfig *cookie.Config) {
	cookieConfig.SetSession(w, "freyja_session", sessionID, 30*24*60*60)
}
