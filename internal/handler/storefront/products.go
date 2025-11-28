package storefront

import (
	"errors"
	"net/http"

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

	data := map[string]interface{}{
		"Products": products,
		"Year":     2024,
	}

	tmpl, err := h.renderer.Execute("products")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
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

	data := map[string]interface{}{
		"Product": detail.Product,
		"SKUs":    detail.SKUs,
		"Images":  detail.Images,
		"Year":    2024,
	}

	tmpl, err := h.renderer.Execute("product_detail")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
