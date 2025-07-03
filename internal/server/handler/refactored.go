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
		Description string `json:"description" validate:"max=1000"`
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

		// Convert string to pgtype.Text for database
		description := pgtype.Text{
			String: strings.TrimSpace(req.Description),
			Valid:  req.Description != "",
		}

		// Create product with sanitized data
		product, err := db.Queries.CreateProduct(ctx, database.CreateProductParams{
			Name:        trimmedName,
			Description: description,
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
		Name        string `json:"name" validate:"required,min=1,max=255"`
		Description string `json:"description" validate:"max=1000"`  // Changed to string
		Active      bool   `json:"active"`
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

		// Trim whitespace from name
		trimmedName := strings.TrimSpace(req.Name)
		if trimmedName == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product name cannot be empty or whitespace only",
				"code":  "INVALID_NAME",
			})
		}

		// Check for name collision (excluding current product)
		if existingProduct, err := db.Queries.GetProductByName(ctx, trimmedName); err == nil {
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

		// Convert string to pgtype.Text
		var description pgtype.Text
		if req.Description != "" {
			description = pgtype.Text{String: req.Description, Valid: true}
		} else {
			description = pgtype.Text{Valid: false}
		}

		product, err := db.Queries.UpdateProduct(ctx, database.UpdateProductParams{
			ID:          int32(id),
			Name:        trimmedName,
			Description: description,
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
			Str("name", trimmedName).
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

func HandleDeleteProduct(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate product ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID. Must be a positive integer",
				"code":  "INVALID_ID",
			})
		}
		productID := int32(id)

		// Check if product exists before attempting deletion
		product, err := db.Queries.GetProduct(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to get product for deletion")

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

		// Check if product has variants (business rule - prevent deletion if variants exist)
		variants, err := db.Queries.GetVariantsByProduct(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to check product variants")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate product deletion",
				"code":  "VALIDATION_ERROR",
			})
		}

		if len(variants) > 0 {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot delete product with existing variants. Delete variants first",
				"code":  "HAS_VARIANTS",
			})
		}

		// Check if product has options (business rule - prevent deletion if options exist)
		options, err := db.Queries.GetProductOptions(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to check product options")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate product deletion",
				"code":  "VALIDATION_ERROR",
			})
		}

		if len(options) > 0 {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot delete product with existing options. Delete options first",
				"code":  "HAS_OPTIONS",
			})
		}

		// Perform the deletion
		err = db.Queries.DeleteProduct(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Str("product_name", product.Name).
				Msg("Failed to delete product")

			// Check for foreign key constraints
			if strings.Contains(err.Error(), "foreign key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Cannot delete product due to existing references",
					"code":  "FOREIGN_KEY_CONSTRAINT",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to delete product",
				"code":  "DELETION_FAILED",
			})
		}

		logger.Info().
			Int32("product_id", productID).
			Str("product_name", product.Name).
			Bool("was_active", product.Active).
			Msg("Product deleted successfully")

		// TODO: Publish event
		// eventBus.Publish("product.deleted", ProductDeletedEvent{
		//     ProductID: productID,
		//     Name:      product.Name,
		//     DeletedBy: getUserID(c),
		// })

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Product deleted successfully",
			"data": map[string]interface{}{
				"deleted_product": map[string]interface{}{
					"id":   product.ID,
					"name": product.Name,
				},
			},
		})
	}
}

