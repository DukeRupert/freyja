// internal/server/handler/product.go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/dukerupert/freyja/internal/server/middleware"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	productService interfaces.ProductService
	variantService interfaces.VariantService
}

func NewProductHandler(productService interfaces.ProductService, variantService interfaces.VariantService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		variantService: variantService,
	}
}

// =============================================================================
// Customer-Facing Product Endpoints (with variant information)
// =============================================================================

// GetProducts handles GET /api/v1/products
// Now returns products with aggregated variant information
func (h *ProductHandler) GetProducts(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.GetLogger(c)

	logger.Info().Msg("Starting GetProducts request")

	// Parse query parameters for filtering
	filters := interfaces.ProductFilters{}

	// Check for 'active' filter
	if activeParam := c.QueryParam("active"); activeParam != "" {
		if active, err := strconv.ParseBool(activeParam); err == nil {
			filters.Active = &active
			logger.Debug().Bool("active_filter", active).Msg("Applied active filter")
		} else {
			logger.Warn().Str("active_param", activeParam).Msg("Invalid active parameter, ignoring")
		}
	}

	// Check for pagination
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 {
			filters.Limit = limit
			logger.Debug().Int("limit", limit).Msg("Applied limit filter")
		} else {
			logger.Warn().Str("limit_param", limitParam).Msg("Invalid limit parameter, ignoring")
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
			logger.Debug().Int("offset", offset).Msg("Applied offset filter")
		} else {
			logger.Warn().Str("offset_param", offsetParam).Msg("Invalid offset parameter, ignoring")
		}
	}

	// Check for search query
	searchQuery := c.QueryParam("search")
	if searchQuery != "" {
		logger.Debug().Str("search_query", searchQuery).Msg("Search query provided")
	}

	var products []interfaces.ProductSummary
	var err error

	logger.Info().
		Interface("filters", filters).
		Str("search_query", searchQuery).
		Msg("Fetching products from service")

	if searchQuery != "" {
		products, err = h.productService.SearchProducts(ctx, searchQuery)
		if err != nil {
			logger.Error().
				Err(err).
				Str("search_query", searchQuery).
				Msg("Failed to search products")
		} else {
			logger.Info().
				Int("result_count", len(products)).
				Str("search_query", searchQuery).
				Msg("Successfully searched products")
		}
	} else {
		products, err = h.productService.GetAll(ctx, filters)
		if err != nil {
			logger.Error().
				Err(err).
				Interface("filters", filters).
				Msg("Failed to fetch products")
		} else {
			logger.Info().
				Int("result_count", len(products)).
				Interface("filters", filters).
				Msg("Successfully fetched products")
		}
	}

	if err != nil {
		c.Logger().Error("Failed to fetch products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch products",
			"code":  "INTERNAL_ERROR",
		})
	}

	logger.Debug().Int("products_count", len(products)).Msg("Starting product transformation")

	// Transform to API-friendly format
	apiProducts := make([]map[string]interface{}, len(products))
	for i, product := range products {
		apiProducts[i] = h.productSummaryToAPI(product)
	}

	logger.Info().
		Int("total_products", len(apiProducts)).
		Bool("has_search", searchQuery != "").
		Msg("Successfully completed GetProducts request")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"products": apiProducts,
		"total":    len(apiProducts),
		"filters":  filters,
		"search":   searchQuery,
	})
}

