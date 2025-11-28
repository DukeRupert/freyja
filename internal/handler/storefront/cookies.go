package storefront

import "net/http"

// GetSessionIDFromCookie retrieves the session ID from the freyja_session cookie.
// Returns empty string if cookie is not present.
func GetSessionIDFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("freyja_session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetSessionCookie sets the freyja_session cookie with appropriate security settings.
func SetSessionCookie(w http.ResponseWriter, sessionID string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "freyja_session",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
