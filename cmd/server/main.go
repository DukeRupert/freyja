package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/provider"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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

	// Initialize layers: Repository -> Service -> Handler
	// Initialize repositories
	productRepo := repository.NewPostgresProductRepository(db)
	cartRepo := repository.NewPostgresCartRepository(db)
	orderRepo := repository.NewPostgresOrderRepository(db)

	// Initialize services
	productService := service.NewProductService(productRepo)
	cartService := service.NewCartService(cartRepo, productRepo)
	orderService := service.NewOrderService(orderRepo, cartService, eventPublisher)

	// Initialize handlers
	productHandler := handler.NewProductHandler(productService)
	cartHandler := handler.NewCartHandler(cartService)
	orderHandler := handler.NewOrderHandler(orderService)

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Add request ID for tracing
	e.Use(middleware.RequestID())

	// Add Prometheus metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Health check endpoint
	e.GET("/health", healthCheck)

	// Hello world endpoint
	e.GET("/", helloWorld)

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
		cart.GET("", cartHandler.GetCart)                 // GET /api/v1/cart
		cart.DELETE("", cartHandler.ClearCart)            // DELETE /api/v1/cart
		cart.GET("/summary", cartHandler.GetCartSummary)  // GET /api/v1/cart/summary
		cart.POST("/items", cartHandler.AddItem)          // POST /api/v1/cart/items
		cart.PUT("/items/:id", cartHandler.UpdateItem)    // PUT /api/v1/cart/items/:id
		cart.DELETE("/items/:id", cartHandler.RemoveItem) // DELETE /api/v1/cart/items/:id
	}

	// Customer routes
	orders := api.Group("/orders")
	{
		orders.GET("", orderHandler.GetOrders)               // Customer order history
		orders.GET("/:id", orderHandler.GetOrder)            // Order details
		orders.POST("/:id/cancel", orderHandler.CancelOrder) // Cancel order
	}

	// Admin routes
	admin := api.Group("/admin")
	{
		admin.GET("/orders", orderHandler.GetAllOrders)                 // All orders
		admin.PUT("/orders/:id/status", orderHandler.UpdateOrderStatus) // Update status
		admin.GET("/orders/stats", orderHandler.GetOrderStats)          // Analytics
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
		"docs":    "/api/v1",
		"health":  "/health",
		"endpoints": map[string]interface{}{
			"products":       "/api/v1/products",
			"product_detail": "/api/v1/products/{id}",
			"in_stock":       "/api/v1/products/in-stock",
			"low_stock":      "/api/v1/products/low-stock",
			"stats":          "/api/v1/products/stats",
		},
	})
}
