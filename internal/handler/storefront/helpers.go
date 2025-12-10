package storefront

import (
	"net/http"
	"time"

	"github.com/dukerupert/hiri/internal/middleware"
)

// BaseTemplateData returns common data for all templates
func BaseTemplateData(r *http.Request) map[string]interface{} {
	data := map[string]interface{}{
		"Year": time.Now().Year(),
	}

	// Add user if authenticated
	user := middleware.GetUserFromContext(r.Context())
	if user != nil {
		data["User"] = user
	}

	// Add CSRF token for forms
	csrfToken := middleware.GetCSRFToken(r.Context())
	if csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	return data
}
