package routes

import (
	"net/http"

	"github.com/dukerupert/freyja/internal/handler/saas"
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
	CartViewHandler       http.Handler
	AddToCartHandler      http.Handler
	UpdateCartItemHandler http.Handler
	RemoveCartItemHandler http.Handler

	// Auth
	SignupHandler http.Handler
	LoginHandler  http.Handler
	LogoutHandler http.Handler

	// Password Reset
	ForgotPasswordHandler http.Handler
	ResetPasswordHandler  http.Handler

	// Checkout
	CheckoutPageHandler        http.Handler
	ValidateAddressHandler     http.Handler
	GetShippingRatesHandler    http.Handler
	CalculateTotalHandler      http.Handler
	CreatePaymentIntentHandler http.Handler
	OrderConfirmationHandler   http.Handler

	// Account (authenticated)
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
	ProductListHandler   http.Handler
	ProductFormHandler   http.Handler
	ProductDetailHandler http.Handler
	SKUFormHandler       http.Handler

	// Orders
	OrderListHandler         http.Handler
	OrderDetailHandler       http.Handler
	UpdateOrderStatusHandler http.Handler
	CreateShipmentHandler    http.Handler

	// Customers
	CustomerListHandler     http.Handler
	CustomerDetailHandler   http.Handler
	WholesaleApprovalHandler http.Handler

	// Subscriptions
	SubscriptionListHandler   http.Handler
	SubscriptionDetailHandler http.Handler

	// Invoices
	InvoiceListHandler    http.Handler
	InvoiceDetailHandler  http.Handler
	SendInvoiceHandler    http.Handler
	VoidInvoiceHandler    http.Handler
	RecordPaymentHandler  http.Handler
	CreateInvoiceHandler  http.Handler
}

// WebhookDeps contains dependencies for webhook routes
type WebhookDeps struct {
	StripeHandler http.HandlerFunc
}
