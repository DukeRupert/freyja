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
	admin.Get("/admin/products", deps.ProductListHandler.ServeHTTP)
	admin.Get("/admin/products/new", deps.ProductFormHandler.ServeHTTP)
	admin.Post("/admin/products/new", deps.ProductFormHandler.ServeHTTP)
	admin.Get("/admin/products/{id}", deps.ProductDetailHandler.ServeHTTP)
	admin.Get("/admin/products/{id}/edit", deps.ProductFormHandler.ServeHTTP)
	admin.Post("/admin/products/{id}/edit", deps.ProductFormHandler.ServeHTTP)

	// SKU management
	admin.Get("/admin/products/{product_id}/skus/new", deps.SKUFormHandler.ServeHTTP)
	admin.Post("/admin/products/{product_id}/skus/new", deps.SKUFormHandler.ServeHTTP)
	admin.Get("/admin/products/{product_id}/skus/{sku_id}/edit", deps.SKUFormHandler.ServeHTTP)
	admin.Post("/admin/products/{product_id}/skus/{sku_id}/edit", deps.SKUFormHandler.ServeHTTP)

	// Order management
	admin.Get("/admin/orders", deps.OrderListHandler.ServeHTTP)
	admin.Get("/admin/orders/{id}", deps.OrderDetailHandler.ServeHTTP)
	admin.Post("/admin/orders/{id}/status", deps.UpdateOrderStatusHandler.ServeHTTP)
	admin.Post("/admin/orders/{id}/shipments", deps.CreateShipmentHandler.ServeHTTP)

	// Customer management
	admin.Get("/admin/customers", deps.CustomerListHandler.ServeHTTP)

	// Subscription management
	admin.Get("/admin/subscriptions", deps.SubscriptionListHandler.ServeHTTP)
	admin.Get("/admin/subscriptions/{id}", deps.SubscriptionDetailHandler.ServeHTTP)
}
