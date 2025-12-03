package routes

import (
	"net/http"

	"github.com/dukerupert/freyja/internal/handler/admin"
	"github.com/dukerupert/freyja/internal/handler/saas"
	"github.com/dukerupert/freyja/internal/handler/storefront"
)

// SaaSDeps contains dependencies for SaaS marketing routes
type SaaSDeps struct {
	Handler *saas.PageHandler
}

// StorefrontDeps contains dependencies for storefront routes
type StorefrontDeps struct {
	// Home
	HomeHandler http.Handler

	// Products
	ProductListHandler   http.Handler
	ProductDetailHandler http.Handler

	// Cart
	CartHandler *storefront.CartHandler

	// Auth
	SignupHandler *storefront.SignupHandler
	LoginHandler  *storefront.LoginHandler
	LogoutHandler *storefront.LogoutHandler

	// Password Reset
	ForgotPasswordHandler *storefront.ForgotPasswordHandler
	ResetPasswordHandler  *storefront.ResetPasswordHandler

	// Checkout
	CheckoutHandler *storefront.CheckoutHandler

	// Account (authenticated) - subscriptions still use http.Handler for now
	SubscriptionListHandler     http.Handler
	SubscriptionDetailHandler   http.Handler
	SubscriptionPortalHandler   http.Handler
	SubscriptionCheckoutHandler http.Handler
	CreateSubscriptionHandler   http.Handler
}

// AdminDeps contains dependencies for admin routes
type AdminDeps struct {
	// Dashboard
	DashboardHandler http.Handler

	// Products
	ProductHandler *admin.ProductHandler

	// Orders
	OrderHandler *admin.OrderHandler

	// Customers
	CustomerHandler *admin.CustomerHandler

	// Subscriptions
	SubscriptionHandler *admin.SubscriptionHandler

	// Invoices
	InvoiceHandler *admin.InvoiceHandler
}

// WebhookDeps contains dependencies for webhook routes
type WebhookDeps struct {
	StripeHandler http.HandlerFunc
}
