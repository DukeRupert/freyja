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
	r.Get("/cart", deps.CartHandler.View)
	r.Post("/cart/add", deps.CartHandler.Add)
	r.Post("/cart/update", deps.CartHandler.Update)
	r.Post("/cart/remove", deps.CartHandler.Remove)

	// Authentication (GET routes only - POST routes registered separately with rate limiting)
	r.Get("/signup", deps.SignupHandler.ShowForm)
	r.Get("/signup-success", deps.SignupSuccessHandler.ServeHTTP)
	r.Get("/login", deps.LoginHandler.ShowForm)
	r.Post("/logout", deps.LogoutHandler.HandleSubmit)

	// Password Reset
	r.Get("/forgot-password", deps.ForgotPasswordHandler.ShowForm)
	r.Post("/forgot-password", deps.ForgotPasswordHandler.HandleSubmit)
	r.Get("/reset-password", deps.ResetPasswordHandler.ShowForm)
	r.Post("/reset-password", deps.ResetPasswordHandler.HandleSubmit)

	// Email Verification
	r.Get("/verify-email", deps.VerifyEmailHandler.HandleVerify)
	r.Get("/resend-verification", deps.ResendVerificationHandler.ShowForm)
	r.Post("/resend-verification", deps.ResendVerificationHandler.HandleSubmit)

	// Checkout flow
	r.Get("/checkout", deps.CheckoutHandler.Page)
	r.Post("/checkout/validate-address", deps.CheckoutHandler.ValidateAddress)
	r.Post("/checkout/shipping-rates", deps.CheckoutHandler.GetShippingRates)
	r.Post("/checkout/calculate-total", deps.CheckoutHandler.CalculateTotal)
	r.Post("/checkout/create-payment-intent", deps.CheckoutHandler.CreatePaymentIntent)
	r.Get("/order-confirmation", deps.CheckoutHandler.OrderConfirmation)

	// Account routes (require authentication)
	account := r.Group(middleware.RequireAuth)
	account.Get("/account", deps.AccountDashboardHandler.ServeHTTP)
	account.Get("/account/orders", deps.OrderHistoryHandler.List)
	account.Get("/account/addresses", deps.AddressHandler.List)
	account.Post("/account/addresses", deps.AddressHandler.Create)
	account.Post("/account/addresses/{id}", deps.AddressHandler.Update)
	account.Post("/account/addresses/{id}/delete", deps.AddressHandler.Delete)
	account.Post("/account/addresses/{id}/default", deps.AddressHandler.SetDefault)
	account.Get("/account/addresses/{id}/json", deps.AddressHandler.GetAddressJSON)
	account.Get("/account/subscriptions", deps.SubscriptionListHandler.ServeHTTP)
	account.Get("/account/subscriptions/portal", deps.SubscriptionPortalHandler.ServeHTTP)
	account.Get("/account/subscriptions/{id}", deps.SubscriptionDetailHandler.ServeHTTP)
	account.Get("/subscribe/checkout", deps.SubscriptionCheckoutHandler.ServeHTTP)
	account.Post("/subscribe", deps.CreateSubscriptionHandler.ServeHTTP)
}
