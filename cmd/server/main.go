package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/server/handler"
	customMiddleware "github.com/dukerupert/freyja/internal/server/middleware"
	"github.com/dukerupert/freyja/internal/server/provider"
	"github.com/dukerupert/freyja/internal/server/repository"
	"github.com/dukerupert/freyja/internal/server/service"
	"github.com/dukerupert/freyja/internal/server/subscriber"
	"github.com/dukerupert/freyja/internal/shared/config"

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
	// Define command line flags
	var (
		logLevel  = flag.String("log-level", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
		logFormat = flag.String("log-format", "json", "Log format (json, console)")
		portFlag  = flag.String("port", "8080", "Server port, default 8080 (overrides PORT environment variable)")
	)
	flag.Parse()

	// Configure zerolog
	logger := customMiddleware.SetupLogger(*logLevel, *logFormat)

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
	eventPublisher, err := provider.NewNATSEventPublisher(cfg.NATSUrl, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create NATS event publisher")
	}
	defer eventPublisher.Close()

	stripeProvider, err := provider.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	if err != nil {
		log.Fatal("Stripe provider initialization failed:", err)
	}

	// Initialize layers: Repository -> Service -> Handler
	// Initialize repositories
	productRepo := repository.NewPostgresProductRepository(db)
	optionRepo := repository.NewPostgresOptionRepository(db)
	variantRepo := repository.NewPostgresVariantRepository(db)
	cartRepo := repository.NewPostgresCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	customerRepo := repository.NewPostgresCustomerRepository(db)

	// Initialize services
	productService := service.NewProductService(productRepo, eventPublisher)
	optionService := service.NewOptionService(optionRepo, variantRepo, productRepo, eventPublisher)
	variantService := service.NewVariantService(variantRepo, productRepo, eventPublisher)
	cartService := service.NewCartService(cartRepo, variantRepo, eventPublisher)
	orderService := service.NewOrderService(orderRepo, cartService, variantRepo, eventPublisher)
	customerService := service.NewCustomerService(customerRepo, stripeProvider, eventPublisher)
	checkoutService := service.NewCheckoutService(customerService, cartService, orderService, stripeProvider, eventPublisher)
	adminService := service.NewAdminService(customerService, productService, variantService, eventPublisher)

	// Initialize handlers
	variantHandler := handler.NewVariantHandler(variantService)
	optionHandler := handler.NewOptionHandler(optionService)
	productHandler := handler.NewProductHandler(productService, variantService)
	cartHandler := handler.NewCartHandler(cartService)
	checkoutHandler := handler.NewCheckoutHandler(checkoutService)
	orderHandler := handler.NewOrderHandler(orderService)
	customerHandler := handler.NewCustomerHandler(customerService)
	adminHandler := handler.NewAdminHandler(adminService)
	webhookHandler := handler.NewWebhookHandler(stripeProvider, orderService, customerService)

	// Initialize event subscribers
	customerSubscriber := subscriber.NewCustomerEventSubscriber(customerService, eventPublisher, logger)
	productSubscriber := subscriber.NewProductEventSubscriber(productService, variantService, eventPublisher, logger)
	materializedViewSubscriber := subscriber.NewMaterializedViewSubscriber(productRepo, eventPublisher, logger)

	// Start event subscribers in background goroutines
	go func() {
		if err := customerSubscriber.Start(context.Background()); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start customer event subscriber")
		}
	}()

	go func() {
		if err := productSubscriber.Start(context.Background()); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start product event subscriber")
		}
	}()

	go func() {
		if err := materializedViewSubscriber.Start(context.Background()); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start materialized view subscriber")
		}
	}()

	logger.Info().Msg("[OK] Event subscribers started")

	// Refresh materialized view on startup
	go func() {
		// Wait a moment for all services to be fully initialized
		time.Sleep(2 * time.Second)

		if err := productService.RefreshProductSummary(context.Background()); err != nil {
			logger.Warn().Err(err).Msg("Failed to refresh materialized view on startup")
		} else {
			logger.Info().Msg("[OK] Materialized view refreshed on startup")
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
	e.Use(customMiddleware.ZerologMiddleware(logger))
	e.Use(middleware.Recover())
	// Permissive settings, DO NOT DEPLOY!
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://refactored-umbrella-rp9xx597vq535wg6-8081.app.github.dev", "http://localhost:8081"},
		AllowMethods:     []string{""},
		AllowHeaders:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodHead},
		AllowCredentials: false, // Must be false when using "*" for origins
	}))

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

		// Variant-specific product endpoints
		products.GET("/:id/variants", productHandler.GetProductVariants)       // GET /api/v1/products/{id}/variants
		products.GET("/variants/search", productHandler.SearchProductVariants) // GET /api/v1/products/variants/search
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
		// Existing admin routes...
		admin.GET("/orders", orderHandler.GetAllOrders)                 // All orders
		admin.PUT("/orders/:id/status", orderHandler.UpdateOrderStatus) // Update status
		admin.GET("/orders/stats", orderHandler.GetOrderStats)          // Analytics

		admin.POST("/products", productHandler.CreateProduct)    // Create product
		admin.PUT("/products/:id", productHandler.UpdateProduct) // Update product

		// Core variant CRUD operations
		admin.POST("/variants", variantHandler.CreateVariant)        // POST /api/v1/admin/variants
		admin.GET("/variants/:id", variantHandler.GetVariant)        // GET /api/v1/admin/variants/{id}
		admin.PUT("/variants/:id", variantHandler.UpdateVariant)     // PUT /api/v1/admin/variants/{id}
		admin.DELETE("/variants/:id", variantHandler.ArchiveVariant) // DELETE /api/v1/admin/variants/{id}

		// Variant activation/deactivation
		admin.POST("/variants/:id/activate", variantHandler.ActivateVariant)     // POST /api/v1/admin/variants/{id}/activate
		admin.POST("/variants/:id/deactivate", variantHandler.DeactivateVariant) // POST /api/v1/admin/variants/{id}/deactivate

		// Stock management routes
		admin.PUT("/variants/:id/stock", variantHandler.UpdateVariantStock)               // PUT /api/v1/admin/variants/{id}/stock
		admin.POST("/variants/:id/stock/increment", variantHandler.IncrementVariantStock) // POST /api/v1/admin/variants/{id}/stock/increment
		admin.POST("/variants/:id/stock/decrement", variantHandler.DecrementVariantStock) // POST /api/v1/admin/variants/{id}/stock/decrement

		// Product-variant relationship routes
		admin.GET("/products/:product_id/variants", variantHandler.GetVariantsByProduct) // GET /api/v1/admin/products/{product_id}/variants

		// Variant discovery and management routes
		admin.GET("/variants/low-stock", variantHandler.GetLowStockVariants)             // GET /api/v1/admin/variants/low-stock
		admin.GET("/variants/search", variantHandler.SearchVariants)                     // GET /api/v1/admin/variants/search
		admin.GET("/variants/:id/availability", variantHandler.CheckVariantAvailability) // GET /api/v1/admin/variants/{id}/availability

		// Existing backfill operations...
		admin.POST("/backfill/customers", adminHandler.BackfillCustomers)     // Start customer backfill
		admin.POST("/backfill/products", adminHandler.BackfillProducts)       // Start product backfill
		admin.GET("/sync/status", adminHandler.GetSyncStatus)                 // Get overall sync status
		admin.GET("/backfill/:job_id/status", adminHandler.GetBackfillStatus) // Get specific job status

		// Product-specific option management
		admin.POST("/products/:product_id/options", optionHandler.CreateProductOption) // POST /api/v1/admin/products/{product_id}/options
		admin.GET("/products/:product_id/options", optionHandler.GetProductOptions)    // GET /api/v1/admin/products/{product_id}/options

		// Individual option management
		admin.GET("/options/:id", optionHandler.GetProductOption)       // GET /api/v1/admin/options/{id}
		admin.PUT("/options/:id", optionHandler.UpdateProductOption)    // PUT /api/v1/admin/options/{id}
		admin.DELETE("/options/:id", optionHandler.DeleteProductOption) // DELETE /api/v1/admin/options/{id}

		// Option value management
		admin.POST("/options/:option_id/values", optionHandler.CreateOptionValue) // POST /api/v1/admin/options/{option_id}/values
		admin.GET("/options/:option_id/values", optionHandler.GetOptionValues)    // GET /api/v1/admin/options/{option_id}/values
		admin.GET("/option-values/:id", optionHandler.GetOptionValue)             // GET /api/v1/admin/option-values/{id}
		admin.PUT("/option-values/:id", optionHandler.UpdateOptionValue)          // PUT /api/v1/admin/option-values/{id}
		admin.DELETE("/option-values/:id", optionHandler.DeleteOptionValue)       // DELETE /api/v1/admin/option-values/{id}

		// Analytics and management endpoints
		admin.GET("/options/:option_id/usage", optionHandler.GetOptionUsageStats)               // GET /api/v1/admin/options/{option_id}/usage
		admin.GET("/products/:product_id/option-popularity", optionHandler.GetOptionPopularity) // GET /api/v1/admin/products/{product_id}/option-popularity
		admin.GET("/options/orphaned", optionHandler.GetOrphanedOptions)
	}

	// Get port from environment or default to 8080
	port := getPort(portFlag)

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

// Get port with precedence: CLI flag > environment variable > default
func getPort(portFlag *string) string {
	// First check command line flag
	if *portFlag != "" {
		return *portFlag
	}

	// Then check environment variable
	if envPort := os.Getenv("PORT"); envPort != "" {
		return envPort
	}

	// Default fallback
	return "8080"
}
