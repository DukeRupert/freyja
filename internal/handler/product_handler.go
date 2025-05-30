// internal/handler/product_handler.go - Updated with logging
package handler

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/dukerupert/freyja/internal/api"
	"github.com/dukerupert/freyja/internal/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ProductHandler struct {
	productService *service.ProductService
	logger         zerolog.Logger
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         log.With().Str("component", "product_handler").Logger(),
	}
}

func (h *ProductHandler) CreateProduct(ctx context.Context, request api.CreateProductRequestObject) (api.CreateProductResponseObject, error) {
	// Create a logger with request context
	logger := h.logger.With().
		Str("method", "CreateProduct").
		Logger()

	logger.Info().Msg("CreateProduct request received")

	// Log request body validation
	if request.Body == nil {
		logger.Warn().Msg("Request body is nil")
		return api.CreateProduct400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "invalid_request",
				Message: "Request body is required",
			},
		}, nil
	}

	// Log request details (be careful with sensitive data)
	logger.Info().
		Str("title", request.Body.Title).
		Str("handle", request.Body.Handle).
		Interface("status", request.Body.Status).
		Bool("subscription_enabled", request.Body.SubscriptionEnabled != nil && *request.Body.SubscriptionEnabled).
		Msg("Processing create product request")

	// Log the full request body for debugging (remove in production)
	if reqBodyBytes, err := json.Marshal(request.Body); err == nil {
		logger.Debug().
			RawJSON("request_body", reqBodyBytes).
			Msg("Full request body")
	}

	// Call service layer
	logger.Debug().Msg("Calling productService.CreateProduct")
	product, err := h.productService.CreateProduct(ctx, *request.Body)
	if err != nil {
		logger.Error().
			Err(err).
			Str("error_type", "service_error").
			Msg("Failed to create product in service layer")

		return api.CreateProduct400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "create_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// Log successful creation
	logger.Info().
		Str("product_id", product.Id.String()).
		Str("product_title", product.Title).
		Msg("Product created successfully")

	// Log response for debugging
	if respBytes, err := json.Marshal(product); err == nil {
		logger.Debug().
			RawJSON("response_body", respBytes).
			Msg("Returning created product")
	}

	return api.CreateProduct201JSONResponse(*product), nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, request api.GetProductRequestObject) (api.GetProductResponseObject, error) {
	logger := h.logger.With().
		Str("method", "GetProduct").
		Str("product_id", request.Id.String()).
		Logger()

	logger.Info().Msg("GetProduct request received")

	product, err := h.productService.GetProduct(ctx, openapi_types.UUID(request.Id))
	if err != nil {
		if err == sql.ErrNoRows || err.Error() == "product not found" {
			logger.Info().Msg("Product not found")
			return api.GetProduct404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Error:   "product_not_found",
					Message: "Product not found",
				},
			}, nil
		}
		
		logger.Error().
			Err(err).
			Msg("Error retrieving product")
		
		return api.GetProduct400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "get_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	logger.Info().
		Str("product_title", product.Title).
		Msg("Product retrieved successfully")

	return api.GetProduct200JSONResponse(*product), nil
}

func (h *ProductHandler) ListProducts(ctx context.Context, request api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	logger := h.logger.With().
		Str("method", "ListProducts").
		Logger()

	// Log pagination parameters
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	logger.Info().
		Int("page", page).
		Int("limit", limit).
		Msg("ListProducts request received")

	// Calculate offset
	offset := (page - 1) * limit

	// Get products
	logger.Debug().Msg("Calling productService.ListProducts")
	products, err := h.productService.ListProducts(ctx, int32(limit), int32(offset))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list products")

		return api.ListProducts400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "list_products_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// Get total count for pagination
	logger.Debug().Msg("Getting product count")
	total, err := h.productService.CountProducts(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to count products")

		return api.ListProducts400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "count_products_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	logger.Info().
		Int("products_found", len(products)).
		Int64("total_products", total).
		Int("total_pages", totalPages).
		Msg("Products listed successfully")

	// Build pagination metadata
	pagination := api.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: totalPages,
	}

	return api.ListProducts200JSONResponse{
		Products:   &products,
		Pagination: &pagination,
	}, nil
}

