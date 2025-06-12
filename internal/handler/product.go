// internal/handler/product.go
package handler

import (
	"fmt"

	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/web/admin/views"
	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	productService interfaces.ProductService
}

func NewProductHandler(productService interfaces.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// GetProducts handles GET /api/v1/products
func (h *ProductHandler) GetProducts(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters for filtering
	filters := interfaces.ProductFilters{}

	// Check for 'active' filter
	if activeParam := c.QueryParam("active"); activeParam != "" {
		if active, err := strconv.ParseBool(activeParam); err == nil {
			filters.Active = &active
		}
	}

	// Check for pagination
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Check for search query
	searchQuery := c.QueryParam("search")

	var products []interfaces.Product
	var err error

	if searchQuery != "" {
		products, err = h.productService.SearchProducts(ctx, searchQuery)
	} else {
		products, err = h.productService.GetAll(ctx, filters)
	}

	if err != nil {
		c.Logger().Error("Failed to fetch products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch products",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Convert to API format
	apiProducts := make([]map[string]interface{}, len(products))
	for i, p := range products {
		apiProducts[i] = h.productToAPI(p)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": apiProducts,
		"total":    len(apiProducts),
		"filters":  filters,
	})
}

// GetProduct handles GET /api/v1/products/:id
func (h *ProductHandler) GetProduct(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse product ID
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid product ID",
			"code":  "INVALID_ID",
		})
	}

	// Fetch product
	product, err := h.productService.GetByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("Failed to fetch product %d: %v", id, err)

		// Check if it's a not found error
		if err.Error() == "product not found" ||
			err.Error() == "no rows in result set" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product not found",
				"code":  "PRODUCT_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch product",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, h.productToAPI(*product))
}

// GetInStockProducts handles GET /api/v1/products/in-stock
func (h *ProductHandler) GetInStockProducts(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := h.productService.GetInStock(ctx)
	if err != nil {
		c.Logger().Error("Failed to fetch in-stock products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch in-stock products",
			"code":  "INTERNAL_ERROR",
		})
	}

	apiProducts := make([]map[string]interface{}, len(products))
	for i, p := range products {
		apiProducts[i] = h.productToAPI(p)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": apiProducts,
		"total":    len(apiProducts),
	})
}

// GetLowStockProducts handles GET /api/v1/products/low-stock
func (h *ProductHandler) GetLowStockProducts(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse threshold parameter (default to 10)
	threshold := 10
	if thresholdParam := c.QueryParam("threshold"); thresholdParam != "" {
		if t, err := strconv.Atoi(thresholdParam); err == nil && t > 0 {
			threshold = t
		}
	}

	products, err := h.productService.GetLowStock(ctx, threshold)
	if err != nil {
		c.Logger().Error("Failed to fetch low-stock products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch low-stock products",
			"code":  "INTERNAL_ERROR",
		})
	}

	apiProducts := make([]map[string]interface{}, len(products))
	for i, p := range products {
		apiProducts[i] = h.productToAPI(p)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products":  apiProducts,
		"total":     len(apiProducts),
		"threshold": threshold,
	})
}

// GetProductStats handles GET /api/v1/products/stats
func (h *ProductHandler) GetProductStats(c echo.Context) error {
	ctx := c.Request().Context()

	stats, err := h.productService.GetStats(ctx)
	if err != nil {
		c.Logger().Error("Failed to fetch product stats: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch product statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// CreateProduct handles POST /api/v1/admin/products
func (h *ProductHandler) CreateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	var formReq interfaces.CreateProductFormRequest
	if err := c.Bind(&formReq); err != nil {
		c.Logger().Errorf("CreateProduct - Binding error: %v", err)

		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, fmt.Sprintf(`<div class="text-red-600">Invalid request format: %v</div>`, err))
		}
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Convert form request to service request
	req := formReq.ToCreateProductRequest()

	product, err := h.productService.CreateProduct(ctx, req)
	if err != nil {
		c.Logger().Errorf("Failed to create product: %v", err)

		// Check if this is an HTMX request
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-600">Failed to create product</div>`)
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create product",
			"code":  "CREATION_FAILED",
		})
	}

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "productCreated")
		c.Response().Header().Set("HX-Reswap", "afterbegin")
		c.Response().Header().Set("HX-Retarget", "#products-table tbody")

		component := views.ProductRow(*product)
		if err := component.Render(ctx, c.Response().Writer); err != nil {
			c.Logger().Errorf("Failed to render product row: %v", err)
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-600">Failed to render product</div>`)
		}
		return nil
	}

	// Default JSON response for API clients
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Product created successfully",
		"product": product,
	})
}

// UpdateProduct handles PUT /api/v1/admin/products/:id
func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid product ID",
			"code":  "INVALID_ID",
		})
	}

	var req interfaces.UpdateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	product, err := h.productService.UpdateProduct(ctx, int32(id), req)
	if err != nil {
		c.Logger().Errorf("Failed to update product %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update product",
			"code":  "UPDATE_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Product updated successfully",
		"product": product,
	})
}

// Helper method to convert Product to API format
func (h *ProductHandler) productToAPI(product interfaces.Product) map[string]interface{} {
	return map[string]interface{}{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"price":       product.Price,
		"stock":       product.Stock,
		"active":      product.Active,
		// Add computed fields
		"price_formatted": formatPrice(product.Price),
		"in_stock":        product.Stock > 0,
		"availability":    getAvailabilityStatus(product.Stock),
	}
}

// Helper function to format price in cents to dollars
func formatPrice(priceInCents int32) string {
	dollars := float64(priceInCents) / 100
	return "$" + strconv.FormatFloat(dollars, 'f', 2, 64)
}

// Helper function to get availability status
func getAvailabilityStatus(stock int32) string {
	switch {
	case stock == 0:
		return "out_of_stock"
	case stock <= 5:
		return "low_stock"
	case stock <= 10:
		return "limited"
	default:
		return "in_stock"
	}
}
