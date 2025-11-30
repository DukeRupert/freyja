package handler

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Renderer manages template parsing and rendering with isolated template sets
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer(templatesDir string) (*Renderer, error) {
	templates := make(map[string]*template.Template)

	// Parse storefront layout once as base template
	layoutPattern := filepath.Join(templatesDir, "layout.html")
	baseTmpl, err := template.New("base").Funcs(TemplateFuncs()).ParseFiles(layoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layout: %w", err)
	}

	// Parse admin layout
	adminLayoutPattern := filepath.Join(templatesDir, "admin", "layout.html")
	adminBaseTmpl, err := template.New("admin_base").Funcs(TemplateFuncs()).ParseFiles(adminLayoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse admin layout: %w", err)
	}

	// Get list of page templates (root level)
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

	// Get admin templates
	adminPages, err := filepath.Glob(filepath.Join(templatesDir, "admin", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob admin templates: %w", err)
	}

	for _, page := range adminPages {
		// Skip admin layout itself
		if filepath.Base(page) == "layout.html" {
			continue
		}

		// Clone the admin base template
		pageTmpl, err := adminBaseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone template for %s: %w", page, err)
		}

		// Parse page-specific content into the clone
		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return nil, fmt.Errorf("failed to parse page %s: %w", page, err)
		}

		// Store with "admin/pagename" as key
		pageName := filepath.Base(page)
		pageName = pageName[:len(pageName)-len(filepath.Ext(pageName))]
		templates["admin/"+pageName] = pageTmpl
	}

	// Get storefront templates
	storefrontPages, err := filepath.Glob(filepath.Join(templatesDir, "storefront", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob storefront templates: %w", err)
	}

	for _, page := range storefrontPages {
		// Skip partial templates (those with "_" prefix or that don't define content blocks)
		baseName := filepath.Base(page)
		if baseName == "checkout_partials.html" {
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

		// Store with "storefront/pagename" as key
		pageName := baseName[:len(baseName)-len(filepath.Ext(baseName))]
		templates["storefront/"+pageName] = pageTmpl
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

// Render is a convenience method that executes a template and writes to an io.Writer
func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	tmpl, err := r.Execute(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// RenderHTTP is a convenience method that renders to an http.ResponseWriter with error handling
func (r *Renderer) RenderHTTP(w http.ResponseWriter, name string, data interface{}) {
	tmpl, err := r.Execute(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Determine which base template to execute based on the template name
	var execName string
	if len(name) >= 6 && name[:6] == "admin/" {
		execName = "admin_base"
	} else if len(name) >= 11 && name[:11] == "storefront/" {
		execName = "base"
	} else {
		execName = "base"
	}

	if err := tmpl.ExecuteTemplate(w, execName, data); err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
