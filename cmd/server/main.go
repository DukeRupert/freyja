package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/handler"
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

	// Initialize layers: Repository -> Service -> Handler
	productRepo := repository.NewPostgresProductRepository(db)
	productService := service.NewProductService(productRepo)
	productHandler := handler.NewProductHandler(productService)

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
