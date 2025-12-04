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
	r.Get("/signup", deps.AuthHandler.ShowSignupForm)
	r.Get("/signup-success", deps.AuthHandler.ShowSignupSuccess)
	r.Get("/login", deps.AuthHandler.ShowLoginForm)
	r.Post("/logout", deps.AuthHandler.HandleLogout)

	// Password Reset
	r.Get("/forgot-password", deps.AuthHandler.ShowForgotPasswordForm)
	r.Post("/forgot-password", deps.AuthHandler.HandleForgotPassword)
	r.Get("/reset-password", deps.AuthHandler.ShowResetPasswordForm)
	r.Post("/reset-password", deps.AuthHandler.HandleResetPassword)

	// Email Verification
	r.Get("/verify-email", deps.AuthHandler.HandleVerifyEmail)
	r.Get("/resend-verification", deps.AuthHandler.ShowResendVerificationForm)
	r.Post("/resend-verification", deps.AuthHandler.HandleResendVerification)

	// Checkout flow
	r.Get("/checkout", deps.CheckoutHandler.Page)
	r.Post("/checkout/validate-address", deps.CheckoutHandler.ValidateAddress)
	r.Post("/checkout/shipping-rates", deps.CheckoutHandler.GetShippingRates)
	r.Post("/checkout/calculate-total", deps.CheckoutHandler.CalculateTotal)
	r.Post("/checkout/create-payment-intent", deps.CheckoutHandler.CreatePaymentIntent)
	r.Get("/order-confirmation", deps.CheckoutHandler.OrderConfirmation)

	// Subscription product selection (public)
	r.Get("/subscribe", deps.SubscriptionProductsHandler.ServeHTTP)

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

	// Wholesale application (require authentication)
	account.Get("/wholesale/apply", deps.WholesaleApplicationHandler.Form)
	account.Post("/wholesale/apply", deps.WholesaleApplicationHandler.Submit)
	account.Get("/wholesale/status", deps.WholesaleApplicationHandler.Status)

	// Payment methods (require authentication)
	account.Get("/account/payment-methods", deps.PaymentMethodHandler.List)
	account.Get("/account/payment-methods/portal", deps.PaymentMethodHandler.Portal)
	account.Post("/account/payment-methods/{id}/default", deps.PaymentMethodHandler.SetDefault)

	// Profile settings (require authentication)
	account.Get("/account/settings", deps.ProfileHandler.Show)
	account.Post("/account/settings/profile", deps.ProfileHandler.UpdateProfile)
	account.Post("/account/settings/password", deps.ProfileHandler.ChangePassword)
}
