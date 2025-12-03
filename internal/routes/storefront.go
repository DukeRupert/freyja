package routes

import (
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/router"
)

// RegisterStorefrontRoutes registers all customer-facing storefront routes.
// These routes are for the tenant's e-commerce storefront.
//
// When multi-tenancy is enabled, these routes will be served per-tenant
// based on subdomain or custom domain.
func RegisterStorefrontRoutes(r *router.Router, deps StorefrontDeps) {
	// Home page
	r.Get("/", deps.HomeHandler.ServeHTTP)

	// Product browsing
	r.Get("/products", deps.ProductListHandler.ServeHTTP)
	r.Get("/products/{slug}", deps.ProductDetailHandler.ServeHTTP)

	// Shopping cart
	r.Get("/cart", deps.CartViewHandler.ServeHTTP)
	r.Post("/cart/add", deps.AddToCartHandler.ServeHTTP)
	r.Post("/cart/update", deps.UpdateCartItemHandler.ServeHTTP)
	r.Post("/cart/remove", deps.RemoveCartItemHandler.ServeHTTP)

	// Authentication (GET routes only - POST routes registered separately with rate limiting)
	r.Get("/signup", deps.SignupHandler.ServeHTTP)
	r.Get("/login", deps.LoginHandler.ServeHTTP)
	r.Post("/logout", deps.LogoutHandler.ServeHTTP)

	// Password Reset
	r.Get("/forgot-password", deps.ForgotPasswordHandler.ServeHTTP)
	r.Post("/forgot-password", deps.ForgotPasswordHandler.ServeHTTP)
	r.Get("/reset-password", deps.ResetPasswordHandler.ServeHTTP)
	r.Post("/reset-password", deps.ResetPasswordHandler.ServeHTTP)

	// Checkout flow
	r.Get("/checkout", deps.CheckoutPageHandler.ServeHTTP)
	r.Post("/checkout/validate-address", deps.ValidateAddressHandler.ServeHTTP)
	r.Post("/checkout/shipping-rates", deps.GetShippingRatesHandler.ServeHTTP)
	r.Post("/checkout/calculate-total", deps.CalculateTotalHandler.ServeHTTP)
	r.Post("/checkout/create-payment-intent", deps.CreatePaymentIntentHandler.ServeHTTP)
	r.Get("/order-confirmation", deps.OrderConfirmationHandler.ServeHTTP)

	// Account routes (require authentication)
	account := r.Group(middleware.RequireAuth)
	account.Get("/account/subscriptions", deps.SubscriptionListHandler.ServeHTTP)
	account.Get("/account/subscriptions/portal", deps.SubscriptionPortalHandler.ServeHTTP)
	account.Get("/account/subscriptions/{id}", deps.SubscriptionDetailHandler.ServeHTTP)
	account.Get("/subscribe/checkout", deps.SubscriptionCheckoutHandler.ServeHTTP)
	account.Post("/subscribe", deps.CreateSubscriptionHandler.ServeHTTP)
}
