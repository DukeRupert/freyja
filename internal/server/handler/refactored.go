package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/server/views/form"
	"github.com/dukerupert/freyja/internal/server/views/page"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func HandleGetHome() echo.HandlerFunc {
	return func(c echo.Context) error {
		component := page.HomePage()
		return component.Render(context.Background(), c.Response().Writer)
	}
}

func HandleGetProducts(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Info().Msg("Starting GetProducts request")
		ctx := c.Request().Context()
		isHTMX := c.Get("htmx").(bool)

		pg := parseIntParam(c.QueryParam("page"), 1) // Default to page 1
		limit := parseIntParam(c.QueryParam("limit"), 10)
		offset := (pg - 1) * limit

		filters := database.GetProductsParams{
			Limit:  limit,
			Offset: offset,
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

		// Get accurate total for pagination
		var total int64
		if filters.Active {
			total, err = db.Queries.CountActiveProducts(ctx)
		} else {
			total, err = db.Queries.CountInactiveProducts(ctx)
		}
		if err != nil {
			logger.Error().Err(err).Msg("Failed to fetch product count")
			total = int64(len(products))
		}

		logger.Info().
			Int64("total_products", total).
			Msg("Successfully completed GetProducts request")

		// handle JSON request
		if c.Request().Header.Get("Accept") == "application/json" {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"products": products,
				"total":    total,
				"filters":  filters,
			})
		}

		// Render the Templ component directly
		pagination := CalculatePagination(filters.Limit, filters.Offset, total)
		data := page.ProductsPageData{
			Products:   products,
			Pagination: pagination,
		}

		// partial page request from htmx
		if isHTMX {
			component := page.ProductsContent(data)
			return component.Render(context.Background(), c.Response().Writer)
		}

		// default full page request
		component := page.ProductsPage(data)
		return component.Render(context.Background(), c.Response().Writer)
	}
}

func HandleGetProduct(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse product ID
		productID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID",
				"code":  "INVALID_ID",
			})
		}

		product, err := db.Queries.GetProduct(ctx, int32(productID))
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

		// Get the product options
		options, err := GetProductOptionsForProduct(c.Request().Context(), db.Queries, int32(productID))
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product options",
				"code":  "INTERNAL_ERROR",
			})
		}

		variants, err := db.Queries.GetActiveVariantsByProduct(ctx, product.ID)
		if err != nil {
			c.Logger().Error("Failed to get product variants: ", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve product variants",
				"code":  "INTERNAL_ERROR",
			})
		}

		logger.Info().
			Int("total_products", len(variants)).
			Msg("Successfully completed GetProducts request")

		if c.Request().Header.Get("Accept") == "application/json" {
			// Transform to API-friendly format
			v := convertProductVariantsToJSON(variants)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"product":  product,
				"variants": v,
				"total":    len(v),
			})
		}

		// Render the Templ component directly
		data := page.ProductDetailsPageData{
			Product:  product,
			Options:  options,
			Variants: variants,
		}
		component := page.ProductDetailsPage(data)
		return component.Render(context.Background(), c.Response().Writer)
	}
}

type CreateProductRequest struct {
	Name        string `json:"name" form:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" form:"description" validate:"max=1000"`
	Active      bool   `json:"active" form:"active"`
}

