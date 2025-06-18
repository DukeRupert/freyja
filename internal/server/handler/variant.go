// internal/server/handler/variant.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type VariantHandler struct {
	variantService interfaces.VariantService
}

func NewVariantHandler(variantService interfaces.VariantService) *VariantHandler {
	return &VariantHandler{
		variantService: variantService,
	}
}

// =============================================================================
// Admin Variant Management Endpoints
// =============================================================================

// CreateVariant creates a new product variant
// POST /api/v1/admin/variants
func (h *VariantHandler) CreateVariant(c echo.Context) error {
	var req interfaces.CreateVariantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.variantService.Create(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    variant,
	})
}

// GetVariant retrieves a specific variant by ID
// GET /api/v1/admin/variants/{id}
func (h *VariantHandler) GetVariant(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	variant, err := h.variantService.GetByIDWithOptions(c.Request().Context(), int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Variant not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    variant,
	})
}

// UpdateVariant updates an existing variant
// PUT /api/v1/admin/variants/{id}
func (h *VariantHandler) UpdateVariant(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	var req interfaces.UpdateVariantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.variantService.Update(c.Request().Context(), int32(id), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    variant,
	})
}

// ArchiveVariant archives a variant (soft delete)
// DELETE /api/v1/admin/variants/{id}
func (h *VariantHandler) ArchiveVariant(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	variant, err := h.variantService.Archive(c.Request().Context(), int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Variant archived successfully",
		"data":    variant,
	})
}

// ActivateVariant activates a variant
// POST /api/v1/admin/variants/{id}/activate
func (h *VariantHandler) ActivateVariant(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	variant, err := h.variantService.Activate(c.Request().Context(), int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Variant activated successfully",
		"data":    variant,
	})
}

// DeactivateVariant deactivates a variant
// POST /api/v1/admin/variants/{id}/deactivate
func (h *VariantHandler) DeactivateVariant(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	variant, err := h.variantService.Deactivate(c.Request().Context(), int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Variant deactivated successfully",
		"data":    variant,
	})
}

// =============================================================================
// Stock Management Endpoints
// =============================================================================

// UpdateVariantStock updates variant stock levels
// PUT /api/v1/admin/variants/{id}/stock
func (h *VariantHandler) UpdateVariantStock(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	var req struct {
		Stock int32 `json:"stock" validate:"min=0"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.variantService.UpdateStock(c.Request().Context(), int32(id), req.Stock)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Stock updated successfully",
		"data":    variant,
	})
}

// IncrementVariantStock increments variant stock
// POST /api/v1/admin/variants/{id}/stock/increment
func (h *VariantHandler) IncrementVariantStock(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	var req struct {
		Delta int32 `json:"delta" validate:"required,min=1"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.variantService.IncrementStock(c.Request().Context(), int32(id), req.Delta)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Stock incremented successfully",
		"data":    variant,
	})
}

// DecrementVariantStock decrements variant stock
// POST /api/v1/admin/variants/{id}/stock/decrement
func (h *VariantHandler) DecrementVariantStock(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	var req struct {
		Delta int32 `json:"delta" validate:"required,min=1"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.variantService.DecrementStock(c.Request().Context(), int32(id), req.Delta)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Stock decremented successfully",
		"data":    variant,
	})
}

// =============================================================================
// Product Variant Listing Endpoints
// =============================================================================

// GetVariantsByProduct retrieves all variants for a specific product
// GET /api/v1/admin/products/{product_id}/variants
func (h *VariantHandler) GetVariantsByProduct(c echo.Context) error {
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	variants, err := h.variantService.GetVariantsByProduct(c.Request().Context(), int32(productID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    variants,
		"count":   len(variants),
	})
}

// GetLowStockVariants retrieves variants with low stock levels
// GET /api/v1/admin/variants/low-stock
func (h *VariantHandler) GetLowStockVariants(c echo.Context) error {
	threshold := int32(10) // Default threshold
	
	if thresholdParam := c.QueryParam("threshold"); thresholdParam != "" {
		if t, err := strconv.ParseInt(thresholdParam, 10, 32); err == nil {
			threshold = int32(t)
		}
	}

	variants, err := h.variantService.GetLowStockVariants(c.Request().Context(), threshold)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"data":      variants,
		"count":     len(variants),
		"threshold": threshold,
	})
}

// SearchVariants searches variants by name or options
// GET /api/v1/admin/variants/search
func (h *VariantHandler) SearchVariants(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Search query parameter 'q' is required")
	}

	variants, err := h.variantService.SearchVariants(c.Request().Context(), query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    variants,
		"count":   len(variants),
		"query":   query,
	})
}

// CheckVariantAvailability checks if a variant has enough stock for a specific quantity
// GET /api/v1/admin/variants/{id}/availability
func (h *VariantHandler) CheckVariantAvailability(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant ID")
	}

	quantity := int32(1) // Default quantity
	if quantityParam := c.QueryParam("quantity"); quantityParam != "" {
		if q, err := strconv.ParseInt(quantityParam, 10, 32); err == nil && q > 0 {
			quantity = int32(q)
		}
	}

	available, err := h.variantService.CheckAvailability(c.Request().Context(), int32(id), quantity)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"available": available,
		"quantity":  quantity,
	})
}