func (h *ProductHandler) SearchProducts(ctx context.Context, request api.SearchProductsRequestObject) (api.SearchProductsResponseObject, error) {
	logger := h.logger.With().
		Str("method", "SearchProducts").
		Str("query", request.Params.Q).
		Logger()

	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	logger.Info().
		Int("page", page).
		Int("limit", limit).
		Msg("SearchProducts request received")

	offset := (page - 1) * limit

	logger.Debug().Msg("Calling productService.SearchProducts")
	products, err := h.productService.SearchProducts(ctx, request.Params.Q, int32(limit), int32(offset))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to search products")

		return api.SearchProducts400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "search_products_failed",
				Message: err.Error(),
			},
		}, nil
	}

	logger.Info().
		Int("results_found", len(products)).
		Msg("Product search completed")

	total := len(products)
	totalPages := 1

	pagination := api.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return api.SearchProducts200JSONResponse{
		Products:   &products,
		Pagination: &pagination,
	}, nil
}

func (h *ProductHandler) DeleteProduct(ctx context.Context, request api.DeleteProductRequestObject) (api.DeleteProductResponseObject, error) {
	logger := h.logger.With().
		Str("method", "DeleteProduct").
		Str("product_id", request.Id.String()).
		Logger()

	logger.Info().Msg("DeleteProduct request received")

	// First check if the product exists
	logger.Debug().Msg("Checking if product exists")
	_, err := h.productService.GetProduct(ctx, openapi_types.UUID(request.Id))
	if err != nil {
		if err == sql.ErrNoRows || err.Error() == "product not found" {
			logger.Info().Msg("Product not found for deletion")
			return api.DeleteProduct404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Error:   "product_not_found",
					Message: "Product not found",
				},
			}, nil
		}
		
		logger.Error().
			Err(err).
			Msg("Error checking product existence")

		return api.DeleteProduct404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   "delete_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// Product exists, proceed with deletion
	logger.Debug().Msg("Proceeding with product deletion")
	err = h.productService.DeleteProduct(ctx, openapi_types.UUID(request.Id))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to delete product")

		return api.DeleteProduct404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   "delete_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	logger.Info().Msg("Product deleted successfully")
	return api.DeleteProduct204Response{}, nil
}

// Placeholder implementations for other endpoints
func (h *ProductHandler) ListSubscribableProducts(ctx context.Context, request api.ListSubscribableProductsRequestObject) (api.ListSubscribableProductsResponseObject, error) {
	logger := h.logger.With().
		Str("method", "ListSubscribableProducts").
		Logger()

	logger.Info().Msg("ListSubscribableProducts request received (not implemented)")

	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	products := []api.Product{}
	pagination := api.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      0,
		TotalPages: 0,
	}

	return api.ListSubscribableProducts200JSONResponse{
		Products:   &products,
		Pagination: &pagination,
	}, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, request api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
	logger := h.logger.With().
		Str("method", "UpdateProduct").
		Str("product_id", request.Id.String()).
		Logger()

	logger.Warn().Msg("UpdateProduct called but not implemented")

	return api.UpdateProduct404JSONResponse{
		NotFoundJSONResponse: api.NotFoundJSONResponse{
			Error:   "not_implemented",
			Message: "Update product not yet implemented",
		},
	}, nil
}

func (h *ProductHandler) UpdateProductSubscription(ctx context.Context, request api.UpdateProductSubscriptionRequestObject) (api.UpdateProductSubscriptionResponseObject, error) {
	logger := h.logger.With().
		Str("method", "UpdateProductSubscription").
		Str("product_id", request.Id.String()).
		Logger()

	logger.Warn().Msg("UpdateProductSubscription called but not implemented")

	return api.UpdateProductSubscription404JSONResponse{
		NotFoundJSONResponse: api.NotFoundJSONResponse{
			Error:   "not_implemented",
			Message: "Update product subscription not yet implemented",
		},
	}, nil
}