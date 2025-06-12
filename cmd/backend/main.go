// cmd/admin/main.go
package main

import (
	"log"

	"github.com/dukerupert/freyja/internal/backend/client"
	"github.com/dukerupert/freyja/internal/backend/handlers"
	"github.com/dukerupert/freyja/internal/shared/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Config validation failed:", err)
	}

	// Default Freyja API URL (can be overridden with env var)
	freyjaAPIURL := "http://localhost:8080"
	if apiURL := cfg.ApiURL; apiURL != "" {
		freyjaAPIURL = apiURL
	}

	log.Printf("✅ Admin panel connecting to Freyja API at: %s", freyjaAPIURL)

	// Initialize Freyja API client
	freyjaClient := client.NewFreyjaClient(freyjaAPIURL)

	// Initialize handlers
	productHandler := handlers.NewProductHandler(freyjaClient)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files (CSS, JS)
	e.Static("/static", "web/static")

	// Product routes
	e.GET("/", productHandler.ShowProductsPage)
	e.GET("/products", productHandler.ShowProductsTable)
	e.GET("/products/add", productHandler.ShowAddProductModal)
	e.POST("/products", productHandler.CreateProduct)
	e.PUT("/products/:id", productHandler.UpdateProduct)
	e.DELETE("/products/:id", productHandler.DeleteProduct)

	// Future: Add other handler groups here
	// orderHandler := handlers.NewOrderHandler(freyjaClient)
	// customerHandler := handlers.NewCustomerHandler(freyjaClient)

	log.Println("🚀 Admin panel starting on :8081")
	log.Fatal(e.Start(":8081"))
}