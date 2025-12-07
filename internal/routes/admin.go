package routes

import (
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/router"
)

// RegisterAdminRoutes registers all admin dashboard routes.
// Auth routes are public, all other routes are protected by admin authentication middleware.
//
// These routes are served at /admin/* and share the same
// domain/port as the storefront.
func RegisterAdminRoutes(r *router.Router, deps AdminDeps) {
	// Admin auth routes (public - no authentication required)
	// Note: POST /admin/login is registered in main.go with rate limiting
	r.Get("/admin/login", deps.LoginHandler.ShowForm)
	r.Post("/admin/logout", deps.LogoutHandler.HandleSubmit)

	// All other admin routes require admin authentication
	admin := r.Group(middleware.RequireAdmin)

	// Dashboard
	admin.Get("/admin", deps.DashboardHandler.ServeHTTP)

	// Product management
	admin.Get("/admin/products", deps.ProductHandler.List)
	admin.Get("/admin/products/new", deps.ProductHandler.ShowForm)
	admin.Post("/admin/products/new", deps.ProductHandler.HandleForm)
	admin.Get("/admin/products/{id}", deps.ProductHandler.Detail)
	admin.Get("/admin/products/{id}/edit", deps.ProductHandler.ShowForm)
	admin.Post("/admin/products/{id}/edit", deps.ProductHandler.HandleForm)

	// SKU management
	admin.Get("/admin/products/{product_id}/skus/new", deps.ProductHandler.ShowSKUForm)
	admin.Post("/admin/products/{product_id}/skus/new", deps.ProductHandler.HandleSKUForm)
	admin.Get("/admin/products/{product_id}/skus/{sku_id}/edit", deps.ProductHandler.ShowSKUForm)
	admin.Post("/admin/products/{product_id}/skus/{sku_id}/edit", deps.ProductHandler.HandleSKUForm)

	// Image management (upload has stricter rate limiting)
	uploadLimited := admin.Group(middleware.StrictRateLimit())
	uploadLimited.Post("/admin/products/{id}/images/upload", deps.ProductHandler.UploadImage)

	admin.Delete("/admin/products/{product_id}/images/{image_id}", deps.ProductHandler.DeleteImage)
	admin.Post("/admin/products/{product_id}/images/{image_id}/default", deps.ProductHandler.SetPrimary)
	admin.Post("/admin/products/{product_id}/images/{image_id}/metadata", deps.ProductHandler.UpdateImageMetadata)

	// Order management
	admin.Get("/admin/orders", deps.OrderHandler.List)
	admin.Get("/admin/orders/{id}", deps.OrderHandler.Detail)
	admin.Post("/admin/orders/{id}/status", deps.OrderHandler.UpdateStatus)
	admin.Post("/admin/orders/{id}/shipments", deps.OrderHandler.CreateShipment)

	// Customer management
	admin.Get("/admin/customers", deps.CustomerHandler.List)
	admin.Get("/admin/customers/{id}", deps.CustomerHandler.Detail)
	admin.Get("/admin/customers/{id}/edit", deps.CustomerHandler.Edit)
	admin.Post("/admin/customers/{id}", deps.CustomerHandler.Update)
	admin.Post("/admin/customers/{id}/wholesale/{action}", deps.CustomerHandler.WholesaleApproval)

	// Subscription management
	admin.Get("/admin/subscriptions", deps.SubscriptionHandler.List)
	admin.Get("/admin/subscriptions/{id}", deps.SubscriptionHandler.Detail)

	// Invoice management
	admin.Get("/admin/invoices", deps.InvoiceHandler.List)
	admin.Get("/admin/invoices/new", deps.InvoiceHandler.ShowCreateForm)
	admin.Post("/admin/invoices/new", deps.InvoiceHandler.HandleCreate)
	admin.Get("/admin/invoices/{id}", deps.InvoiceHandler.Detail)
	admin.Post("/admin/invoices/{id}/send", deps.InvoiceHandler.Send)
	admin.Post("/admin/invoices/{id}/void", deps.InvoiceHandler.Void)
	admin.Get("/admin/invoices/{id}/payment", deps.InvoiceHandler.ShowPaymentForm)
	admin.Post("/admin/invoices/{id}/payment", deps.InvoiceHandler.HandlePayment)

	// Price list management
	admin.Get("/admin/price-lists", deps.PriceListHandler.List)
	admin.Get("/admin/price-lists/new", deps.PriceListHandler.ShowForm)
	admin.Post("/admin/price-lists/new", deps.PriceListHandler.HandleForm)
	admin.Get("/admin/price-lists/{id}", deps.PriceListHandler.Detail)
	admin.Get("/admin/price-lists/{id}/edit", deps.PriceListHandler.ShowForm)
	admin.Post("/admin/price-lists/{id}/edit", deps.PriceListHandler.HandleForm)
	admin.Post("/admin/price-lists/{id}/entries", deps.PriceListHandler.UpdateEntry)
	admin.Post("/admin/price-lists/{id}/delete", deps.PriceListHandler.Delete)

	// Settings: Tax rates
	admin.Get("/admin/settings/tax-rates", deps.TaxRateHandler.ListPage)
	admin.Post("/admin/settings/tax-rates", deps.TaxRateHandler.Create)
	admin.Post("/admin/settings/tax-rates/{id}", deps.TaxRateHandler.Update)
	admin.Delete("/admin/settings/tax-rates/{id}", deps.TaxRateHandler.Delete)

	// Settings: Provider integrations
	admin.Get("/admin/settings/integrations", deps.IntegrationsHandler.ListPage)
	admin.Get("/admin/settings/integrations/{type}", deps.IntegrationsHandler.ConfigPage)
	admin.Post("/admin/settings/integrations/{type}", deps.IntegrationsHandler.SaveConfig)
	admin.Post("/admin/settings/integrations/{type}/validate", deps.IntegrationsHandler.ValidateConfig)
	admin.Post("/admin/settings/integrations/{type}/test", deps.IntegrationsHandler.TestConnection)

	// Onboarding checklist
	admin.Get("/admin/onboarding", deps.OnboardingHandler.GetStatus)
	admin.Get("/admin/api/onboarding", deps.OnboardingHandler.GetStatusJSON)
	admin.Get("/admin/api/onboarding/launch-ready", deps.OnboardingHandler.IsLaunchReady)
	admin.Post("/admin/onboarding/{item_id}/skip", deps.OnboardingHandler.SkipItem)
	admin.Delete("/admin/onboarding/{item_id}/skip", deps.OnboardingHandler.UnskipItem)
}