func HandleGetProductOptions(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	type OptionWithValues struct {
		ID           int32                           `json:"id"`
		ProductID    int32                           `json:"product_id"`
		OptionKey    string                          `json:"option_key"`
		CreatedAt    time.Time                       `json:"created_at"`
		OptionValues []database.ProductOptionValues  `json:"option_values"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate product ID
		productID, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || productID <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID. Must be a positive integer",
				"code":  "INVALID_PRODUCT_ID",
			})
		}
		id := int32(productID)

		// Check if product exists
		_, err = db.Queries.GetProduct(ctx, id)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", id).
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

		// Get all options for this product
		options, err := db.Queries.GetProductOptions(ctx, id)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", id).
				Msg("Failed to get product options")

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product options",
				"code":  "OPTIONS_RETRIEVAL_FAILED",
			})
		}

		// Build response with option values included
		optionsWithValues := make([]OptionWithValues, len(options))
		for i, option := range options {
			// Get values for each option
			values, err := db.Queries.GetProductOptionValues(ctx, option.ID)
			if err != nil {
				logger.Warn().
					Err(err).
					Int32("option_id", option.ID).
					Msg("Failed to get option values, using empty array")
				values = []database.ProductOptionValues{} // Empty slice on error
			}

			optionsWithValues[i] = OptionWithValues{
				ID:           option.ID,
				ProductID:    option.ProductID,
				OptionKey:    option.OptionKey,
				CreatedAt:    option.CreatedAt,
				OptionValues: values,
			}
		}

		logger.Debug().
			Int32("product_id", id).
			Int("options_count", len(optionsWithValues)).
			Msg("Product options with values retrieved successfully")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    optionsWithValues,
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

func HandleUpdateProductOption(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type UpdateProductOptionRequest struct {
		OptionKey string `json:"option_key" validate:"required,min=1,max=50"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate option ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option ID. Must be a positive integer",
				"code":  "INVALID_OPTION_ID",
			})
		}
		optionID := int32(id)

		// Parse and validate request body
		var req UpdateProductOptionRequest
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

		// Check if option exists
		existingOption, err := db.Queries.GetProductOption(ctx, optionID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Msg("Failed to get product option for update")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Product option not found",
					"code":  "OPTION_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product option",
				"code":  "OPTION_RETRIEVAL_FAILED",
			})
		}

		// Check for option key collision within the same product (excluding current option)
		existingOptions, err := db.Queries.GetProductOptions(ctx, existingOption.ProductID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", existingOption.ProductID).
				Msg("Failed to check existing options for collision")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate option key uniqueness",
				"code":  "VALIDATION_ERROR",
			})
		}

		for _, option := range existingOptions {
			if option.ID != optionID && strings.ToLower(option.OptionKey) == normalizedOptionKey {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "An option with this key already exists for this product",
					"code":  "OPTION_KEY_CONFLICT",
				})
			}
		}

		// Update the option
		updatedOption, err := db.Queries.UpdateProductOption(ctx, database.UpdateProductOptionParams{
			ID:        optionID,
			OptionKey: normalizedOptionKey,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Str("option_key", normalizedOptionKey).
				Int32("product_id", existingOption.ProductID).
				Msg("Failed to update product option")

			// Check for specific database constraints
			if strings.Contains(err.Error(), "duplicate key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Option key must be unique for this product",
					"code":  "DUPLICATE_OPTION_KEY",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update product option",
				"code":  "UPDATE_FAILED",
			})
		}

		logger.Info().
			Int32("option_id", optionID).
			Str("old_option_key", existingOption.OptionKey).
			Str("new_option_key", updatedOption.OptionKey).
			Int32("product_id", updatedOption.ProductID).
			Msg("Product option updated successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option.updated", ProductOptionUpdatedEvent{
		//     OptionID:     optionID,
		//     ProductID:    updatedOption.ProductID,
		//     OldOptionKey: existingOption.OptionKey,
		//     NewOptionKey: updatedOption.OptionKey,
		//     UpdatedBy:    getUserID(c),
		// })

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    updatedOption,
			"message": "Product option updated successfully",
		})
	}
}

func HandleDeleteProductOption(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate option ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option ID. Must be a positive integer",
				"code":  "INVALID_OPTION_ID",
			})
		}
		optionID := int32(id)

		// Check if option exists before attempting deletion
		option, err := db.Queries.GetProductOption(ctx, optionID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Msg("Failed to get product option for deletion")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Product option not found",
					"code":  "OPTION_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product option",
				"code":  "OPTION_RETRIEVAL_FAILED",
			})
		}

		// Check if option has values (business rule - prevent deletion if values exist)
		optionValues, err := db.Queries.GetProductOptionValues(ctx, optionID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Msg("Failed to check option values")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate option deletion",
				"code":  "VALIDATION_ERROR",
			})
		}

		if len(optionValues) > 0 {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot delete option with existing values. Delete option values first",
				"code":  "HAS_OPTION_VALUES",
			})
		}

		// Perform the deletion
		err = db.Queries.DeleteProductOption(ctx, optionID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Str("option_key", option.OptionKey).
				Int32("product_id", option.ProductID).
				Msg("Failed to delete product option")

			// Check for foreign key constraints
			if strings.Contains(err.Error(), "foreign key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Cannot delete option due to existing references",
					"code":  "FOREIGN_KEY_CONSTRAINT",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to delete product option",
				"code":  "DELETION_FAILED",
			})
		}

		logger.Info().
			Int32("option_id", optionID).
			Str("option_key", option.OptionKey).
			Int32("product_id", option.ProductID).
			Msg("Product option deleted successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option.deleted", ProductOptionDeletedEvent{
		//     OptionID:  optionID,
		//     ProductID: option.ProductID,
		//     OptionKey: option.OptionKey,
		//     DeletedBy: getUserID(c),
		// })

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Product option deleted successfully",
			"data": map[string]interface{}{
				"deleted_option": map[string]interface{}{
					"id":         option.ID,
					"product_id": option.ProductID,
					"option_key": option.OptionKey,
				},
			},
		})
	}
}

