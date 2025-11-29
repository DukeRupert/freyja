package storefront

import (
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/middleware"
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

	return data
}
