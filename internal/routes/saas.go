package routes

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/router"
)

// RegisterSaaSRoutes registers all SaaS marketing routes.
// These routes are for the Hiri platform marketing site itself,
// not for tenant storefronts.
//
// When ready to split, these routes can be served on a separate
// port or domain (e.g., hiri.coffee vs app.hiri.coffee).
func RegisterSaaSRoutes(r *router.Router, deps SaaSDeps) {
	r.Get("/", deps.Handler.Landing())
	r.Get("/pricing", deps.Handler.Pricing(deps.CheckoutURL))
	r.Get("/about", deps.Handler.About())
	r.Get("/contact", deps.Handler.Contact())
	r.Get("/privacy", deps.Handler.Privacy())
	r.Get("/terms", deps.Handler.Terms())

	// Redirect /signup to /pricing (the signup flow starts from pricing page checkout)
	r.Get("/signup", redirectToPricing())
}

// redirectToPricing returns a handler that redirects to the pricing page.
// This provides a clean /signup URL for marketing while the actual
// signup flow starts from the Stripe checkout on the pricing page.
func redirectToPricing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/pricing", http.StatusSeeOther)
	}
}