// GetProduct handles GET /api/v1/products/:id
// Now returns product with all its variants
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

	// Get product summary with aggregated variant data
	productSummary, err := h.productService.GetByID(ctx, id)
	if err != nil {
		c.Logger().Error("Failed to get product: ", err)

		if err.Error() == "product not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product not found",
				"code":  "PRODUCT_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get all active variants for this product
	variants, err := h.variantService.GetActiveVariantsByProduct(ctx, int32(id))
	if err != nil {
		c.Logger().Error("Failed to get product variants: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product variants",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Build comprehensive product response
	response := h.productSummaryToAPI(*productSummary)
	response["variants"] = h.variantsToAPI(variants)
	response["has_variants"] = len(variants) > 0

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// GetInStockProducts handles GET /api/v1/products/in-stock
func (h *ProductHandler) GetInStockProducts(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := h.productService.GetInStock(ctx)
	if err != nil {
		c.Logger().Error("Failed to get in-stock products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve in-stock products",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Transform to API-friendly format
	apiProducts := make([]map[string]interface{}, len(products))
	for i, product := range products {
		apiProducts[i] = h.productSummaryToAPI(product)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"products": apiProducts,
		"total":    len(apiProducts),
		"filter":   "in_stock",
	})
}

// GetLowStockProducts handles GET /api/v1/products/low-stock
func (h *ProductHandler) GetLowStockProducts(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse threshold parameter (default to 10)
	threshold := int32(10)
	if thresholdParam := c.QueryParam("threshold"); thresholdParam != "" {
		if t, err := strconv.ParseInt(thresholdParam, 10, 32); err == nil && t > 0 {
			threshold = int32(t)
		}
	}

	// Get low stock variants first
	lowStockVariants, err := h.variantService.GetLowStockVariants(ctx, threshold)
	if err != nil {
		c.Logger().Error("Failed to get low-stock variants: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve low-stock products",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Group variants by product and get product summaries
	productIDs := make(map[int32]bool)
	for _, variant := range lowStockVariants {
		productIDs[variant.ProductID] = true
	}

	var products []interfaces.ProductSummary
	for productID := range productIDs {
		if product, err := h.productService.GetByID(ctx, int(productID)); err == nil {
			products = append(products, *product)
		}
	}

	// Transform to API-friendly format
	apiProducts := make([]map[string]interface{}, len(products))
	for i, product := range products {
		apiProducts[i] = h.productSummaryToAPI(product)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"products":  apiProducts,
		"total":     len(apiProducts),
		"threshold": threshold,
		"filter":    "low_stock",
	})
}

// GetProductStats handles GET /api/v1/products/stats
func (h *ProductHandler) GetProductStats(c echo.Context) error {
	ctx := c.Request().Context()

	// Get basic product counts by calling the service method
	// Note: This assumes a GetStats method exists on ProductService
	// If not, we'll gather stats manually using existing methods

	// Get all products to calculate basic stats
	allProducts, err := h.productService.GetAll(ctx, interfaces.ProductFilters{})
	if err != nil {
		c.Logger().Error("Failed to get products for stats: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get in-stock products
	inStockProducts, err := h.productService.GetInStock(ctx)
	if err != nil {
		c.Logger().Error("Failed to get in-stock products for stats: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get low stock variants for additional stats
	lowStockVariants, err := h.variantService.GetLowStockVariants(ctx, 10)
	if err != nil {
		c.Logger().Error("Failed to get low-stock variants for stats: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Calculate stats
	activeProducts := 0
	totalStock := int32(0)
	totalVariants := int32(0)

	for _, product := range allProducts {
		if product.ProductID > 0 { // Valid product
			totalStock += product.TotalStock
			totalVariants += product.TotalVariants
		}
	}

	// Count active products (those with variants in stock)
	for _, product := range inStockProducts {
		if product.HasStock {
			activeProducts++
		}
	}

	stats := map[string]interface{}{
		"total_products":      len(allProducts),
		"active_products":     activeProducts,
		"products_in_stock":   len(inStockProducts),
		"products_low_stock":  len(lowStockVariants),
		"total_stock":         totalStock,
		"total_variants":      totalVariants,
		"low_stock_threshold": 10,
		"generated_at":        time.Now().UTC(),
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// =============================================================================
// New Variant-Specific Product Endpoints
// =============================================================================

// GetProductVariants handles GET /api/v1/products/:id/variants
// Customer-facing endpoint to get all available variants for a product
func (h *ProductHandler) GetProductVariants(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid product ID",
			"code":  "INVALID_ID",
		})
	}

	// Get active variants only for customers
	variants, err := h.variantService.GetActiveVariantsByProduct(ctx, int32(productID))
	if err != nil {
		c.Logger().Error("Failed to get product variants: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve product variants",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"variants":   h.variantsToAPI(variants),
		"total":      len(variants),
		"product_id": productID,
	})
}

// SearchProductVariants handles GET /api/v1/products/variants/search
// Search across all product variants
func (h *ProductHandler) SearchProductVariants(c echo.Context) error {
	ctx := c.Request().Context()

	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Search query parameter 'q' is required",
			"code":  "MISSING_QUERY",
		})
	}

	variants, err := h.variantService.SearchVariants(ctx, query)
	if err != nil {
		c.Logger().Error("Failed to search variants: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to search variants",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"variants": h.variantsToAPI(variants),
		"total":    len(variants),
		"query":    query,
	})
}

// =============================================================================
// Admin Product Endpoints (existing but updated for clarity)
// =============================================================================

// CreateProduct handles POST /api/v1/admin/products
func (h *ProductHandler) CreateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	var req interfaces.CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	product, err := h.productService.Create(ctx, req)
	if err != nil {
		c.Logger().Error("Failed to create product: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create product",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    h.basicProductToAPI(*product),
		"message": "Product created successfully",
	})
}

// UpdateProduct handles PUT /api/v1/admin/products/:id
func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid product ID",
			"code":  "INVALID_ID",
		})
	}

	// Get existing product
	product, err := h.productService.GetBasicProductByID(ctx, int(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Product not found",
			"code":  "PRODUCT_NOT_FOUND",
		})
	}

	var req interfaces.CreateProductRequest // Using same struct for updates
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Update product fields
	product.Name = req.Name
	product.Description = stringToPgText(req.Description)
	product.Active = req.Active

	if err := h.productService.Update(ctx, product); err != nil {
		c.Logger().Error("Failed to update product: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update product",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    h.basicProductToAPI(*product),
		"message": "Product updated successfully",
	})
}

// =============================================================================
// Helper Methods for Response Formatting
// =============================================================================

// productSummaryToAPI converts ProductSummary to API response format
func (h *ProductHandler) productSummaryToAPI(product interfaces.ProductSummary) map[string]interface{} {
	return map[string]interface{}{
		"id":                product.ProductID,
		"name":              product.Name,
		"description":       product.Description,
		"total_stock":       product.TotalStock,
		"variants_in_stock": product.VariantsInStock,
		"total_variants":    product.TotalVariants,
		"min_price":         product.MinPrice,
		"max_price":         product.MaxPrice,
		"has_stock":         product.HasStock,
		"stock_status":      product.StockStatus,
		"available_options": product.AvailableOptions,
	}
}

// basicProductToAPI converts basic Product to API response format (admin use)
func (h *ProductHandler) basicProductToAPI(product interfaces.Product) map[string]interface{} {
	return map[string]interface{}{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"active":      product.Active,
		"created_at":  product.CreatedAt,
		"updated_at":  product.UpdatedAt,
	}
}

// variantsToAPI converts slice of ProductVariant to API format
func (h *ProductHandler) variantsToAPI(variants []interfaces.ProductVariant) []map[string]interface{} {
	apiVariants := make([]map[string]interface{}, len(variants))
	for i, variant := range variants {
		apiVariants[i] = map[string]interface{}{
			"id":                      variant.ID,
			"product_id":              variant.ProductID,
			"name":                    variant.Name,
			"price":                   variant.Price,
			"stock":                   variant.Stock,
			"active":                  variant.Active,
			"is_subscription":         variant.IsSubscription,
			"options_display":         variant.OptionsDisplay,
			"stripe_product_id":       variant.StripeProductID,
			"stripe_price_onetime_id": variant.StripePriceOnetimeID,
			"created_at":              variant.CreatedAt,
			"updated_at":              variant.UpdatedAt,
		}
	}
	return apiVariants
}

func stringToPgText(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  s != "",
	}
}
