// internal/server/handler/option.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type OptionHandler struct {
	optionService interfaces.OptionService
}

func NewOptionHandler(optionService interfaces.OptionService) *OptionHandler {
	return &OptionHandler{
		optionService: optionService,
	}
}

// =============================================================================
// Product Option Management (Admin)
// =============================================================================

// CreateProductOption handles POST /api/v1/admin/products/{product_id}/options
func (h *OptionHandler) CreateProductOption(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	var req interfaces.CreateProductOptionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	req.ProductID = int32(productID)

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	option, err := h.optionService.CreateProductOption(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    option,
	})
}

// GetProductOptions handles GET /api/v1/admin/products/{product_id}/options
func (h *OptionHandler) GetProductOptions(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	options, err := h.optionService.GetProductOptions(ctx, int32(productID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"data":       options,
		"count":      len(options),
		"product_id": productID,
	})
}

// GetProductOption handles GET /api/v1/admin/options/{id}
func (h *OptionHandler) GetProductOption(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	option, err := h.optionService.GetProductOptionByID(ctx, int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Option not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    option,
	})
}

// UpdateProductOption handles PUT /api/v1/admin/options/{id}
func (h *OptionHandler) UpdateProductOption(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	var req interfaces.UpdateProductOptionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	option, err := h.optionService.UpdateProductOption(ctx, int32(id), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    option,
	})
}

// DeleteProductOption handles DELETE /api/v1/admin/options/{id}
func (h *OptionHandler) DeleteProductOption(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	err = h.optionService.DeleteProductOption(ctx, int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Option deleted successfully",
	})
}

// =============================================================================
// Option Value Management (Admin)
// =============================================================================

// CreateOptionValue handles POST /api/v1/admin/options/{option_id}/values
func (h *OptionHandler) CreateOptionValue(c echo.Context) error {
	ctx := c.Request().Context()

	optionID, err := strconv.ParseInt(c.Param("option_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	var req interfaces.CreateOptionValueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	req.OptionID = int32(optionID)

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	value, err := h.optionService.CreateOptionValue(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    value,
	})
}

// GetOptionValues handles GET /api/v1/admin/options/{option_id}/values
func (h *OptionHandler) GetOptionValues(c echo.Context) error {
	ctx := c.Request().Context()

	optionID, err := strconv.ParseInt(c.Param("option_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	values, err := h.optionService.GetOptionValues(ctx, int32(optionID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"data":      values,
		"count":     len(values),
		"option_id": optionID,
	})
}

// GetOptionValue handles GET /api/v1/admin/option-values/{id}
func (h *OptionHandler) GetOptionValue(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option value ID")
	}

	value, err := h.optionService.GetOptionValueByID(ctx, int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Option value not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    value,
	})
}

// UpdateOptionValue handles PUT /api/v1/admin/option-values/{id}
func (h *OptionHandler) UpdateOptionValue(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option value ID")
	}

	var req interfaces.UpdateOptionValueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	value, err := h.optionService.UpdateOptionValue(ctx, int32(id), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    value,
	})
}

// DeleteOptionValue handles DELETE /api/v1/admin/option-values/{id}
func (h *OptionHandler) DeleteOptionValue(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option value ID")
	}

	err = h.optionService.DeleteOptionValue(ctx, int32(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Option value deleted successfully",
	})
}

// =============================================================================
// Customer-Facing Option Discovery Endpoints
// =============================================================================

// GetProductOptionsForCustomers handles GET /api/v1/products/{product_id}/options
func (h *OptionHandler) GetProductOptionsForCustomers(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	options, err := h.optionService.GetAvailableOptions(ctx, int32(productID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"data":       options,
		"count":      len(options),
		"product_id": productID,
	})
}

// GetOptionCombinationsInStock handles GET /api/v1/products/{product_id}/option-combinations
func (h *OptionHandler) GetOptionCombinationsInStock(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	combinations, err := h.optionService.GetOptionCombinationsInStock(ctx, int32(productID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":          true,
		"data":             combinations,
		"count":            len(combinations),
		"product_id":       productID,
		"combinations_available": len(combinations) > 0,
	})
}

// FindVariantByOptions handles POST /api/v1/products/{product_id}/find-variant
func (h *OptionHandler) FindVariantByOptions(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	var req interfaces.FindVariantByOptionsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	req.ProductID = int32(productID)

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	variant, err := h.optionService.FindVariantByOptions(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "No variant found for the selected options")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    variant,
	})
}

// =============================================================================
// Option Analytics and Management Endpoints
// =============================================================================

// GetOptionUsageStats handles GET /api/v1/admin/options/{option_id}/usage
func (h *OptionHandler) GetOptionUsageStats(c echo.Context) error {
	ctx := c.Request().Context()

	optionID, err := strconv.ParseInt(c.Param("option_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
	}

	stats, err := h.optionService.GetOptionUsageStats(ctx, int32(optionID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"data":      stats,
		"option_id": optionID,
	})
}

// GetOptionPopularity handles GET /api/v1/admin/products/{product_id}/option-popularity
func (h *OptionHandler) GetOptionPopularity(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	// Parse optional date filters
	filters := interfaces.OptionPopularityFilters{
		ProductID: int32(productID),
	}

	if startDate := c.QueryParam("start_date"); startDate != "" {
		// Parse start_date if provided
		// You might want to add proper date parsing here
		filters.StartDate = &startDate
	}

	if endDate := c.QueryParam("end_date"); endDate != "" {
		// Parse end_date if provided
		filters.EndDate = &endDate
	}

	popularity, err := h.optionService.GetOptionPopularity(ctx, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"data":       popularity,
		"count":      len(popularity),
		"product_id": productID,
		"filters":    filters,
	})
}

// GetOrphanedOptions handles GET /api/v1/admin/options/orphaned
func (h *OptionHandler) GetOrphanedOptions(c echo.Context) error {
	ctx := c.Request().Context()

	orphanedOptions, err := h.optionService.GetOrphanedOptions(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	orphanedValues, err := h.optionService.GetOrphanedOptionValues(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"orphaned_options": orphanedOptions,
			"orphaned_values":  orphanedValues,
		},
		"counts": map[string]int{
			"options": len(orphanedOptions),
			"values":  len(orphanedValues),
		},
	})
}