func HandleCreateProduct(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		isHTMX := c.Get("htmx").(bool)

		// DEBUG: Log all incoming request details
		logger.Info().
			Str("method", c.Request().Method).
			Str("content_type", c.Request().Header.Get("Content-Type")).
			Bool("is_htmx", isHTMX).
			Int64("content_length", c.Request().ContentLength).
			Msg("Incoming request details")

		// DEBUG: Log all form values before binding
		if err := c.Request().ParseForm(); err == nil {
			logger.Info().Msg("Form values:")
			for key, values := range c.Request().Form {
				logger.Info().
					Str("key", key).
					Strs("values", values).
					Msg("Form field")
			}
		} else {
			logger.Error().Err(err).Msg("Failed to parse form")
		}

		// Initialize error collection
		var fieldErrors []form.FieldError

		var req CreateProductRequest
		var formData map[string]interface{}
		if err := c.Bind(&req); err != nil {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "form",
				Message: "Invalid request format",
				Code:    "INVALID_FORMAT",
			})
			logger.Info().Interface("fieldErrors", fieldErrors).Msg("Validation errors")
			formData = map[string]interface{}{
				"name":        req.Name,
				"description": req.Description,
				"active":      req.Active,
			}
			return handleErrorResponse(c, formData, isHTMX, fieldErrors, "Invalid request format")
		}

		// Validate request using validator tags
		if err := c.Validate(&req); err != nil {
			validationErrors := extractValidationErrors(err)
			fieldErrors = append(fieldErrors, validationErrors...)
		}

		// Custom validation: sanitize and validate name
		trimmedName := strings.TrimSpace(req.Name)
		if trimmedName == "" && req.Name != "" {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "name",
				Message: "Product name cannot be empty or whitespace only",
				Code:    "INVALID_NAME",
			})
		}

		// Check for name collision
		if trimmedName != "" {
			_, err := db.Queries.GetProductByName(ctx, trimmedName)
			if err != nil && err != pgx.ErrNoRows {
				logger.Error().
					Err(err).
					Str("product_name", trimmedName).
					Msg("Failed to check product name uniqueness")

				fieldErrors = append(fieldErrors, form.FieldError{
					Field:   "name",
					Message: "Unable to validate product name uniqueness",
					Code:    "VALIDATION_ERROR",
				})
			} else if err == nil {
				fieldErrors = append(fieldErrors, form.FieldError{
					Field:   "name",
					Message: "A product with this name already exists",
					Code:    "NAME_CONFLICT",
				})
			}
		}

		// Return validation errors if any exist
		formData = map[string]interface{}{
			"name":        req.Name,
			"description": req.Description,
			"active":      req.Active,
		}
		if len(fieldErrors) > 0 {
			logger.Info().Interface("fieldErrors", fieldErrors).Msg("Validation errors")
			return handleErrorResponse(c, formData, isHTMX, fieldErrors, "Validation failed")
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
				fieldErrors = append(fieldErrors, form.FieldError{
					Field:   "name",
					Message: "Product name must be unique",
					Code:    "DUPLICATE_NAME",
				})
				return handleErrorResponse(c, formData, isHTMX, fieldErrors, "Product name must be unique")
			}

			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "form",
				Message: "Failed to create product",
				Code:    "CREATION_FAILED",
			})
			return handleErrorResponse(c, formData, isHTMX, fieldErrors, "Failed to create product")
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

		// Handle successful response
		if isHTMX {
			component := form.CreateProductSuccess(product)
			return component.Render(context.Background(), c.Response().Writer)
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"success": true,
			"data":    product,
			"message": "Product created successfully",
		})
	}
}

func HandleGetCreateProductForm() echo.HandlerFunc {
	return func(c echo.Context) error {
		errors := []form.FieldError{}
		component := form.CreateProductModal(errors)
		return component.Render(context.Background(), c.Response().Writer)
	}
}

