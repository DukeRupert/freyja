package routes

import (
	"github.com/dukerupert/hiri/internal/router"
)

// RegisterWebhookRoutes registers all webhook routes.
// These routes handle incoming webhooks from external services.
//
// Note: Webhook routes do NOT have authentication middleware.
// Each webhook handler is responsible for verifying the request
// signature (e.g., Stripe signature verification).
func RegisterWebhookRoutes(r *router.Router, deps WebhookDeps) {
	r.Post("/webhooks/stripe", deps.StripeHandler)
}
