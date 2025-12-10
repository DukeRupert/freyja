package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/jackc/pgx/v5/pgtype"
)

// PageHandler handles admin pages management routes
type PageHandler struct {
	pageService domain.PageService
	renderer    *handler.Renderer
	tenantID    pgtype.UUID
}

// NewPageHandler creates a new pages handler
func NewPageHandler(pageService domain.PageService, renderer *handler.Renderer, tenantID string) *PageHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &PageHandler{
		pageService: pageService,
		renderer:    renderer,
		tenantID:    tenantUUID,
	}
}

// ListPage handles GET /admin/settings/pages
func (h *PageHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pages, err := h.pageService.ListPages(ctx, h.tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get all known page metadata for display
	allPageMeta := domain.GetPageMetadata()

	// Build a map of existing pages
	existingPages := make(map[string]domain.Page)
	for _, p := range pages {
		existingPages[p.Slug] = p
	}

	// Combine metadata with existing page data
	type DisplayPage struct {
		Slug        string
		Title       string
		Description string
		IsPublished bool
		Exists      bool
		UpdatedAt   string
	}

	var displayPages []DisplayPage
	for _, meta := range allPageMeta {
		dp := DisplayPage{
			Slug:        meta.Slug,
			Title:       meta.Title,
			Description: meta.Description,
			Exists:      false,
		}

		if existing, ok := existingPages[meta.Slug]; ok {
			dp.Exists = true
			dp.Title = existing.Title
			dp.IsPublished = existing.IsPublished
			dp.UpdatedAt = existing.UpdatedAt.Format("Jan 2, 2006")
		}

		displayPages = append(displayPages, dp)
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Pages":       displayPages,
	}

	h.renderer.RenderHTTP(w, "admin/pages", data)
}

// EditPage handles GET /admin/settings/pages/{slug}
func (h *PageHandler) EditPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if slug == "" || !domain.IsValidPageSlug(slug) {
		handler.ErrorResponse(w, r, domain.ErrInvalidSlug)
		return
	}

	page, err := h.pageService.GetPage(ctx, domain.GetPageParams{
		TenantID: h.tenantID,
		Slug:     slug,
	})

	// If page doesn't exist, create an empty one for editing
	var pageData map[string]interface{}
	if err != nil {
		if err == domain.ErrPageNotFound {
			// Get default metadata for this page type
			var title, description string
			for _, meta := range domain.GetPageMetadata() {
				if meta.Slug == slug {
					title = meta.Title
					description = meta.Description
					break
				}
			}

			pageData = map[string]interface{}{
				"CurrentPath":      r.URL.Path,
				"Slug":             slug,
				"Title":            title,
				"Content":          "",
				"MetaDescription":  description,
				"LastUpdatedLabel": "",
				"IsPublished":      true,
				"IsNew":            true,
			}
		} else {
			handler.InternalErrorResponse(w, r, err)
			return
		}
	} else {
		pageData = map[string]interface{}{
			"CurrentPath":      r.URL.Path,
			"Slug":             page.Slug,
			"Title":            page.Title,
			"Content":          page.Content,
			"MetaDescription":  page.MetaDescription,
			"LastUpdatedLabel": page.LastUpdatedLabel,
			"IsPublished":      page.IsPublished,
			"IsNew":            false,
		}
	}

	// Add CSRF token
	if csrfToken := middleware.GetCSRFToken(ctx); csrfToken != "" {
		pageData["CSRFToken"] = csrfToken
	}

	h.renderer.RenderHTTP(w, "admin/page_edit", pageData)
}

// UpdatePage handles POST /admin/settings/pages/{slug}
func (h *PageHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if slug == "" || !domain.IsValidPageSlug(slug) {
		handler.ErrorResponse(w, r, domain.ErrInvalidSlug)
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := r.FormValue("content") // Don't trim - preserve whitespace in HTML
	metaDescription := strings.TrimSpace(r.FormValue("meta_description"))
	lastUpdatedLabel := strings.TrimSpace(r.FormValue("last_updated_label"))
	isPublished := r.FormValue("is_published") == "on" || r.FormValue("is_published") == "true"

	if title == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Title is required"))
		return
	}

	// Check if page exists - if not, create it
	// UpdatePage handles upsert-like behavior, so we don't need to check if page exists first
	_, err := h.pageService.UpdatePage(ctx, domain.UpdatePageParams{
		TenantID:         h.tenantID,
		Slug:             slug,
		Title:            title,
		Content:          content,
		MetaDescription:  metaDescription,
		LastUpdatedLabel: lastUpdatedLabel,
		IsPublished:      isPublished,
	})
	if err != nil {
		if err == domain.ErrPageNotFound {
			// Page doesn't exist yet - show error asking to seed defaults first
			handler.ErrorResponse(w, r, domain.Errorf(domain.ENOTFOUND, "", "Page does not exist. Please initialize default pages first."))
			return
		}
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// If HTMX request, return success message
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "page-saved")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Page saved successfully"))
		return
	}

	http.Redirect(w, r, "/admin/settings/pages", http.StatusSeeOther)
}

// InitializePages handles POST /admin/settings/pages/initialize
// Creates default pages for tenant if they don't exist
func (h *PageHandler) InitializePages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Get store name and email from tenant config
	storeName := "Freyja Coffee"
	contactEmail := "support@freyjacoffee.com"

	err := h.pageService.EnsureDefaultPages(ctx, h.tenantID, storeName, contactEmail)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/settings/pages", http.StatusSeeOther)
}