func HandleUpdateProduct(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type UpdateParams struct {
		Name        string `json:"name" validate:"required,min=1,max=255"`
		Description string `json:"description" validate:"max=1000"` // Changed to string
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
		isHTMX := c.Get("htmx").(bool)

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

		if isHTMX {
			// Set proper header for redirect
			c.Response().Header().Add("HX-Redirect", "/products")
			return c.HTML(http.StatusOK, "<div>Product deleted.</div>")
		}

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
		ID           int32                          `json:"id"`
		ProductID    int32                          `json:"product_id"`
		OptionKey    string                         `json:"option_key"`
		CreatedAt    time.Time                      `json:"created_at"`
		OptionValues []database.ProductOptionValues `json:"option_values"`
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
		options, err := db.Queries.GetProductOptionKeys(ctx, id)
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
		OptionKey string `json:"option_key" form:"option_key" validate:"required,min=1,max=50"`
	}

	return func(c echo.Context) error {
		ctx := c.Request().Context()
		isHTMX := c.Get("htmx").(bool)

		// DEBUG: Log all incoming request details
		logger.Info().
			Str("method", c.Request().Method).
			Str("content_type", c.Request().Header.Get("Content-Type")).
			Bool("is_htmx", isHTMX).
			Int64("content_length", c.Request().ContentLength).
			Msg("Incoming request details")

		// Initialize error collection
		var fieldErrors []form.FieldError

		// Parse and validate product ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "product_id",
				Message: "Invalid product ID. Must be a positive integer",
				Code:    "INVALID_PRODUCT_ID",
			})
		}
		productID := int32(id)

		// Parse and validate request body
		var req CreateProductOptionRequest
		var formData map[string]interface{}
		if err := c.Bind(&req); err != nil {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "form",
				Message: "Invalid request format",
				Code:    "INVALID_FORMAT",
			})
			logger.Info().Interface("fieldErrors", fieldErrors).Msg("Validation errors")

			formData = map[string]interface{}{
				"option_key": req.OptionKey,
			}

			return handleOptionKeyErrorResponse(c, productID, formData, isHTMX, fieldErrors, "Invalid request format")
		}

		// Validate request using validator tags
		if err := c.Validate(&req); err != nil {
			validationErrors := extractValidationErrors(err)
			fieldErrors = append(fieldErrors, validationErrors...)
		}

		// Custom validation: sanitize and validate option key
		trimmedOptionKey := strings.TrimSpace(req.OptionKey)
		if trimmedOptionKey == "" && req.OptionKey != "" {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "option_key",
				Message: "Option key cannot be empty or whitespace only",
				Code:    "INVALID_OPTION_KEY",
			})
		}

		// Normalize option key (lowercase for consistency)
		normalizedOptionKey := strings.ToLower(trimmedOptionKey)

		// Check if product exists (only if product ID is valid)
		var product database.Products
		if id > 0 {
			product, err = db.Queries.GetProduct(ctx, productID)
			if err != nil {
				logger.Error().
					Err(err).
					Int32("product_id", productID).
					Msg("Failed to get product")

				if err == pgx.ErrNoRows {
					fieldErrors = append(fieldErrors, form.FieldError{
						Field:   "product_id",
						Message: "Product not found",
						Code:    "PRODUCT_NOT_FOUND",
					})
				} else {
					fieldErrors = append(fieldErrors, form.FieldError{
						Field:   "form",
						Message: "Failed to retrieve product",
						Code:    "PRODUCT_RETRIEVAL_FAILED",
					})
				}
			} else {
				// Check if product is active (optional business rule)
				if !product.Active {
					fieldErrors = append(fieldErrors, form.FieldError{
						Field:   "product_id",
						Message: "Cannot add options to inactive product",
						Code:    "PRODUCT_INACTIVE",
					})
				}
			}
		}

		// Check for option key collision (only if we have a valid product and option key)
		if id > 0 && normalizedOptionKey != "" && len(fieldErrors) == 0 {
			existingOptions, err := db.Queries.GetProductOptionKeys(ctx, productID)
			if err != nil {
				logger.Error().
					Err(err).
					Int32("product_id", productID).
					Msg("Failed to check existing options")

				fieldErrors = append(fieldErrors, form.FieldError{
					Field:   "option_key",
					Message: "Unable to validate option uniqueness",
					Code:    "VALIDATION_ERROR",
				})
			} else {
				for _, existingOption := range existingOptions {
					if strings.ToLower(existingOption.OptionKey) == normalizedOptionKey {
						fieldErrors = append(fieldErrors, form.FieldError{
							Field:   "option_key",
							Message: "An option with this key already exists for this product",
							Code:    "OPTION_KEY_CONFLICT",
						})
						break
					}
				}
			}
		}

		// Return validation errors if any exist
		formData = map[string]interface{}{
			"option_key": req.OptionKey,
		}

		if len(fieldErrors) > 0 {
			logger.Info().Interface("fieldErrors", fieldErrors).Msg("Validation errors")
			return handleOptionKeyErrorResponse(c, productID, formData, isHTMX, fieldErrors, "Validation failed")
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
				fieldErrors = append(fieldErrors, form.FieldError{
					Field:   "option_key",
					Message: "Option key must be unique for this product",
					Code:    "DUPLICATE_OPTION_KEY",
				})
				return handleOptionKeyErrorResponse(c, productID, formData, isHTMX, fieldErrors, "Option key must be unique for this product")
			}

			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   "form",
				Message: "Failed to create product option",
				Code:    "OPTION_CREATION_FAILED",
			})
			return handleOptionKeyErrorResponse(c, productID, formData, isHTMX, fieldErrors, "Failed to create product option")
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

		// Handle successful response
		if isHTMX {
			// Get the product options
			options, err := GetProductOptionsForProduct(ctx, db.Queries, productID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error": "Failed to retrieve product options",
					"code":  "INTERNAL_ERROR",
				})
			}
			component := form.CreateProductOptionSuccess(productID, options)
			return component.Render(context.Background(), c.Response().Writer)
		}

		return c.JSON(http.StatusCreated, map[string]any{
			"success": true,
			"data":    option,
			"message": "Product option created successfully",
		})
	}
}

