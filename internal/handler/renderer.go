package handler

import (
	"fmt"
	"html/template"
	"path/filepath"
)

// Renderer manages template parsing and rendering with isolated template sets
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer(templatesDir string) (*Renderer, error) {
	templates := make(map[string]*template.Template)

	// Parse layout once as base template
	layoutPattern := filepath.Join(templatesDir, "layout.html")
	baseTmpl, err := template.New("base").Funcs(TemplateFuncs()).ParseFiles(layoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layout: %w", err)
	}

	// Get list of page templates
	pages, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob templates: %w", err)
	}

	// Clone base template for each page
	for _, page := range pages {
		// Skip layout itself
		if filepath.Base(page) == "layout.html" {
			continue
		}

		// Clone the base template
		pageTmpl, err := baseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone template for %s: %w", page, err)
		}

		// Parse page-specific content into the clone
		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return nil, fmt.Errorf("failed to parse page %s: %w", page, err)
		}

		// Store with base name as key (without extension)
		pageName := filepath.Base(page)
		pageName = pageName[:len(pageName)-len(filepath.Ext(pageName))]
		templates[pageName] = pageTmpl
	}

	return &Renderer{
		templates: templates,
	}, nil
}

// Execute renders a named template with the given data
func (r *Renderer) Execute(name string) (*template.Template, error) {
	tmpl, ok := r.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return tmpl, nil
}
