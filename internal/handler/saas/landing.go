package saas

import (
	"html/template"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/handler"
)

// LandingHandler handles the SaaS marketing landing page
type LandingHandler struct {
	tmpl *template.Template
}

// LandingData holds data for the landing page template
type LandingData struct {
	Year int
}

// NewLandingHandler creates a new landing page handler
func NewLandingHandler(templatesDir string) (*LandingHandler, error) {
	tmpl, err := template.New("landing.html").Funcs(handler.TemplateFuncs()).ParseFiles(
		templatesDir + "/saas/landing.html",
	)
	if err != nil {
		return nil, err
	}

	return &LandingHandler{
		tmpl: tmpl,
	}, nil
}

// ServeHTTP handles GET requests for the landing page
func (h *LandingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := LandingData{
		Year: time.Now().Year(),
	}

	if err := h.tmpl.ExecuteTemplate(w, "saas-landing", data); err != nil {
		http.Error(w, "Failed to render landing page", http.StatusInternalServerError)
		return
	}
}
