package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/handler"
	customMiddleware "github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/provider"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/subscriber"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Return validation error in the format Echo expects
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Config validation failed:", err)
	}

	// Initialize database
	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	// Run migrations (auto-migrate in development)
	autoMigrate := os.Getenv("ENV") == "development"
	if err := db.RunMigrations(autoMigrate); err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("✅ Database connected and migrations completed")

	// Initialize NATS event publisher
	eventPublisher, err := provider.NewNATSEventPublisher(cfg.NATSUrl)
	if err != nil {
		log.Fatal("Failed to create NATS event publisher:", err)
	}
	defer eventPublisher.Close()

	stripeProvider, err := provider.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	if err != nil {
		log.Fatal("Stripe provider initialization failed:", err)
	}

	// Initialize layers: Repository -> Service -> Handler
	// Initialize repositories
	productRepo := repository.NewPostgresProductRepository(db)
	cartRepo := repository.NewPostgresCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	customerRepo := repository.NewPostgresCustomerRepository(db)

	// Initialize services
	productService := service.NewProductService(productRepo, eventPublisher)
	cartService := service.NewCartService(cartRepo, productRepo)
	orderService := service.NewOrderService(orderRepo, cartService, eventPublisher)
	customerService := service.NewCustomerService(customerRepo, stripeProvider, eventPublisher)
	checkoutService := service.NewCheckoutService(customerService, cartService, orderService, stripeProvider, eventPublisher)
	adminService := service.NewAdminService(customerService, productService, eventPublisher)

	// Initialize handlers
	productHandler := handler.NewProductHandler(productService)
	cartHandler := handler.NewCartHandler(cartService)
	checkoutHandler := handler.NewCheckoutHandler(checkoutService)
	orderHandler := handler.NewOrderHandler(orderService)
	customerHandler := handler.NewCustomerHandler(customerService)
	adminHandler := handler.NewAdminHandler(adminService)
	webhookHandler := handler.NewWebhookHandler(stripeProvider, orderService, customerService)

	// Initialize event subscribers
	customerSubscriber := subscriber.NewCustomerEventSubscriber(customerService, eventPublisher)
	productSubscriber := subscriber.NewProductEventSubscriber(productService, eventPublisher)

	// Start event subscribers in background goroutines
	go func() {
		if err := customerSubscriber.Start(context.Background()); err != nil {
			log.Printf("Failed to start customer event subscriber: %v", err)
		}
	}()

	go func() {
		if err := productSubscriber.Start(context.Background()); err != nil {
			log.Printf("Failed to start product event subscriber: %v", err)
		}
	}()

	// Optional: Run backfill for existing customers (uncomment if needed)
	// go func() {
	//     time.Sleep(5 * time.Second) // Wait for services to be fully ready
	//     if err := customerSubscriber.EnsureAllCustomersHaveStripeIDs(context.Background()); err != nil {
	//         log.Printf("Failed to backfill customer Stripe IDs: %v", err)
	//     }
	// }()

	log.Println("✅ Event subscribers started")

	// Create Echo instance
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Add request ID for tracing
	e.Use(middleware.RequestID())

	e.Use(customMiddleware.PrometheusMiddleware())

	// Add Prometheus metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Health check endpoint
	e.GET("/health", healthCheck)

	// Hello world endpoint
	e.GET("/", helloWorld)

	webhooks := e.Group("/webhooks")
	{
		webhooks.POST("/stripe", webhookHandler.HandleStripeWebhook)
	}

	// API v1 routes group
	api := e.Group("/api/v1")

	// Product routes
	products := api.Group("/products")
	{
		products.GET("", productHandler.GetProducts)                   // GET /api/v1/products
		products.GET("/in-stock", productHandler.GetInStockProducts)   // GET /api/v1/products/in-stock
		products.GET("/low-stock", productHandler.GetLowStockProducts) // GET /api/v1/products/low-stock
		products.GET("/stats", productHandler.GetProductStats)         // GET /api/v1/products/stats
		products.GET("/:id", productHandler.GetProduct)                // GET /api/v1/products/:id
	}

	// Cart routes
	cart := api.Group("/cart")
	{
		cart.GET("", cartHandler.GetCart)      // GET /api/v1/cart
		cart.DELETE("", cartHandler.ClearCart) // DELETE /api/v1/cart
		// cart.GET("/summary", cartHandler.GetCartSummary)  // GET /api/v1/cart/summary
		cart.POST("/items", cartHandler.AddItem)          // POST /api/v1/cart/items
		cart.PUT("/items/:id", cartHandler.UpdateItem)    // PUT /api/v1/cart/items/:id
		cart.DELETE("/items/:id", cartHandler.RemoveItem) // DELETE /api/v1/cart/items/:id
	}

	checkout := api.Group("/checkout")
	{
		checkout.POST("", checkoutHandler.CreateCheckoutSession)
	}

	// Customer routes
	orders := api.Group("/orders")
	{
		orders.GET("", orderHandler.GetOrders)    // Customer order history
		orders.GET("/:id", orderHandler.GetOrder) // Order details
		// orders.POST("/:id/cancel", orderHandler.CancelOrder) // Cancel order
	}

	customers := api.Group("/customers")
	{
		customers.POST("", customerHandler.CreateCustomer)                    // POST /api/v1/customers
		customers.GET("", customerHandler.GetCustomers)                       // GET /api/v1/customers
		customers.GET("/:id", customerHandler.GetCustomerByID)                // GET /api/v1/customers/:id
		customers.PUT("/:id", customerHandler.UpdateCustomer)                 // PUT /api/v1/customers/:id
		customers.DELETE("/:id", customerHandler.DeleteCustomer)              // DELETE /api/v1/customers/:id
		customers.GET("/by-email/:email", customerHandler.GetCustomerByEmail) // GET /api/v1/customers/by-email/:email
		customers.GET("/search", customerHandler.SearchCustomers)             // GET /api/v1/customers/search
		customers.POST("/:id/stripe", customerHandler.EnsureStripeCustomer)   // POST /api/v1/customers/:id/stripe

		// Admin/analytics routes
		customers.GET("/stats", customerHandler.GetCustomerStats)                   // GET /api/v1/customers/stats
		customers.POST("/sync/stripe", customerHandler.SyncStripeCustomers)         // POST /api/v1/customers/sync/stripe
		customers.GET("/without-stripe", customerHandler.GetCustomersWithoutStripe) // GET /api/v1/customers/without-stripe
	}

	// Admin routes
	admin := api.Group("/admin")
	{
		admin.GET("/orders", orderHandler.GetAllOrders) // All orders
		// admin.PUT("/orders/:id/status", orderHandler.UpdateOrderStatus) // Update status
		// admin.GET("/orders/stats", orderHandler.GetOrderStats)          // Analytics

		admin.POST("/products", productHandler.CreateProduct)    // Create product
		admin.PUT("/products/:id", productHandler.UpdateProduct) // Update product
		// admin.PUT("/products/:id/stock", productHandler.UpdateStock) // Update stock only
		// admin.DELETE("/products/:id", productHandler.DeleteProduct)  // Delete product

		// Backfill operations
		admin.POST("/backfill/customers", adminHandler.BackfillCustomers)     // Start customer backfill
		admin.POST("/backfill/products", adminHandler.BackfillProducts)       // Start product backfill
		admin.GET("/sync/status", adminHandler.GetSyncStatus)                 // Get overall sync status
		admin.GET("/backfill/:job_id/status", adminHandler.GetBackfillStatus) // Get specific job status
	}

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server with graceful shutdown support
	e.Logger.Info("🚀 Coffee E-commerce API starting on port " + port)
	e.Logger.Fatal(e.Start(":" + port))
}

