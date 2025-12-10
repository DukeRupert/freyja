package api

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/service"
)

// DomainValidationHandler handles Caddy's on-demand TLS validation
type DomainValidationHandler struct {
	service service.CustomDomainService
	logger  *slog.Logger
}

// NewDomainValidationHandler creates a new domain validation handler
func NewDomainValidationHandler(
	service service.CustomDomainService,
	logger *slog.Logger,
) *DomainValidationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DomainValidationHandler{
		service: service,
		logger:  logger,
	}
}

// ValidateDomain handles GET /api/validate-domain?domain=example.com
// Called by Caddy before issuing a Let's Encrypt certificate
//
// Response codes:
// - 200 OK: Domain is valid and active, Caddy can issue certificate
// - 404 Not Found: Domain is not valid/active, Caddy should reject
// - 500 Internal Server Error: Database error (Caddy should retry)
//
// Query parameters:
// - domain: The domain to validate (e.g., "shop.example.com")
//
// Security considerations:
// - This endpoint is called by Caddy, not end users
// - No authentication required (rate limiting recommended)
// - Must be fast (< 5ms) to not delay certificate issuance
// - Prevents unauthorized certificate issuance for arbitrary domains
//
// Performance:
// - Uses idx_tenants_custom_domain_active index (partial index, very fast)
// - Single EXISTS query, no joins
// - Critical path for on-demand TLS
func (h *DomainValidationHandler) ValidateDomain(w http.ResponseWriter, r *http.Request) {
	domainParam := r.URL.Query().Get("domain")
	if domainParam == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "domain parameter required"))
		return
	}

	isValid, err := h.service.ValidateDomainForCaddy(r.Context(), domainParam)
	if err != nil {
		h.logger.Error("domain validation failed", "domain", domainParam, "error", err)
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if isValid {
		h.logger.Info("domain validation succeeded", "domain", domainParam)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	} else {
		h.logger.Debug("domain validation rejected", "domain", domainParam)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterRoutes registers domain validation routes
// Called from main.go or router setup
//
// Route:
//   GET /api/validate-domain - Caddy on-demand TLS validation
//
// Middleware:
// - Rate limiting recommended (100 req/min per IP)
// - No authentication required (Caddy calls this)
// - No CSRF protection (GET request, no side effects)
//
// Caddy configuration:
// In Caddyfile:
//   tls {
//     on_demand
//   }
//
// In /etc/caddy/caddy.json:
//   "ask": "http://freyja-app:3000/api/validate-domain"
func (h *DomainValidationHandler) RegisterRoutes(router interface{}) {
	// TODO: Implement route registration
	//
	// Example (adapt to actual router interface):
	// router.Get("/api/validate-domain", h.ValidateDomain)
	//
	// Note: This should be registered at the global router level,
	// not under /admin/* (Caddy needs public access)
}
