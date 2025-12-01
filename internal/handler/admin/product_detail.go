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
		SkuID                pgtype.UUID
		Sku                  string
		WeightValueFormatted string
		WeightUnit           string
		GrindType            string
		BasePriceDollars     string
		TrackInventory       bool
		StockQuantity        int32
		IsActive             bool
	}

	displaySKUs := make([]DisplaySKU, len(skus))
	for i, sku := range skus {
		// Format weight value
		weightFormatted := ""
		if sku.WeightValue.Valid {
			f, err := sku.WeightValue.Float64Value()
			if err == nil && f.Valid {
				// Format without unnecessary decimal places
				weightStr := fmt.Sprintf("%.2f", f.Float64)
				// Trim trailing zeros and decimal point if needed
				for len(weightStr) > 0 && weightStr[len(weightStr)-1] == '0' {
					weightStr = weightStr[:len(weightStr)-1]
				}
				if len(weightStr) > 0 && weightStr[len(weightStr)-1] == '.' {
					weightStr = weightStr[:len(weightStr)-1]
				}
				weightFormatted = weightStr
			}
		}

		displaySKUs[i] = DisplaySKU{
			SkuID:                sku.ID,
			Sku:                  sku.Sku,
			WeightValueFormatted: weightFormatted,
			WeightUnit:           sku.WeightUnit,
			GrindType:            sku.Grind,
			BasePriceDollars:     fmt.Sprintf("%.2f", float64(sku.BasePriceCents)/100),
			TrackInventory:       sku.InventoryPolicy == "deny", // Track if policy is deny (prevent backorders)
			StockQuantity:        sku.InventoryQuantity,
			IsActive:             sku.IsActive,
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Product":     product,
		"SKUs":        displaySKUs,
	}

	h.renderer.RenderHTTP(w, "admin/product_detail", data)
}
