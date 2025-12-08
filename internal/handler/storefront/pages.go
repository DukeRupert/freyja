package storefront

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/jackc/pgx/v5/pgtype"
)

// PagesHandler handles static content pages (legal, about, contact, etc.)
type PagesHandler struct {
	pageService domain.PageService
	renderer    *handler.Renderer
	tenantID    pgtype.UUID
}

// NewPagesHandler creates a new pages handler
func NewPagesHandler(pageService domain.PageService, renderer *handler.Renderer, tenantID string) *PagesHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &PagesHandler{
		pageService: pageService,
		renderer:    renderer,
		tenantID:    tenantUUID,
	}
}

// PageData contains data for rendering a page from the database
type PageData struct {
	StoreName   string
	Title       string
	Content     template.HTML // Safe HTML from database
	LastUpdated string
	Year        int
	User        interface{}
	CartCount   int
	CSRFToken   string
}

// renderPage is a helper that fetches and renders a page by slug
func (h *PagesHandler) renderPage(w http.ResponseWriter, r *http.Request, slug string) {
	ctx := r.Context()

	page, err := h.pageService.GetPublishedPage(ctx, domain.GetPageParams{
		TenantID: h.tenantID,
		Slug:     slug,
	})

	if err != nil {
		if err == domain.ErrPageNotFound {
			handler.NotFoundResponse(w, r)
			return
		}
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := PageData{
		StoreName:   "Freyja Coffee", // TODO: Get from tenant config
		Title:       page.Title,
		Content:     template.HTML(page.Content), // Trust HTML from database (sanitized by Tiptap)
		LastUpdated: page.LastUpdatedLabel,
		Year:        time.Now().Year(),
		User:        middleware.GetUserFromContext(ctx),
		CSRFToken:   middleware.GetCSRFToken(ctx),
	}

	if cartCount, ok := ctx.Value("cart_count").(int); ok {
		data.CartCount = cartCount
	}

	h.renderer.RenderHTTP(w, "storefront/page", data)
}

// Privacy handles GET /privacy
func (h *PagesHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, domain.PageSlugPrivacy)
}

// Terms handles GET /terms
func (h *PagesHandler) Terms(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, domain.PageSlugTerms)
}

// Shipping handles GET /shipping
func (h *PagesHandler) Shipping(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, domain.PageSlugShipping)
}
