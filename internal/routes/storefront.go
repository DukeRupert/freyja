package routes

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/router"
)

// RegisterStorefrontRoutes registers all customer-facing storefront routes.
// These routes are for the tenant's e-commerce storefront.
//
// When multi-tenancy is enabled, these routes will be served per-tenant
// based on subdomain or custom domain.
//
// If tenantMiddleware is provided and not nil, it will be applied to wrap
// all storefront routes with tenant resolution. This should be the
// middleware.ResolveTenant middleware configured with TenantConfig.
func RegisterStorefrontRoutes(r *router.Router, deps StorefrontDeps, tenantMiddleware func(http.Handler) http.Handler) {
	// If tenant middleware is provided, wrap all storefront routes
	var storefrontRouter *router.Router
	if tenantMiddleware != nil {
		storefrontRouter = r.Group(
			tenantMiddleware,
			middleware.RequireTenant,
		)
	} else {
		// No tenant middleware - use router directly (single-tenant mode)
		storefrontRouter = r
	}

	// Home page
	storefrontRouter.Get("/", deps.HomeHandler.ServeHTTP)

	// Legal/static pages
	storefrontRouter.Get("/privacy", deps.PagesHandler.Privacy)
	storefrontRouter.Get("/terms", deps.PagesHandler.Terms)
	storefrontRouter.Get("/shipping", deps.PagesHandler.Shipping)

	// Product browsing
	storefrontRouter.Get("/products", deps.ProductHandler.List)
	storefrontRouter.Get("/products/{slug}", deps.ProductHandler.Detail)

	// Shopping cart
	storefrontRouter.Get("/cart", deps.CartHandler.View)
	storefrontRouter.Post("/cart/add", deps.CartHandler.Add)
	storefrontRouter.Post("/cart/update", deps.CartHandler.Update)
	storefrontRouter.Post("/cart/remove", deps.CartHandler.Remove)

	// Authentication (GET routes only - POST routes registered separately with rate limiting)
	storefrontRouter.Get("/signup", deps.AuthHandler.ShowSignupForm)
	storefrontRouter.Get("/signup-success", deps.AuthHandler.ShowSignupSuccess)
	storefrontRouter.Get("/login", deps.AuthHandler.ShowLoginForm)
	storefrontRouter.Post("/logout", deps.AuthHandler.HandleLogout)

	// Password Reset
	storefrontRouter.Get("/forgot-password", deps.AuthHandler.ShowForgotPasswordForm)
	storefrontRouter.Post("/forgot-password", deps.AuthHandler.HandleForgotPassword)
	storefrontRouter.Get("/reset-password", deps.AuthHandler.ShowResetPasswordForm)
	storefrontRouter.Post("/reset-password", deps.AuthHandler.HandleResetPassword)

	// Email Verification
	storefrontRouter.Get("/verify-email", deps.AuthHandler.HandleVerifyEmail)
	storefrontRouter.Get("/resend-verification", deps.AuthHandler.ShowResendVerificationForm)
	storefrontRouter.Post("/resend-verification", deps.AuthHandler.HandleResendVerification)

	// Checkout flow
	storefrontRouter.Get("/checkout", deps.CheckoutHandler.Page)
	storefrontRouter.Post("/checkout/validate-address", deps.CheckoutHandler.ValidateAddress)
	storefrontRouter.Post("/checkout/shipping-rates", deps.CheckoutHandler.GetShippingRates)
	storefrontRouter.Post("/checkout/calculate-total", deps.CheckoutHandler.CalculateTotal)
	storefrontRouter.Post("/checkout/create-payment-intent", deps.CheckoutHandler.CreatePaymentIntent)
	storefrontRouter.Get("/order-confirmation", deps.CheckoutHandler.OrderConfirmation)

	// Subscription product selection (public)
	storefrontRouter.Get("/subscribe", deps.ProductHandler.SubscribeProducts)

	// Account routes (require authentication)
	account := storefrontRouter.Group(middleware.RequireAuth)
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
