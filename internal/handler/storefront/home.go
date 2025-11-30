package storefront

import (
	"html/template"
	"net/http"
)

// HomeHandler handles the homepage
type HomeHandler struct {
	templates *template.Template
}

// NewHomeHandler creates a new home handler
func NewHomeHandler(templates *template.Template) *HomeHandler {
	return &HomeHandler{
		templates: templates,
	}
}

// ServeHTTP handles GET /
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement homepage with featured products, hero section
	data := map[string]interface{}{
		"Year": 2024,
	}

	if err := h.templates.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
