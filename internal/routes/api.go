package routes

import (
	"github.com/dukerupert/freyja/internal/router"
)

// RegisterAPIRoutes registers API routes for external services (e.g., Caddy)
// These routes do not require authentication and are called by external systems
func RegisterAPIRoutes(r *router.Router, deps APIDeps) {
	// Domain validation for Caddy on-demand TLS
	r.Get("/api/validate-domain", deps.DomainValidationHandler.ValidateDomain)
}
