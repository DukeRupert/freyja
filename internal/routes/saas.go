package routes

import (
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
}