func HandleCreateProductOptionValue(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type CreateProductOptionValueRequest struct {
		Value string `json:"value" validate:"required,min=1,max=100"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate option ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option ID. Must be a positive integer",
				"code":  "INVALID_OPTION_ID",
			})
		}
		optionID := int32(id)

		// Parse and validate request body
		var req CreateProductOptionValueRequest
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

		// Sanitize and validate value
		trimmedValue := strings.TrimSpace(req.Value)
		if trimmedValue == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option value cannot be empty or whitespace only",
				"code":  "INVALID_OPTION_VALUE",
			})
		}

		// Check if option exists
		_, err = db.Queries.GetProductOption(ctx, optionID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Msg("Failed to get option")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Option not found",
					"code":  "OPTION_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve option",
				"code":  "OPTION_RETRIEVAL_FAILED",
			})
		}

		// Check if value already exists for this option
		_, err = db.Queries.GetProductOptionValueByValue(ctx, database.GetProductOptionValueByValueParams{
			ProductOptionID: optionID,
			Value:           trimmedValue,
		})
		if err == nil {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "An option value with this name already exists for this option",
				"code":  "OPTION_VALUE_CONFLICT",
			})
		}
		if err != pgx.ErrNoRows {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Str("value", trimmedValue).
				Msg("Failed to check existing option values")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate option value uniqueness",
				"code":  "VALIDATION_ERROR",
			})
		}

		// Create the option value
		optionValue, err := db.Queries.CreateProductOptionValue(ctx, database.CreateProductOptionValueParams{
			ProductOptionID: optionID,
			Value:           trimmedValue,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", optionID).
				Str("value", trimmedValue).
				Msg("Failed to create option value")

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create option value",
				"code":  "OPTION_VALUE_CREATION_FAILED",
			})
		}

		logger.Info().
			Int32("option_id", optionID).
			Int32("option_value_id", optionValue.ID).
			Str("value", optionValue.Value).
			Msg("Option value created successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option_value.created", ProductOptionValueCreatedEvent{
		//     OptionID:      optionID,
		//     OptionValueID: optionValue.ID,
		//     Value:         optionValue.Value,
		// })

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"success": true,
			"data":    optionValue,
			"message": "Option value created successfully",
		})
	}
}

func HandleUpdateProductOptionValue(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type UpdateProductOptionValueRequest struct {
		Value string `json:"value" validate:"required,min=1,max=100"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate option ID
		optionID, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || optionID <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option ID. Must be a positive integer",
				"code":  "INVALID_OPTION_ID",
			})
		}

		// Parse and validate option value ID
		valueID, err := strconv.ParseInt(c.Param("value_id"), 10, 32)
		if err != nil || valueID <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option value ID. Must be a positive integer",
				"code":  "INVALID_OPTION_VALUE_ID",
			})
		}

		// Parse and validate request body
		var req UpdateProductOptionValueRequest
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

		// Sanitize and validate value
		trimmedValue := strings.TrimSpace(req.Value)
		if trimmedValue == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option value cannot be empty or whitespace only",
				"code":  "INVALID_OPTION_VALUE",
			})
		}

		// Check if option exists
		option, err := db.Queries.GetProductOption(ctx, int32(optionID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", int32(optionID)).
				Msg("Failed to get option")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Option not found",
					"code":  "OPTION_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve option",
				"code":  "OPTION_RETRIEVAL_FAILED",
			})
		}

		// Check if option value exists and belongs to the specified option
		existingValue, err := db.Queries.GetProductOptionValue(ctx, int32(valueID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_value_id", int32(valueID)).
				Msg("Failed to get option value")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Option value not found",
					"code":  "OPTION_VALUE_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve option value",
				"code":  "OPTION_VALUE_RETRIEVAL_FAILED",
			})
		}

		// Verify that the option value belongs to the specified option
		if existingValue.ProductOptionID != option.ID {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option value does not belong to the specified option",
				"code":  "OPTION_VALUE_MISMATCH",
			})
		}

		// Check if another value with the same name already exists for this option
		conflictingValue, err := db.Queries.GetProductOptionValueByValue(ctx, database.GetProductOptionValueByValueParams{
			ProductOptionID: option.ID,
			Value:           trimmedValue,
		})
		if err == nil && conflictingValue.ID != int32(valueID) {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "An option value with this name already exists for this option",
				"code":  "OPTION_VALUE_CONFLICT",
			})
		}
		if err != nil && err != pgx.ErrNoRows {
			logger.Error().
				Err(err).
				Int32("option_id", option.ID).
				Str("value", trimmedValue).
				Msg("Failed to check existing option values")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate option value uniqueness",
				"code":  "VALIDATION_ERROR",
			})
		}

		// Update the option value
		updatedValue, err := db.Queries.UpdateProductOptionValue(ctx, database.UpdateProductOptionValueParams{
			ID:    int32(valueID),
			Value: trimmedValue,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_value_id", int32(valueID)).
				Str("value", trimmedValue).
				Msg("Failed to update option value")

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update option value",
				"code":  "OPTION_VALUE_UPDATE_FAILED",
			})
		}

		logger.Info().
			Int32("option_id", option.ID).
			Int32("option_value_id", updatedValue.ID).
			Str("option_key", option.OptionKey).
			Str("old_value", existingValue.Value).
			Str("new_value", updatedValue.Value).
			Msg("Option value updated successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option_value.updated", ProductOptionValueUpdatedEvent{
		//     OptionID:      option.ID,
		//     OptionValueID: updatedValue.ID,
		//     OptionKey:     option.OptionKey,
		//     OldValue:      existingValue.Value,
		//     NewValue:      updatedValue.Value,
		// })

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    updatedValue,
			"message": "Option value updated successfully",
		})
	}
}

