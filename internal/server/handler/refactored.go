package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func HandleGetProducts(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger.Info().Msg("Starting GetProducts request")

		filters := database.GetProductsParams{
			Limit:  parseIntParam(c.QueryParam("limit"), 10),
			Offset: parseIntParam(c.QueryParam("offset"), 0),
			Active: parseBoolParam(c.QueryParam("active"), true),
		}

		products, err := db.Queries.GetProducts(ctx, filters)
		if err != nil {
			logger.Error().
				Err(err).
				Interface("filters", filters).
				Msg("Failed to fetch products")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to fetch products",
				"code":  "INTERNAL_ERROR",
			})
		} else {
			logger.Info().
				Int("result_count", len(products)).
				Interface("filters", filters).
				Msg("Successfully fetched products")
		}

		logger.Info().
			Int("total_products", len(products)).
			Msg("Successfully completed GetProducts request")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"products": products,
			"total":    len(products),
			"filters":  filters,
		})
	}
}

func HandleGetProduct(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse product ID
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID",
				"code":  "INVALID_ID",
			})
		}

		product, err := db.Queries.GetProduct(ctx, int32(id))
		if err != nil {
			c.Logger().Error("Failed to get product: ", err)
			if err == pgx.ErrNoRows {
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

		rows, err := db.Queries.GetActiveVariantsByProduct(ctx, product.ID)
		if err != nil {
			c.Logger().Error("Failed to get product variants: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product variants",
				"code":  "INTERNAL_ERROR",
			})
		}

		// Transform to API-friendly format
		variants := convertProductVariantsToJSON(rows)

		logger.Info().
			Int("total_products", len(variants)).
			Msg("Successfully completed GetProducts request")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"product":  product,
			"variants": variants,
			"total":    len(variants),
		})

	}
}

