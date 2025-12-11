package routes

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/handler/admin"
	"github.com/dukerupert/hiri/internal/handler/api"
	"github.com/dukerupert/hiri/internal/handler/saas"
	"github.com/dukerupert/hiri/internal/handler/storefront"
)

// SaaSDeps contains dependencies for SaaS marketing routes
type SaaSDeps struct {
	// Page rendering (landing, pricing, about, etc.)
	Handler *saas.PageHandler

	// Checkout (creates Stripe checkout session)
	CheckoutHandler *saas.CheckoutHandler

	// Auth (login, logout, password reset)
	AuthHandler *saas.AuthHandler

	// Setup (account setup after checkout)
	SetupHandler *saas.SetupHandler

	// Webhook (Stripe webhook handler)
	WebhookHandler *saas.WebhookHandler

	// DevBypassHandler for development-only signup bypass (nil in production)
	DevBypassHandler *saas.DevBypassHandler

	// CheckoutURL is passed to pricing page template for client-side fetch
	CheckoutURL string
}

// StorefrontDeps contains dependencies for storefront routes
type StorefrontDeps struct {
	// Home
	HomeHandler http.Handler

	// Products (consolidated: list, detail, subscription products)
	ProductHandler *storefront.ProductHandler

	// Cart
	CartHandler *storefront.CartHandler

	// Auth (consolidated: signup, login, logout, password reset, email verification)
	AuthHandler *storefront.AuthHandler

	// Checkout
	CheckoutHandler *storefront.CheckoutHandler

	// Subscriptions (consolidated: list, detail, portal, checkout, create)
	SubscriptionHandler *storefront.SubscriptionHandler

	// Account (consolidated: dashboard, orders, addresses, payment methods, profile)
	AccountHandler *storefront.AccountHandler

	// Wholesale
	WholesaleApplicationHandler *storefront.WholesaleApplicationHandler
	WholesaleOrderingHandler    *storefront.WholesaleOrderingHandler

	// Static pages (legal, about, contact, etc.)
	PagesHandler *storefront.PagesHandler
}

// AdminDeps contains dependencies for admin routes
type AdminDeps struct {
	// Auth
	LoginHandler  *admin.LoginHandler
	LogoutHandler *admin.LogoutHandler

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

	// Price Lists
	PriceListHandler *admin.PriceListHandler

	// Settings
	TaxRateHandler      *admin.TaxRateHandler
	IntegrationsHandler *admin.IntegrationsHandler
	CustomDomainHandler *admin.CustomDomainHandler
	PageHandler         *admin.PageHandler

	// Onboarding
	OnboardingHandler *admin.OnboardingHandler
}

// WebhookDeps contains dependencies for webhook routes
type WebhookDeps struct {
	StripeHandler http.HandlerFunc
}

// APIDeps contains dependencies for API routes
type APIDeps struct {
	DomainValidationHandler *api.DomainValidationHandler
}