func HandleGetCreateProductOptionForm() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse and validate product ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid product ID. Must be a positive integer",
				"code":  "INVALID_PRODUCT_ID",
			})
		}
		productID := int32(id)

		errors := []form.FieldError{}
		component := form.Create_Options_Modal(productID, errors)
		return component.Render(context.Background(), c.Response().Writer)
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
		existingOptions, err := db.Queries.GetProductOptionKeys(ctx, existingOption.ProductID)
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
		isHTMX := c.Get("htmx").(bool)

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

		if isHTMX {

			// Get the product options
			options, err := GetProductOptionsForProduct(ctx, db.Queries, option.ProductID)
			if err != nil {
				return c.JSON(500, map[string]string{"error": err.Error()})
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error": "Failed to retrieve product options",
					"code":  "INTERNAL_ERROR",
				})
			}
			component := page.ProductOptionsCard(option.ProductID, options)
			return component.Render(ctx, c.Response().Writer)
		}

		return c.JSON(http.StatusOK, map[string]any{
			"success": true,
			"message": "Product option deleted successfully",
			"data": map[string]any{
				"deleted_option": map[string]any{
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

func HandleGetProductOptionValue(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
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

		// Get the option value
		optionValue, err := db.Queries.GetProductOptionValue(ctx, int32(valueID))
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
		if optionValue.ProductOptionID != option.ID {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Option value does not belong to the specified option",
				"code":  "OPTION_VALUE_MISMATCH",
			})
		}

		logger.Debug().
			Int32("option_id", option.ID).
			Int32("option_value_id", optionValue.ID).
			Str("option_key", option.OptionKey).
			Str("value", optionValue.Value).
			Msg("Option value retrieved successfully")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    optionValue,
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
				"error":       "Cannot delete option value because it is being used by existing product variants",
				"code":        "OPTION_VALUE_IN_USE",
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

// CreateVariant handles variant creation with database-enforced option constraints.
//
// Database Trigger Constraints:
// 1. If product has NO options defined → variant can exist with empty option set (default variant)
// 2. If product has ANY options defined → variant MUST have exactly one value for EACH option
// 3. Option values must belong to their specified options (validated automatically)
// 4. Cannot create partial option sets - it's all or nothing
//
// This means: products either have all variants with complete option combinations,
// or all variants with no options (default variants only).
func HandleCreateProductVariant(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	type CreateVariantRequest struct {
		Name           string  `json:"name" validate:"required,min=1,max=500"`
		Price          int32   `json:"price" validate:"required,min=1"`
		Stock          int32   `json:"stock" validate:"min=0"`
		Active         bool    `json:"active"`
		IsSubscription bool    `json:"is_subscription"`
		OptionValueIDs []int32 `json:"option_value_ids"` // For setting variant options
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
		var req CreateVariantRequest
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

		// Sanitize name
		trimmedName := strings.TrimSpace(req.Name)
		if trimmedName == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Variant name cannot be empty or whitespace only",
				"code":  "INVALID_VARIANT_NAME",
			})
		}

		// Verify product exists
		_, err = db.Queries.GetProduct(ctx, productID)
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

		// Check for existing variant with same name for this product
		existingVariants, err := db.Queries.GetVariantsByProduct(ctx, productID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Msg("Failed to check existing variants")
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate variant name uniqueness",
				"code":  "VALIDATION_ERROR",
			})
		}

		for _, existingVariant := range existingVariants {
			if strings.EqualFold(existingVariant.Name, trimmedName) && existingVariant.ArchivedAt.Time.IsZero() {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "A variant with this name already exists for this product",
					"code":  "VARIANT_NAME_CONFLICT",
				})
			}
		}

		// Create variant
		variant, err := db.Queries.CreateVariant(ctx, database.CreateVariantParams{
			ProductID:      productID,
			Name:           trimmedName,
			Price:          req.Price,
			Stock:          req.Stock,
			Active:         req.Active,
			IsSubscription: req.IsSubscription,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int32("product_id", productID).
				Str("name", trimmedName).
				Msg("Failed to create variant")

			// Check for specific database constraints
			if strings.Contains(err.Error(), "duplicate key") {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error": "Variant with this configuration already exists",
					"code":  "DUPLICATE_VARIANT",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create variant",
				"code":  "VARIANT_CREATION_FAILED",
			})
		}

		// Create variant option associations if provided
		if len(req.OptionValueIDs) > 0 {
			for _, valueID := range req.OptionValueIDs {
				// Get the option value to validate it exists and get its option ID
				optionValue, err := db.Queries.GetProductOptionValue(ctx, valueID)
				if err != nil {
					logger.Error().
						Err(err).
						Int32("option_value_id", valueID).
						Int32("variant_id", variant.ID).
						Msg("Failed to get option value for variant")

					// Clean up the variant since option association failed
					db.Queries.ArchiveVariant(ctx, variant.ID)

					if err == pgx.ErrNoRows {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": fmt.Sprintf("Option value with ID %d not found", valueID),
							"code":  "OPTION_VALUE_NOT_FOUND",
						})
					}
					return c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"error": "Failed to validate option values",
						"code":  "OPTION_VALUE_RETRIEVAL_FAILED",
					})
				}

				// Create the variant option association
				_, err = db.Queries.CreateVariantOption(ctx, database.CreateVariantOptionParams{
					ProductVariantID:     variant.ID,
					ProductOptionID:      optionValue.ProductOptionID,
					ProductOptionValueID: valueID,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int32("variant_id", variant.ID).
						Int32("option_value_id", valueID).
						Msg("Failed to create variant option association")

					// Clean up the variant since option association failed
					db.Queries.ArchiveVariant(ctx, variant.ID)

					// Check for database constraint violations
					if strings.Contains(err.Error(), "variant must have exactly one value") {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": "Variant must have exactly one value for each product option",
							"code":  "INCOMPLETE_OPTION_SET",
						})
					}
					if strings.Contains(err.Error(), "option value") && strings.Contains(err.Error(), "does not belong") {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": "Option value does not belong to the specified option",
							"code":  "INVALID_OPTION_COMBINATION",
						})
					}

					return c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"error": "Failed to create variant option associations",
						"code":  "VARIANT_OPTION_CREATION_FAILED",
					})
				}
			}
		}

		// Publish event
		// event := interfaces.Event{
		// 	Type:        "variant.created",
		// 	AggregateID: fmt.Sprintf("variant:%d", variant.ID),
		// 	Data: map[string]interface{}{
		// 		"variant_id":      variant.ID,
		// 		"product_id":      variant.ProductID,
		// 		"name":           variant.Name,
		// 		"price":          variant.Price,
		// 		"stock":          variant.Stock,
		// 		"option_value_ids": req.OptionValueIDs,
		// 	},
		// 	Timestamp: time.Now(),
		// }

		// if err := eventBus.PublishEvent(ctx, event); err != nil {
		// 	logger.Error().
		// 		Err(err).
		// 		Int32("variant_id", variant.ID).
		// 		Msg("Failed to publish variant.created event")
		// 	// Don't fail the request for event publishing failures
		// }

		logger.Info().
			Int32("product_id", productID).
			Int32("variant_id", variant.ID).
			Str("name", variant.Name).
			Int32("price", variant.Price).
			Msg("Created product variant")

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"success": true,
			"data":    variant,
			"message": "Variant created successfully",
		})
	}
}

