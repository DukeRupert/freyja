package api

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/freyja/internal/service"
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
	// TODO: Implement ValidateDomain
	//
	// Implementation steps:
	// 1. Parse query parameter:
	//    - domain := r.URL.Query().Get("domain")
	//    - If domain is empty: return 400 Bad Request
	//
	// 2. Validate domain:
	//    - isValid, err := h.service.ValidateDomainForCaddy(r.Context(), domain)
	//    - If err != nil:
	//      - Log error: h.logger.Error("domain validation failed", "domain", domain, "error", err)
	//      - Return 500 Internal Server Error (Caddy will retry)
	//
	// 3. Return response:
	//    - If isValid:
	//      - w.WriteHeader(http.StatusOK)
	//      - w.Write([]byte("OK"))
	//      - Log: h.logger.Info("domain validation succeeded", "domain", domain)
	//    - If !isValid:
	//      - w.WriteHeader(http.StatusNotFound)
	//      - w.Write([]byte("Not Found"))
	//      - Log: h.logger.Debug("domain validation rejected", "domain", domain)
	//
	// 4. Record telemetry:
	//    - telemetry.Business.CaddyValidationRequests.WithLabelValues(status).Inc()
	//    - telemetry.Business.CaddyValidationLatency.Observe(duration)
	//
	// Caddy behavior:
	// - On 200: Proceed with ACME challenge, issue certificate
	// - On 404: Reject request, do not issue certificate
	// - On 500: Retry after delay (up to 3 times)
	//
	// Why this prevents abuse:
	// - Without this check, anyone could request a certificate for any domain
	// - Caddy would attempt ACME challenge for attacker-controlled domains
	// - This endpoint ensures only verified, active domains get certificates

	http.Error(w, "Not implemented", http.StatusNotImplemented)
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
