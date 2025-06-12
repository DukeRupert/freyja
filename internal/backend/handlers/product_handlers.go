// internal/admin/handlers/product_handlers.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/backend/client"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/dukerupert/freyja/internal/backend/templates"

	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	freyjaClient *client.FreyjaClient
}

func NewProductHandler(freyjaClient *client.FreyjaClient) *ProductHandler {
	return &ProductHandler{
		freyjaClient: freyjaClient,
	}
}

// ShowProductsPage renders the full products page
func (h *ProductHandler) ShowProductsPage(c echo.Context) error {
	products, err := h.freyjaClient.GetProducts(c.Request().Context())
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return c.String(http.StatusInternalServerError, "Error fetching products")
	}

	component := views.ProductsPage(products)
	return component.Render(context.Background(), c.Response().Writer)
}

// ShowProductsTable renders just the products table (for HTMX updates)
func (h *ProductHandler) ShowProductsTable(c echo.Context) error {
	products, err := h.freyjaClient.GetProducts(c.Request().Context())
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return c.String(http.StatusInternalServerError, "Error fetching products")
	}

	component := views.ProductsTable(products)
	return component.Render(context.Background(), c.Response().Writer)
}

// ShowAddProductModal renders the add product modal
func (h *ProductHandler) ShowAddProductModal(c echo.Context) error {
	component := views.AddProductModal()
	return component.Render(context.Background(), c.Response().Writer)
}

// CreateProduct handles product creation from form submission
func (h *ProductHandler) CreateProduct(c echo.Context) error {
	// Parse form data
	name := c.FormValue("name")
	description := c.FormValue("description")
	priceStr := c.FormValue("price")
	stockStr := c.FormValue("stock")
	active := c.FormValue("active") == "on"

	// Validate required fields
	if name == "" {
		return c.String(http.StatusBadRequest, "Product name is required")
	}

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
	_, err := h.freyjaClient.CreateProduct(c.Request().Context(), req)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to create product")
	}

	// Return success response that closes modal and refreshes table
	c.Response().Header().Set("HX-Trigger", "productCreated")
	return c.String(http.StatusOK, "")
}

// UpdateProduct handles product updates (for future use)
func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	// Parse product ID from URL
	idStr := c.Param("id")
	var id int32
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return c.String(http.StatusBadRequest, "Invalid product ID")
	}

	// Parse form data (similar to CreateProduct)
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
		price = int32(priceFloat * 100)
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

	// Create update request
	req := interfaces.UpdateProductRequest{
		Name:        &name,
		Description: &description,
		Price:       &price,
		Stock:       &stock,
		Active:      &active,
	}

	// Send update request to Freyja API
	_, err := h.freyjaClient.UpdateProduct(c.Request().Context(), id, req)
	if err != nil {
		log.Printf("Error updating product: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to update product")
	}

	// Return success response
	c.Response().Header().Set("HX-Trigger", "productUpdated")
	return c.String(http.StatusOK, "")
}

// DeleteProduct handles product deletion (for future use)
func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	// Parse product ID from URL
	idStr := c.Param("id")
	var id int32
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return c.String(http.StatusBadRequest, "Invalid product ID")
	}

	// Send delete request to Freyja API
	err := h.freyjaClient.DeleteProduct(c.Request().Context(), id)
	if err != nil {
		log.Printf("Error deleting product: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to delete product")
	}

	// Return success response
	c.Response().Header().Set("HX-Trigger", "productDeleted")
	return c.String(http.StatusOK, "")
}