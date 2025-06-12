package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/dukerupert/freyja/web/admin/views"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type AdminServer struct {
	freyjaAPIURL string
	httpClient   *http.Client
}

func NewAdminServer(freyjaAPIURL string) *AdminServer {
	return &AdminServer{
		freyjaAPIURL: freyjaAPIURL,
		httpClient:   &http.Client{},
	}
}

func (s *AdminServer) fetchProducts() ([]interfaces.Product, error) {
	resp, err := s.httpClient.Get(s.freyjaAPIURL + "/products")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// The API returns a wrapper object with a "products" field
	var response struct {
		Products []interfaces.Product `json:"products"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal products: %w", err)
	}

	return response.Products, nil
}

func (s *AdminServer) showProductsPage(c echo.Context) error {
	// Fetch products from Freyja API
	products, err := s.fetchProducts()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return c.String(http.StatusInternalServerError, "Error fetching products")
	}

	// Render the products page using the views package
	component := views.ProductsPage(products)
	return component.Render(context.Background(), c.Response().Writer)
}

func (s *AdminServer) showProductsTable(c echo.Context) error {
	// Fetch products from Freyja API
	products, err := s.fetchProducts()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return c.String(http.StatusInternalServerError, "Error fetching products")
	}

	// Render just the table component using the views package
	component := views.ProductsTable(products)
	return component.Render(context.Background(), c.Response().Writer)
}

func (s *AdminServer) showAddProductModal(c echo.Context) error {
	// Render the add product modal using the views package
	component := views.AddProductModal()
	return component.Render(context.Background(), c.Response().Writer)
}

func (s *AdminServer) createProduct(c echo.Context) error {
	// Parse form data
	name := c.FormValue("name")
	description := c.FormValue("description")
	priceStr := c.FormValue("price")
	stockStr := c.FormValue("stock")
	active := c.FormValue("active") == "on"

	// Convert price from dollars to cents
	var price int32
	if priceStr != "" {
		priceFloat := 0.0
		if _, err := fmt.Sscanf(priceStr, "%f", &priceFloat); err != nil {
			return c.String(http.StatusBadRequest, "Invalid price format")
		}
		price = int32(priceFloat * 100) // Convert to cents
	}

	// Convert stock
	var stock int32
	if stockStr != "" {
		stockInt := 0
		if _, err := fmt.Sscanf(stockStr, "%d", &stockInt); err != nil {
			return c.String(http.StatusBadRequest, "Invalid stock format")
		}
		stock = int32(stockInt)
	}

	// Create product request
	req := interfaces.CreateProductRequest{
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		Active:      active,
	}

	// Send request to Freyja API
	reqBody, err := json.Marshal(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to marshal request")
	}

	resp, err := s.httpClient.Post(
		s.freyjaAPIURL+"/products",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return c.String(http.StatusInternalServerError, "Error creating product")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("API error: %s", string(body))
		return c.String(http.StatusInternalServerError, "Failed to create product")
	}

	// Return success response that closes modal and refreshes table
	c.Response().Header().Set("HX-Trigger", "productCreated")
	return c.String(http.StatusOK, "")
}

func main() {
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

	// Initialize admin server
	adminServer := NewAdminServer(freyjaAPIURL)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files (CSS, JS)
	e.Static("/static", "web/static")

	// Routes
	e.GET("/", adminServer.showProductsPage)
	e.GET("/products", adminServer.showProductsTable)
	e.GET("/products/add", adminServer.showAddProductModal)
	e.POST("/products", adminServer.createProduct)

	log.Println("🚀 Admin panel starting on :8081")
	log.Fatal(e.Start(":8081"))
}
