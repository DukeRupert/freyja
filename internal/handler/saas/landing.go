package saas

import (
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/dukerupert/freyja/internal/handler"
)

// PageHandler handles SaaS marketing pages
type PageHandler struct {
	templates map[string]*template.Template
}

// PageData holds common data for SaaS page templates
type PageData struct {
	Year int
}

// NewPageHandler creates a new SaaS page handler
func NewPageHandler(templatesDir string) (*PageHandler, error) {
	templates := make(map[string]*template.Template)

	// Base layout file
	layoutFile := filepath.Join(templatesDir, "saas", "layout.html")

	// Page files to load
	pages := []string{
		"landing",
		"about",
		"contact",
		"privacy",
		"terms",
	}

	// Parse each page with the layout
	for _, page := range pages {
		pageFile := filepath.Join(templatesDir, "saas", page+".html")
		tmpl, err := template.New("layout.html").Funcs(handler.TemplateFuncs()).ParseFiles(layoutFile, pageFile)
		if err != nil {
			return nil, err
		}
		templates[page] = tmpl
	}

	return &PageHandler{
		templates: templates,
	}, nil
}

// ServePage serves a specific SaaS page
func (h *PageHandler) ServePage(pageName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, ok := h.templates[pageName]
		if !ok {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		data := PageData{
			Year: time.Now().Year(),
		}

		if err := tmpl.ExecuteTemplate(w, "saas-base", data); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
			return
		}
	}
}

// Landing returns a handler for the landing page
func (h *PageHandler) Landing() http.HandlerFunc {
	return h.ServePage("landing")
}

// About returns a handler for the about page
func (h *PageHandler) About() http.HandlerFunc {
	return h.ServePage("about")
}

// Contact returns a handler for the contact page
func (h *PageHandler) Contact() http.HandlerFunc {
	return h.ServePage("contact")
}

// Privacy returns a handler for the privacy policy page
func (h *PageHandler) Privacy() http.HandlerFunc {
	return h.ServePage("privacy")
}

// Terms returns a handler for the terms of service page
func (h *PageHandler) Terms() http.HandlerFunc {
	return h.ServePage("terms")
}