// Health check handler
func healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "coffee-ecommerce-api",
		"version": "1.0.0",
	})
}

// Hello world handler
func helloWorld(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Welcome to the Coffee E-commerce API!",
		"version": "1.0.0",
		"docs":    "/api/v1",
		"health":  "/health",
		"metrics": "/metrics",
		"endpoints": map[string]interface{}{
			"products": map[string]interface{}{
				"list":      "GET /api/v1/products",
				"detail":    "GET /api/v1/products/{id}",
				"search":    "GET /api/v1/products?search={query}",
				"in_stock":  "GET /api/v1/products/in-stock",
				"low_stock": "GET /api/v1/products/low-stock",
				"stats":     "GET /api/v1/products/stats",
			},
			"cart": map[string]interface{}{
				"get":         "GET /api/v1/cart",
				"clear":       "DELETE /api/v1/cart",
				"summary":     "GET /api/v1/cart/summary",
				"add_item":    "POST /api/v1/cart/items",
				"update_item": "PUT /api/v1/cart/items/{id}",
				"remove_item": "DELETE /api/v1/cart/items/{id}",
			},
			"checkout": map[string]interface{}{
				"create_session": "POST /api/v1/checkout",
				"webhook":        "POST /api/v1/webhooks/stripe",
			},
			"orders": map[string]interface{}{
				"customer_orders": "GET /api/v1/orders",
				"order_detail":    "GET /api/v1/orders/{id}",
				"cancel_order":    "POST /api/v1/orders/{id}/cancel",
			},
			"admin": map[string]interface{}{
				"all_orders":    "GET /api/v1/admin/orders",
				"update_status": "PUT /api/v1/admin/orders/{id}/status",
				"order_stats":   "GET /api/v1/admin/orders/stats",
			},
		},
		"authentication": map[string]interface{}{
			"note":         "Most endpoints require authentication via JWT token or X-Customer-ID header for testing",
			"cart_session": "Cart operations for guests require X-Session-ID header",
		},
		"example_usage": map[string]interface{}{
			"get_products":    "curl -X GET 'http://localhost:8080/api/v1/products'",
			"add_to_cart":     "curl -X POST 'http://localhost:8080/api/v1/cart/items' -H 'X-Customer-ID: 1' -H 'Content-Type: application/json' -d '{\"product_id\": 1, \"quantity\": 2}'",
			"create_checkout": "curl -X POST 'http://localhost:8080/api/v1/checkout' -H 'X-Customer-ID: 1' -H 'Content-Type: application/json' -d '{\"success_url\": \"http://localhost:3000/success\", \"cancel_url\": \"http://localhost:3000/cart\"}'",
			"view_orders":     "curl -X GET 'http://localhost:8080/api/v1/orders' -H 'X-Customer-ID: 1'",
		},
		"status": "MVP Ready",
		"features": []string{
			"Product catalog with search",
			"Shopping cart (authenticated & guest)",
			"Stripe checkout integration",
			"Order management",
			"Admin dashboard",
			"Event-driven architecture with NATS",
			"Prometheus metrics",
		},
	})
}
