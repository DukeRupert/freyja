package storefront

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/dukerupert/freyja/internal/service"
)

// ProductListHandler handles the product listing page
type ProductListHandler struct {
	productService service.ProductService
	templates      *template.Template
}

// NewProductListHandler creates a new product list handler
func NewProductListHandler(productService service.ProductService, templates *template.Template) *ProductListHandler {
	return &ProductListHandler{
		productService: productService,
		templates:      templates,
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

	// TODO: Render template with products
	// For now, return simple HTML stub
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head><title>Products - Freyja Coffee</title></head>
		<body>
			<h1>Products</h1>
			<p>Found %d products</p>
			<ul>
	`, len(products))

	for _, p := range products {
		fmt.Fprintf(w, `<li><a href="/products/%s">%s</a> - %s</li>`, p.Slug, p.Name, p.Origin.String)
	}

	fmt.Fprint(w, `
			</ul>
		</body>
		</html>
	`)
}

// ProductDetailHandler handles the product detail page
type ProductDetailHandler struct {
	productService service.ProductService
	templates      *template.Template
}

// NewProductDetailHandler creates a new product detail handler
func NewProductDetailHandler(productService service.ProductService, templates *template.Template) *ProductDetailHandler {
	return &ProductDetailHandler{
		productService: productService,
		templates:      templates,
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

	// TODO: Render template with product detail
	// For now, return simple HTML stub
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head><title>%s - Freyja Coffee</title></head>
		<body>
			<h1>%s</h1>
			<p>Origin: %s</p>
			<p>Roast Level: %s</p>
			<p>%s</p>
			<h2>Available Options</h2>
			<ul>
	`, detail.Product.Name, detail.Product.Name, detail.Product.Origin.String, detail.Product.RoastLevel.String, detail.Product.Description.String)

	for _, sku := range detail.SKUs {
		fmt.Fprintf(w, `<li>%s %s - %s - $%.2f - %s</li>`,
			sku.SKU.WeightValue.Int.String(),
			sku.SKU.WeightUnit,
			sku.SKU.Grind,
			float64(sku.PriceCents)/100.0,
			sku.InventoryMessage,
		)
	}

	fmt.Fprint(w, `
			</ul>
		</body>
		</html>
	`)
}