func HandleGetVariant(db *database.DB, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate variant ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid variant ID. Must be a positive integer",
				"code":  "INVALID_VARIANT_ID",
			})
		}
		variantID := int32(id)

		// Get variant
		variant, err := db.Queries.GetVariant(ctx, variantID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("variant_id", variantID).
				Msg("Failed to get variant")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Variant not found",
					"code":  "VARIANT_NOT_FOUND",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve variant",
				"code":  "VARIANT_RETRIEVAL_FAILED",
			})
		}

		logger.Info().
			Int32("variant_id", variantID).
			Str("name", variant.Name).
			Msg("Retrieved variant")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    variant,
		})
	}
}

// HandleDeleteProductVariant archives a product variant (soft delete).
// - Soft deletes by archiving (following existing pattern)
// - Validates that the variant exists and isn't already archived
// - Publishes events for the deletion
// - Returns the archived variant so you can see the archived_at timestamp
func HandleDeleteProductVariant(db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// Parse and validate variant ID
		id, err := strconv.ParseInt(c.Param("id"), 10, 32)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid variant ID. Must be a positive integer",
				"code":  "INVALID_VARIANT_ID",
			})
		}
		variantID := int32(id)

		// Check if variant exists and get it for logging/events
		existingVariant, err := db.Queries.GetVariant(ctx, variantID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("variant_id", variantID).
				Msg("Failed to get variant for deletion")

			if err == pgx.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Variant not found",
					"code":  "VARIANT_NOT_FOUND",
				})
			}

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve variant",
				"code":  "VARIANT_RETRIEVAL_FAILED",
			})
		}

		// Check if variant is already archived
		if existingVariant.ArchivedAt.Valid {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Variant is already archived",
				"code":  "VARIANT_ALREADY_ARCHIVED",
			})
		}

		// Archive the variant (soft delete)
		archivedVariant, err := db.Queries.ArchiveVariant(ctx, variantID)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("variant_id", variantID).
				Str("variant_name", existingVariant.Name).
				Msg("Failed to archive variant")

			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to archive variant",
				"code":  "VARIANT_ARCHIVE_FAILED",
			})
		}

		// Publish event
		// event := interfaces.Event{
		// 	Type:        "variant.archived",
		// 	AggregateID: fmt.Sprintf("variant:%d", variantID),
		// 	Data: map[string]interface{}{
		// 		"variant_id": variantID,
		// 		"product_id": existingVariant.ProductID,
		// 		"name":       existingVariant.Name,
		// 		"archived_at": archivedVariant.ArchivedAt.Time,
		// 	},
		// 	Timestamp: time.Now(),
		// }

		// if err := eventBus.PublishEvent(ctx, event); err != nil {
		// 	logger.Error().
		// 		Err(err).
		// 		Int32("variant_id", variantID).
		// 		Msg("Failed to publish variant.archived event")
		// 	// Don't fail the request for event publishing failures
		// }

		logger.Info().
			Int32("variant_id", variantID).
			Int32("product_id", existingVariant.ProductID).
			Str("name", existingVariant.Name).
			Msg("Archived product variant")

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Variant archived successfully",
			"data":    archivedVariant,
		})
	}
}