func HandleDeleteProductOptionValue(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate option ID
		optionID, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || optionID <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option ID. Must be a positive integer",
				"code":  "INVALID_OPTION_ID",
			})
		}

		// Parse and validate option value ID
		valueID, err := strconv.ParseInt(c.Param("value_id"), 10, 32)
		if err != nil || valueID <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid option value ID. Must be a positive integer",
				"code":  "INVALID_OPTION_VALUE_ID",
			})
		}

		// Check if option exists
		option, err := db.Queries.GetProductOption(ctx, int32(optionID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_id", int32(optionID)).
				Msg("Failed to get option")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Option not found",
					"code":  "OPTION_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve option",
				"code":  "OPTION_RETRIEVAL_FAILED",
			})
		}

		// Check if option value exists and belongs to the specified option
		existingValue, err := db.Queries.GetProductOptionValue(ctx, int32(valueID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_value_id", int32(valueID)).
				Msg("Failed to get option value")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Option value not found",
					"code":  "OPTION_VALUE_NOT_FOUND",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve option value",
				"code":  "OPTION_VALUE_RETRIEVAL_FAILED",
			})
		}

		// Verify that the option value belongs to the specified option
		if existingValue.ProductOptionID != option.ID {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option value does not belong to the specified option",
				"code":  "OPTION_VALUE_MISMATCH",
			})
		}

		// Check if option value is being used by any variants
		usageCount, err := db.Queries.CheckOptionValueUsage(ctx, int32(valueID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_value_id", int32(valueID)).
				Msg("Failed to check option value usage")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check option value usage",
				"code":  "USAGE_CHECK_FAILED",
			})
		}

		if usageCount > 0 {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot delete option value because it is being used by existing product variants",
				"code":  "OPTION_VALUE_IN_USE",
				"usage_count": usageCount,
			})
		}

		// Delete the option value
		err = db.Queries.DeleteProductOptionValue(ctx, int32(valueID))
		if err != nil {
			logger.Error().
				Err(err).
				Int32("option_value_id", int32(valueID)).
				Str("value", existingValue.Value).
				Msg("Failed to delete option value")

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to delete option value",
				"code":  "OPTION_VALUE_DELETION_FAILED",
			})
		}

		logger.Info().
			Int32("option_id", option.ID).
			Int32("option_value_id", int32(valueID)).
			Str("option_key", option.OptionKey).
			Str("deleted_value", existingValue.Value).
			Msg("Option value deleted successfully")

		// TODO: Publish event
		// eventBus.Publish("product.option_value.deleted", ProductOptionValueDeletedEvent{
		//     OptionID:      option.ID,
		//     OptionValueID: int32(valueID),
		//     OptionKey:     option.OptionKey,
		//     DeletedValue:  existingValue.Value,
		// })

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Option value deleted successfully",
		})
	}
}

// helpers
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

func convertStringToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
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
