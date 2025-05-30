// internal/handler/product_handler.go
package handler

import (
	"context"

	"github.com/dukerupert/freyja/internal/api"
	"github.com/dukerupert/freyja/internal/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type ProductHandler struct {
	productService *service.ProductService
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// Implement the generated API interface
func (h *ProductHandler) ListProducts(ctx context.Context, request api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	// Set defaults for pagination
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get products
	products, err := h.productService.ListProducts(ctx, int32(limit), int32(offset))
	if err != nil {
		return api.ListProducts400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "list_products_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// Get total count for pagination
	total, err := h.productService.CountProducts(ctx)
	if err != nil {
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

func (h *ProductHandler) CreateProduct(ctx context.Context, request api.CreateProductRequestObject) (api.CreateProductResponseObject, error) {
	if request.Body == nil {
		return api.CreateProduct400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "invalid_request",
				Message: "Request body is required",
			},
		}, nil
	}

	product, err := h.productService.CreateProduct(ctx, *request.Body)
	if err != nil {
		return api.CreateProduct400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "create_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	return api.CreateProduct201JSONResponse(*product), nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, request api.GetProductRequestObject) (api.GetProductResponseObject, error) {
	product, err := h.productService.GetProduct(ctx, openapi_types.UUID(request.Id))
	if err != nil {
		if err.Error() == "product not found" {
			return api.GetProduct404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Error:   "product_not_found",
					Message: "Product not found",
				},
			}, nil
		}
		return api.GetProduct401JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
				Error:   "get_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	return api.GetProduct200JSONResponse(*product), nil
}

func (h *ProductHandler) SearchProducts(ctx context.Context, request api.SearchProductsRequestObject) (api.SearchProductsResponseObject, error) {
	// Set defaults for pagination
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Search products
	products, err := h.productService.SearchProducts(ctx, request.Params.Q, int32(limit), int32(offset))
	if err != nil {
		return api.SearchProducts400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "search_products_failed",
				Message: err.Error(),
			},
		}, nil
	}

	// For search, we might want to implement a count query as well
	// For now, we'll use the length of results
	total := len(products)
	totalPages := 1

	// Build pagination metadata
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
	err := h.productService.DeleteProduct(ctx, openapi_types.UUID(request.Id))
	if err != nil {
		return api.DeleteProduct404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   "delete_product_failed",
				Message: err.Error(),
			},
		}, nil
	}

	return api.DeleteProduct204Response{}, nil
}

// Placeholder implementations for other endpoints
func (h *ProductHandler) ListSubscribableProducts(ctx context.Context, request api.ListSubscribableProductsRequestObject) (api.ListSubscribableProductsResponseObject, error) {
	// Set defaults for pagination
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	// Calculate offset
	// offset := (page - 1) * limit

	// For now, return empty list - you would implement this with the subscribable products query
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
	// This would need to be implemented with the update service method
	return api.UpdateProduct404JSONResponse{
		NotFoundJSONResponse: api.NotFoundJSONResponse{
			Error:   "not_implemented",
			Message: "Update product not yet implemented",
		},
	}, nil
}

func (h *ProductHandler) UpdateProductSubscription(ctx context.Context, request api.UpdateProductSubscriptionRequestObject) (api.UpdateProductSubscriptionResponseObject, error) {
	// This would need to be implemented with the update subscription service method
	return api.UpdateProductSubscription404JSONResponse{
		NotFoundJSONResponse: api.NotFoundJSONResponse{
			Error:   "not_implemented",
			Message: "Update product subscription not yet implemented",
		},
	}, nil
}