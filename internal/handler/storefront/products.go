package storefront

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
)

// ProductListHandler handles the product listing page
type ProductListHandler struct {
	productService service.ProductService
	renderer       *handler.Renderer
}

// NewProductListHandler creates a new product list handler
func NewProductListHandler(productService service.ProductService, renderer *handler.Renderer) *ProductListHandler {
	return &ProductListHandler{
		productService: productService,
		renderer:       renderer,
	}
}

// ServeHTTP handles GET /products
func (h *ProductListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	products, err := h.productService.ListProducts(ctx)
	if err != nil {
		// TODO: Log error with structured logging
		http.Error(w, "Failed to load products", http.StatusInternalServerError)
		return
	}

	data := BaseTemplateData(r)
	data["Products"] = products

	h.renderer.RenderHTTP(w, "products", data)
}

// ProductDetailHandler handles the product detail page
type ProductDetailHandler struct {
	productService service.ProductService
	renderer       *handler.Renderer
}

// NewProductDetailHandler creates a new product detail handler
func NewProductDetailHandler(productService service.ProductService, renderer *handler.Renderer) *ProductDetailHandler {
	return &ProductDetailHandler{
		productService: productService,
		renderer:       renderer,
	}
}

// ServeHTTP handles GET /products/{slug}
func (h *ProductDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")

	if slug == "" {
		http.NotFound(w, r)
		return
	}

	detail, err := h.productService.GetProductDetail(ctx, slug)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			http.NotFound(w, r)
			return
		}
		// TODO: Log error with structured logging
		http.Error(w, "Failed to load product", http.StatusInternalServerError)
		return
	}

	// Extract unique weights and grinds for option selectors
	weightSet := make(map[string]bool)
	grindSet := make(map[string]bool)
	for _, sku := range detail.SKUs {
		if sku.SKU.WeightValue.Valid {
			// Convert pgtype.Numeric to float64 for display
			f, err := sku.SKU.WeightValue.Float64Value()
			if err == nil && f.Valid {
				// Format without unnecessary decimal places
				weightStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f.Float64), "0"), ".")
				weight := weightStr + sku.SKU.WeightUnit
				weightSet[weight] = true
			}
		}
		grindSet[sku.SKU.Grind] = true
	}

	weights := make([]string, 0, len(weightSet))
	for w := range weightSet {
		weights = append(weights, w)
	}

	grinds := make([]string, 0, len(grindSet))
	for g := range grindSet {
		grinds = append(grinds, g)
	}

	data := BaseTemplateData(r)
	data["Product"] = detail.Product
	data["SKUs"] = detail.SKUs
	data["Weights"] = weights
	data["Grinds"] = grinds
	data["Images"] = detail.Images
	data["RequestPath"] = r.URL.Path

	h.renderer.RenderHTTP(w, "product_detail", data)
}
