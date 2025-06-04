package main

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Health check endpoint
	e.GET("/health", healthCheck)

	// Hello world endpoint
	e.GET("/", helloWorld)

	// API v1 routes group
	api := e.Group("/api/v1")

	// Basic API endpoints for MVP
	api.GET("/products", getProducts)
	api.GET("/products/:id", getProduct)

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
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
	})
}

// Placeholder product handlers (MVP stubs)
func getProducts(c echo.Context) error {
	// Mock product data for now
	products := []map[string]interface{}{
		{
			"id":          1,
			"name":        "Ethiopian Yirgacheffe",
			"description": "Bright, floral notes with citrus finish",
			"price":       1800, // $18.00 in cents
			"stock":       25,
			"active":      true,
		},
		{
			"id":          2,
			"name":        "Colombian Supremo",
			"description": "Rich, full-bodied with chocolate undertones",
			"price":       1600, // $16.00 in cents
			"stock":       32,
			"active":      true,
		},
		{
			"id":          3,
			"name":        "Guatemala Antigua",
			"description": "Medium body with spicy and smoky flavors",
			"price":       1750, // $17.50 in cents
			"stock":       18,
			"active":      true,
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
		"total":    len(products),
	})
}

func getProduct(c echo.Context) error {
	id := c.Param("id")

	// Mock single product data
	if id == "1" {
		product := map[string]interface{}{
			"id":            1,
			"name":          "Ethiopian Yirgacheffe",
			"description":   "Bright, floral notes with citrus finish. Grown at high altitude in the Sidama region.",
			"price":         1800,
			"stock":         25,
			"active":        true,
			"origin":        "Ethiopia",
			"roast_level":   "Medium-Light",
			"tasting_notes": []string{"Floral", "Citrus", "Tea-like", "Bright acidity"},
		}
		return c.JSON(http.StatusOK, product)
	}

	return c.JSON(http.StatusNotFound, map[string]interface{}{
		"error": "Product not found",
		"code":  "PRODUCT_NOT_FOUND",
	})
}
