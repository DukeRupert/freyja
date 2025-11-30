package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductDetailHandler shows product details with SKU list
type ProductDetailHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewProductDetailHandler creates a new product detail handler
func NewProductDetailHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *ProductDetailHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductDetailHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *ProductDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	productID := r.PathValue("id")
	if productID == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Get product
	product, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
		TenantID: h.tenantID,
		ID:       productUUID,
	})
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Get SKUs
	skus, err := h.repo.GetProductSKUs(r.Context(), productUUID)
	if err != nil {
		http.Error(w, "Failed to load SKUs", http.StatusInternalServerError)
		return
	}

	// Format SKUs for display
	type DisplaySKU struct {
		repository.ProductSku
		BasePriceDollars string
	}

	displaySKUs := make([]DisplaySKU, len(skus))
	for i, sku := range skus {
		displaySKUs[i] = DisplaySKU{
			ProductSku:       sku,
			BasePriceDollars: fmt.Sprintf("%.2f", float64(sku.BasePriceCents)/100),
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Product":     product,
		"SKUs":        displaySKUs,
	}

	h.renderer.RenderHTTP(w, "admin/product_detail", data)
}