// helpers
func GetProductOptionsForProduct(ctx context.Context, db *database.Queries, productID int32) ([]page.ProductOption, error) {
	// Get all option keys for the product
	optionKeys, err := db.GetProductOptionKeys(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product option keys: %w", err)
	}

	var productOptions []page.ProductOption

	// For each option key, get its values
	for _, optionKey := range optionKeys {
		// Get all values for this option
		optionValues, err := db.GetProductOptionValues(ctx, optionKey.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get values for option %s: %w", optionKey.OptionKey, err)
		}

		// Convert to string slice
		var values []string
		for _, val := range optionValues {
			values = append(values, val.Value)
		}

		// Create ProductOption struct
		productOption := page.ProductOption{
			ID:     optionKey.ID,
			Key:    optionKey.OptionKey,
			Values: values,
		}

		productOptions = append(productOptions, productOption)
	}

	return productOptions, nil
}

// handleErrorResponse handles both JSON and HTMX error responses
func handleErrorResponse(c echo.Context, formData map[string]interface{}, isHTMX bool, fieldErrors []form.FieldError, message string) error {
	if isHTMX {
		component := form.CreateProductForm(fieldErrors, formData)
		return component.Render(context.Background(), c.Response().Writer)
	}

	// For JSON API requests
	return c.JSON(http.StatusBadRequest, form.ErrorResponse{
		Success: false,
		Message: message,
		Errors:  fieldErrors,
	})
}

// handleErrorResponse handles both JSON and HTMX error responses
func handleOptionKeyErrorResponse(c echo.Context, product_id int32, formData map[string]interface{}, isHTMX bool, fieldErrors []form.FieldError, message string) error {
	if isHTMX {
		component := form.CreateProductOptionForm(product_id, fieldErrors, formData)
		return component.Render(context.Background(), c.Response().Writer)
	}

	// For JSON API requests
	return c.JSON(http.StatusBadRequest, form.ErrorResponse{
		Success: false,
		Message: message,
		Errors:  fieldErrors,
	})
}

// extractValidationErrors converts validator errors to FieldError slice
func extractValidationErrors(err error) []form.FieldError {
	var fieldErrors []form.FieldError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			fieldErrors = append(fieldErrors, form.FieldError{
				Field:   strings.ToLower(fieldError.Field()),
				Message: getValidationErrorMessage(fieldError),
				Code:    fmt.Sprintf("VALIDATION_%s", strings.ToUpper(fieldError.Tag())),
			})
		}
	}

	return fieldErrors
}

// getValidationErrorMessage converts validator field errors to human-readable messages
func getValidationErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s cannot exceed %s characters", fe.Field(), fe.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
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