func HandleCreateProduct(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type CreateProductRequest struct {
		Name        string      `json:"name" validate:"required,min=1,max=255"`
		Description pgtype.Text `json:"description" validate:"max=1000"`
		Active      bool        `json:"active"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req CreateProductRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid JSON format in request body",
				"code":  "INVALID_JSON",
			})
		}

		if err := c.Validate(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
		}

		// Sanitize and validate name
		trimmedName := strings.TrimSpace(req.Name)
		if trimmedName == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product name cannot be empty or whitespace only",
				"code":  "INVALID_NAME",
			})
		}

		// Check for name collision
		_, err := db.Queries.GetProductByName(ctx, trimmedName)
		if err != nil && err != pgx.ErrNoRows {
			logger.Error().
				Err(err).
				Str("product_name", trimmedName).
				Msg("Failed to check product name uniqueness")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate product name",
				"code":  "VALIDATION_ERROR",
			})
		}

		if err == nil {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "A product with this name already exists",
				"code":  "NAME_CONFLICT",
			})
		}

		// Create product with sanitized data
		product, err := db.Queries.CreateProduct(ctx, database.CreateProductParams{
			Name:        trimmedName,
			Description: req.Description,
			Active:      req.Active,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("product_name", trimmedName).
				Bool("active", req.Active).
				Msg("Failed to create product")

			// Handle specific database errors
			if strings.Contains(err.Error(), "duplicate key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Product name must be unique",
					"code":  "DUPLICATE_NAME",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create product",
				"code":  "CREATION_FAILED",
			})
		}

		logger.Info().
			Int32("product_id", product.ID).
			Str("name", product.Name).
			Bool("active", product.Active).
			Msg("Product created successfully")

		// TODO: Publish event
		// eventBus.Publish("product.created", ProductCreatedEvent{
		//     ProductID: product.ID,
		//     Name:      product.Name,
		//     Active:    product.Active,
		// })

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"success": true,
			"data":    product,
			"message": "Product created successfully",
		})
	}
}

func HandleUpdateProduct(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type UpdateParams struct {
		Name        string      `db:"name" json:"name" validate:"required,min=1,max=255"`
		Description pgtype.Text `db:"description" json:"description" validate:"max=1000"`
		Active      bool        `db:"active" json:"active"`
	}
	
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse product ID with better validation
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID. Must be a positive integer",
				"code":  "INVALID_ID",
			})
		}

		var req UpdateParams
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid JSON format in request body",
				"code":  "INVALID_JSON",
			})
		}

		if err := c.Validate(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
		}

		// Check for name collision (excluding current product)
		if existingProduct, err := db.Queries.GetProductByName(ctx, req.Name); err == nil {
			if existingProduct.ID != int32(id) {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "A product with this name already exists",
					"code":  "NAME_CONFLICT",
				})
			}
		} else if err != pgx.ErrNoRows {
			logger.Error().Err(err).Msg("Failed to check product name uniqueness")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate product name",
				"code":  "VALIDATION_ERROR",
			})
		}

		product, err := db.Queries.UpdateProduct(ctx, database.UpdateProductParams{
			ID:          int32(id),
			Name:        req.Name,
			Description: req.Description,
			Active:      req.Active,
		})
		if err != nil {
			logger.Error().Err(err).Int64("product_id", id).Msg("Failed to update product")
			
			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Product not found",
					"code":  "PRODUCT_NOT_FOUND",
				})
			}
			
			// Check for specific database constraints
			if strings.Contains(err.Error(), "duplicate key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Product name must be unique",
					"code":  "DUPLICATE_NAME",
				})
			}
			
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update product",
				"code":  "UPDATE_FAILED",
			})
		}

		logger.Info().
			Int64("product_id", id).
			Str("name", req.Name).
			Bool("active", req.Active).
			Msg("Product updated successfully")

		// TODO: Publish event

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    product,
			"message": "Product updated successfully",
		})
	}
}

func HandleGetOption(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		option_id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid option ID")
		}
		id := int32(option_id)

		option, err := db.Queries.GetProductOption(ctx, id)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get option")
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
					"error": "option not found",
					"code":  "INTERNAL_ERROR",
				})
			}
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to retrieve option",
				"code":  "INTERNAL_ERROR",
			})
		}

		return c.JSON(http.StatusOK, &option)
	}
}

func HandleGetProductOptions(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
		}
		product_id := int32(id)

		// check if product exists
		_, err = db.Queries.GetProduct(ctx, product_id)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get product")
			if err == pgx.ErrNoRows {
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
					"error": "product not found",
					"code":  "INTERNAL_ERROR",
				})
			}
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to retrieve product",
				"code":  "INTERNAL_ERROR",
			})
		}

		options, err := db.Queries.GetProductOptions(ctx, product_id)
		if err != nil {
			logger.Error().Err(err).Msg("failed to retrieve options")
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to retrieve product",
				"code":  "INTERNAL_ERROR",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"endpoint":   "/products/:id/option",
			"product_id": product_id,
			"options":    options,
		})
	}
}

func HandleCreateProductOption(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type CreateProductOptionRequest struct {
		OptionKey string `json:"option_key" validate:"required,min=1,max=50"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate product ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID. Must be a positive integer",
				"code":  "INVALID_PRODUCT_ID",
			})
		}
		productID := int32(id)

		// Parse and validate request body
		var req CreateProductOptionRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid JSON format in request body",
				"code":  "INVALID_JSON",
			})
		}

		if err := c.Validate(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
		}

		// Sanitize and validate option key
		trimmedOptionKey := strings.TrimSpace(req.OptionKey)
		if trimmedOptionKey == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option key cannot be empty or whitespace only",
				"code":  "INVALID_OPTION_KEY",
			})
		}

		// Normalize option key (lowercase for consistency)
		normalizedOptionKey := strings.ToLower(trimmedOptionKey)

		// Check if product exists
		product, err := db.Queries.GetProduct(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to get product")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Product not found",
					"code":  "PRODUCT_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product",
				"code":  "PRODUCT_RETRIEVAL_FAILED",
			})
		}

		// Check if product is active (optional business rule)
		if !product.Active {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Cannot add options to inactive product",
				"code":  "PRODUCT_INACTIVE",
			})
		}

		// Check if option key already exists for this product
		existingOptions, err := db.Queries.GetProductOptions(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to check existing options")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate option uniqueness",
				"code":  "VALIDATION_ERROR",
			})
		}

		for _, existingOption := range existingOptions {
			if strings.ToLower(existingOption.OptionKey) == normalizedOptionKey {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "An option with this key already exists for this product",
					"code":  "OPTION_KEY_CONFLICT",
				})
			}
		}

		// Create the option
		option, err := db.Queries.CreateProductOption(ctx, database.CreateProductOptionParams{
			ProductID: productID,
			OptionKey: normalizedOptionKey,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Str("option_key", normalizedOptionKey).
				Msg("Failed to create product option")

			// Handle specific database errors
			if strings.Contains(err.Error(), "duplicate key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Option key must be unique for this product",
					"code":  "DUPLICATE_OPTION_KEY",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create product option",
				"code":  "OPTION_CREATION_FAILED",
			})
		}

		logger.Info().
			Int32("product_id", productID).
			Int32("option_id", option.ID).
			Str("option_key", option.OptionKey).
			Msg("Product option created successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option.created", ProductOptionCreatedEvent{
		//     ProductID: productID,
		//     OptionID:  option.ID,
		//     OptionKey: option.OptionKey,
		// })

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"success": true,
			"data":    option,
			"message": "Product option created successfully",
		})
	}
}

