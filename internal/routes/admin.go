package routes

import (
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/router"
)

// RegisterAdminRoutes registers all admin dashboard routes.
// All routes are protected by admin authentication middleware.
//
// These routes are served at /admin/* and share the same
// domain/port as the storefront.
func RegisterAdminRoutes(r *router.Router, deps AdminDeps) {
	// All admin routes require admin authentication
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

	// Order management
	admin.Get("/admin/orders", deps.OrderHandler.List)
	admin.Get("/admin/orders/{id}", deps.OrderHandler.Detail)
	admin.Post("/admin/orders/{id}/status", deps.OrderHandler.UpdateStatus)
	admin.Post("/admin/orders/{id}/shipments", deps.OrderHandler.CreateShipment)

	// Customer management
	admin.Get("/admin/customers", deps.CustomerHandler.List)
	admin.Get("/admin/customers/{id}", deps.CustomerHandler.Detail)
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
}
