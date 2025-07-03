package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
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

		// Get product summary with aggregated variant data
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
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req database.CreateProductParams
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

		// Validate that the product name is unique
		_, err := db.Queries.GetProductByName(ctx, req.Name)
		if err != nil && err != pgx.ErrNoRows {
			// Database error (not "no rows found")
			logger.Error().Err(err).Msg("failed to check if product name already exists")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to validate product name",
				"code":  "INTERNAL_ERROR",
			})
		}

		if err == nil {
			// Product with this name already exists (no error = found a product)
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "product with this name already exists",
				"code":  "NAME_CONFLICT",
			})
		}

		// Create product
		product, err := db.Queries.CreateProduct(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("failed to create product")
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to create product",
				"code":  "INTERNAL_ERROR",
			})
		}

		// TODO: publish event
		return c.JSON(http.StatusCreated, product)
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

func HandleGetProductOption(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
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

		options, err := db.Queries.GetProductOptionsByProduct(ctx, product_id)
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
		OptionKey string `json:"option_key" form:"option_key" validate:"required,min=1,max=50"`
	}

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

		var req CreateProductOptionRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}

		if err := c.Validate(&req); err != nil {
			logger.Error().Err(err).Interface("req", req).Msg("validation failed")
			return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
				"error": "invalid request body",
				"code":  "BAD_REQUEST",
			})
		}

		// create option
		params := database.CreateProductOptionParams{
			ProductID: product_id,
			OptionKey: req.OptionKey,
		}

		option, err := db.Queries.CreateProductOption(ctx, params)
		if err != nil {
			logger.Error().Err(err).Msg("failed to create option")
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
				"error": "failed to create product",
				"code":  "INTERNAL_ERROR",
			})
		}

		// TODO: publish event

		return c.JSON(http.StatusCreated, &option)
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
