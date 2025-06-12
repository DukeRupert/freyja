// internal/backend/handlers/product_handlers.go
package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/backend/client"
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
// This now just forwards the form data to the server API
func (h *ProductHandler) CreateProduct(c echo.Context) error {
	// Forward the entire form to the server's API
	// The server will handle validation, conversion, and business logic
	err := h.freyjaClient.CreateProductFromForm(c.Request().Context(), c.Request())
	if err != nil {
		log.Printf("Error creating product: %v", err)
		
		// Check if it's an HTMX request and return appropriate error
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-600">Failed to create product. Please check your input.</div>`)
		}
		return c.String(http.StatusInternalServerError, "Failed to create product")
	}

	// Return success response that closes modal and refreshes table
	c.Response().Header().Set("HX-Trigger", "productCreated")
	return c.String(http.StatusOK, "")
}

// UpdateProduct handles product updates
func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	productID := c.Param("id")
	
	err := h.freyjaClient.UpdateProductFromForm(c.Request().Context(), productID, c.Request())
	if err != nil {
		log.Printf("Error updating product: %v", err)
		
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-600">Failed to update product. Please check your input.</div>`)
		}
		return c.String(http.StatusInternalServerError, "Failed to update product")
	}

	c.Response().Header().Set("HX-Trigger", "productUpdated")
	return c.String(http.StatusOK, "")
}

// DeleteProduct handles product deletion
func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	productID := c.Param("id")
	
	err := h.freyjaClient.DeleteProduct(c.Request().Context(), productID)
	if err != nil {
		log.Printf("Error deleting product: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to delete product")
	}

	c.Response().Header().Set("HX-Trigger", "productDeleted")
	return c.String(http.StatusOK, "")
}