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

	// Legal/static pages
	r.Get("/privacy", deps.PagesHandler.Privacy)
	r.Get("/terms", deps.PagesHandler.Terms)
	r.Get("/shipping", deps.PagesHandler.Shipping)

	// Product browsing
	r.Get("/products", deps.ProductHandler.List)
	r.Get("/products/{slug}", deps.ProductHandler.Detail)

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
	r.Get("/subscribe", deps.ProductHandler.SubscribeProducts)

	// Account routes (require authentication)
	account := r.Group(middleware.RequireAuth)
	account.Get("/account", deps.AccountHandler.Dashboard)
	account.Get("/account/orders", deps.AccountHandler.OrderList)
	account.Get("/account/addresses", deps.AccountHandler.AddressList)
	account.Post("/account/addresses", deps.AccountHandler.AddressCreate)
	account.Post("/account/addresses/{id}", deps.AccountHandler.AddressUpdate)
	account.Post("/account/addresses/{id}/delete", deps.AccountHandler.AddressDelete)
	account.Post("/account/addresses/{id}/default", deps.AccountHandler.AddressSetDefault)
	account.Get("/account/addresses/{id}/json", deps.AccountHandler.AddressGetJSON)
	account.Get("/account/subscriptions", deps.SubscriptionHandler.List)
	account.Get("/account/subscriptions/portal", deps.SubscriptionHandler.Portal)
	account.Get("/account/subscriptions/{id}", deps.SubscriptionHandler.Detail)
	account.Get("/subscribe/checkout", deps.SubscriptionHandler.Checkout)
	account.Post("/subscribe", deps.SubscriptionHandler.Create)

	// Wholesale application (require authentication)
	account.Get("/wholesale/apply", deps.WholesaleApplicationHandler.Form)
	account.Post("/wholesale/apply", deps.WholesaleApplicationHandler.Submit)
	account.Get("/wholesale/status", deps.WholesaleApplicationHandler.Status)

	// Wholesale ordering (require authentication + wholesale account)
	account.Get("/wholesale/order", deps.WholesaleOrderingHandler.Order)
	account.Post("/wholesale/cart/batch", deps.WholesaleOrderingHandler.BatchAdd)

	// Payment methods (require authentication)
	account.Get("/account/payment-methods", deps.AccountHandler.PaymentMethodList)
	account.Get("/account/payment-methods/portal", deps.AccountHandler.PaymentMethodPortal)
	account.Post("/account/payment-methods/{id}/default", deps.AccountHandler.PaymentMethodSetDefault)

	// Profile settings (require authentication)
	account.Get("/account/settings", deps.AccountHandler.ProfileShow)
	account.Post("/account/settings/profile", deps.AccountHandler.ProfileUpdate)
	account.Post("/account/settings/password", deps.AccountHandler.PasswordChange)
}
