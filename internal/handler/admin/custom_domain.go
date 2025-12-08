package admin

import (
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/service"
)

// CustomDomainHandler handles custom domain management routes
type CustomDomainHandler struct {
	service  service.CustomDomainService
	renderer *handler.Renderer
}

// NewCustomDomainHandler creates a new custom domain handler
func NewCustomDomainHandler(
	service service.CustomDomainService,
	renderer *handler.Renderer,
) *CustomDomainHandler {
	return &CustomDomainHandler{
		service:  service,
		renderer: renderer,
	}
}

// ShowDomainSettings handles GET /admin/settings/domain
// Displays the custom domain settings page with current status
func (h *CustomDomainHandler) ShowDomainSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantIDFromOperator(ctx)

	customDomain, err := h.service.GetDomainStatus(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"CurrentPath":  r.URL.Path,
		"CustomDomain": customDomain,
	}

	if csrfToken := middleware.GetCSRFToken(ctx); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	h.renderer.RenderHTTP(w, "admin/settings/custom_domain", data)
}

// InitiateDomain handles POST /admin/settings/domain
// Starts the custom domain setup process
//
// Form fields:
// - domain: string (e.g., "shop.example.com")
//
// Response:
// - Success: Redirect to domain settings page (shows DNS instructions)
// - Error: Re-render form with error message
//
// Errors handled:
// - ErrInvalidDomain: "Please enter a valid domain name"
// - ErrApexDomainNotAllowed: "Apex domains not supported. Use a subdomain like shop.example.com"
// - ErrDomainAlreadyInUse: "This domain is already in use by another store"
func (h *CustomDomainHandler) InitiateDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantIDFromOperator(ctx)

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	domainName := strings.TrimSpace(r.FormValue("domain"))

	_, err := h.service.InitiateVerification(ctx, tenantID, domainName)
	if err != nil {
		errCode := domain.ErrorCode(err)
		switch errCode {
		case domain.EINVALID:
			handler.ErrorResponse(w, r, err)
		case domain.ECONFLICT:
			handler.ErrorResponse(w, r, err)
		default:
			handler.InternalErrorResponse(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/admin/settings/domain", http.StatusSeeOther)
}

// VerifyDomain handles POST /admin/settings/domain/verify
// Checks DNS records and verifies the domain
//
// No form fields required (domain is already stored)
//
// Response:
// - Success (verified): Redirect to domain settings (shows activate button)
// - Success (still invalid): Redirect with error message about DNS not propagated
// - Error: Show error page
//
// Process:
// 1. Lookup CNAME and TXT records via DNS
// 2. If valid: mark domain as 'verified'
// 3. If invalid: mark as 'failed' with error message
func (h *CustomDomainHandler) VerifyDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantIDFromOperator(ctx)

	verification, err := h.service.CheckVerification(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if !verification.Verified {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "%s. DNS changes can take up to 48 hours to propagate.", verification.ErrorMessage))
		return
	}

	http.Redirect(w, r, "/admin/settings/domain", http.StatusSeeOther)
}

// ActivateDomain handles POST /admin/settings/domain/activate
// Activates a verified custom domain
//
// No form fields required
//
// Response:
// - Success: Redirect to domain settings (shows success message)
// - Error: Show error message
//
// After activation:
// - Caddy will provision SSL certificate on first request
// - Storefront redirects from subdomain to custom domain
// - Admin routes remain accessible on both domains
func (h *CustomDomainHandler) ActivateDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantIDFromOperator(ctx)

	err := h.service.ActivateDomain(ctx, tenantID)
	if err != nil {
		errCode := domain.ErrorCode(err)
		if errCode == domain.EINVALID || errCode == domain.EFORBIDDEN {
			handler.ErrorResponse(w, r, err)
		} else {
			handler.InternalErrorResponse(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/admin/settings/domain", http.StatusSeeOther)
}

// RemoveDomain handles DELETE /admin/settings/domain
// Removes the custom domain configuration
//
// No form fields required
//
// Response:
// - Success: Redirect to domain settings (shows empty state)
// - Error: Show error message
//
// After removal:
// - Storefront reverts to subdomain (*.freyja.app)
// - Custom domain stops serving traffic
// - SSL certificate remains cached in Caddy for 90 days (no manual cleanup)
func (h *CustomDomainHandler) RemoveDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantIDFromOperator(ctx)

	err := h.service.RemoveDomain(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/settings/domain", http.StatusSeeOther)
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterRoutes registers custom domain routes on the admin router
// Called from main.go or router setup
//
// Routes:
//   GET    /admin/settings/domain          - Show domain settings page
//   POST   /admin/settings/domain          - Initiate domain setup
//   POST   /admin/settings/domain/verify   - Verify DNS records
//   POST   /admin/settings/domain/activate - Activate verified domain
//   DELETE /admin/settings/domain          - Remove custom domain
//
// Middleware required:
// - WithOperator (load operator from session)
// - RequireOperator (ensure authenticated)
// - RequireActiveTenant (ensure tenant subscription is active)
// - CSRF protection (for POST/DELETE routes)
func (h *CustomDomainHandler) RegisterRoutes(router interface{}) {
	// TODO: Implement route registration
	//
	// Example (adapt to actual router interface):
	// router.Get("/admin/settings/domain", h.ShowDomainSettings)
	// router.Post("/admin/settings/domain", h.InitiateDomain)
	// router.Post("/admin/settings/domain/verify", h.VerifyDomain)
	// router.Post("/admin/settings/domain/activate", h.ActivateDomain)
	// router.Delete("/admin/settings/domain", h.RemoveDomain)
}