// helpers
func convertProductStockSummarytoJSON(summaries []database.ProductStockSummary) []map[string]interface{} {
	result := make([]map[string]interface{}, len(summaries))

	for i, summary := range summaries {
		result[i] = map[string]interface{}{
			"product_id":        summary.ProductID,
			"name":              summary.Name,
			"description":       convertPgText(summary.Description),
			"product_active":    summary.ProductActive,
			"total_stock":       summary.TotalStock,
			"variants_in_stock": summary.VariantsInStock,
			"total_variants":    summary.TotalVariants,
			"min_price":         summary.MinPrice,
			"max_price":         summary.MaxPrice,
			"has_stock":         summary.HasStock,
			"stock_status":      summary.StockStatus,
			"available_options": convertJSONBytes(summary.AvailableOptions),
			"last_stock_update": summary.LastStockUpdate,
		}
	}

	return result
}

func convertProductVariantsToJSON(variants []database.ProductVariants) []map[string]interface{} {
	result := make([]map[string]interface{}, len(variants))

	for i, variant := range variants {
		result[i] = map[string]interface{}{
			"id":                      variant.ID,
			"product_id":              variant.ProductID,
			"name":                    variant.Name,
			"price":                   variant.Price,
			"stock":                   variant.Stock,
			"active":                  variant.Active,
			"is_subscription":         variant.IsSubscription,
			"archived_at":             convertPgTimestamp(variant.ArchivedAt),
			"created_at":              variant.CreatedAt,
			"updated_at":              variant.UpdatedAt,
			"stripe_product_id":       convertPgText(variant.StripeProductID),
			"stripe_price_onetime_id": convertPgText(variant.StripePriceOnetimeID),
			"stripe_price_14day_id":   convertPgText(variant.StripePrice14dayID),
			"stripe_price_21day_id":   convertPgText(variant.StripePrice21dayID),
			"stripe_price_30day_id":   convertPgText(variant.StripePrice30dayID),
			"stripe_price_60day_id":   convertPgText(variant.StripePrice60dayID),
			"options_display":         convertPgText(variant.OptionsDisplay),
		}
	}

	return result
}

func convertPgTimestamp(pgTimestamp pgtype.Timestamp) *time.Time {
	if !pgTimestamp.Valid {
		return nil
	}
	return &pgTimestamp.Time
}

func convertPgText(pgText pgtype.Text) *string {
	if !pgText.Valid {
		return nil
	}
	return &pgText.String
}

func convertJSONBytes(jsonBytes []byte) []map[string]interface{} {
	if len(jsonBytes) == 0 {
		return []map[string]interface{}{}
	}

	var options []map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &options); err != nil {
		// If parsing fails, return empty slice instead of nil
		return []map[string]interface{}{}
	}

	return options
}

func parseIntParam(param string, defaultValue int) int32 {
	if val, err := strconv.Atoi(param); err == nil {
		return int32(val)
	}
	return int32(defaultValue)
}

func parseBoolParam(param string, defaultValue bool) bool {
	if val, err := strconv.ParseBool(param); err == nil {
		return val
	}
	return defaultValue
}
