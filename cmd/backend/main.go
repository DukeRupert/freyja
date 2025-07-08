// internal/backend/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/dukerupert/freyja/internal/backend/client"
	"github.com/dukerupert/freyja/internal/backend/database"
	"github.com/dukerupert/freyja/internal/backend/handlers"
)

func main() {
		// Load .env file first
    if err := godotenv.Load(); err != nil {
        fmt.Println("No .env file found")
    }
	
	// Load environment variables
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable required")
	}
	
	serverURL := os.Getenv("FREYJA_SERVER_URL")
	if serverURL == "" {
		log.Fatal("FREYJA_SERVER_URL environment variable required")
	}

	// Create simplified database connection for reads
	db, err := database.NewSimplifiedDB(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create HTTP client for API mutations
	freyjaClient := client.NewFreyjaClient(serverURL)

	// Create handlers with hybrid access
	productHandler := handlers.NewProductHandler(freyjaClient, db.GetQueries())

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Admin routes
	admin := e.Group("/admin")
	{
		// Products
		admin.GET("/products", productHandler.ShowProductsPage)
		admin.GET("/products/:id", productHandler.GetProductDetail)
		
		// Add more admin routes as needed
	}

	// Static files (if needed)
	e.Static("/static", "static")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Different from your main server
	}

	log.Printf("🚀 Backend admin panel starting on port %s", port)
	log.Fatal(e.Start(":" + port))
}