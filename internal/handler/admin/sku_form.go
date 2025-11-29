package admin

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// SKUFormHandler handles SKU create and edit
type SKUFormHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewSKUFormHandler creates a new SKU form handler
func NewSKUFormHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *SKUFormHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SKUFormHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *SKUFormHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.showForm(w, r)
	} else if r.Method == http.MethodPost {
		h.handleSubmit(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (h *SKUFormHandler) showForm(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("product_id")
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

	// Check if editing existing SKU
	skuID := r.PathValue("sku_id")
	var sku repository.ProductSku
	var basePriceDollars string

	if skuID != "" {
		// Edit mode
		var skuUUID pgtype.UUID
		if err := skuUUID.Scan(skuID); err != nil {
			http.Error(w, "Invalid SKU ID", http.StatusBadRequest)
			return
		}

		sku, err = h.repo.GetSKUByID(r.Context(), skuUUID)
		if err != nil {
			http.Error(w, "SKU not found", http.StatusNotFound)
			return
		}

		basePriceDollars = fmt.Sprintf("%.2f", float64(sku.BasePriceCents)/100)
	}

	// Calculate suggested price for new SKUs
	suggestedPrice := ""
	if skuID == "" {
		// Default: 12oz whole bean
		suggestedPrice = calculateSuggestedPrice(12, "oz")
	}

	data := map[string]interface{}{
		"Product":         product,
		"SKU":             sku,
		"BasePriceDollars": basePriceDollars,
		"SuggestedPrice":  suggestedPrice,
	}

	h.renderer.RenderHTTP(w, "admin/sku_form", data)
}

func (h *SKUFormHandler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	productID := r.PathValue("product_id")
	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Get product for slug generation
	product, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
		TenantID: h.tenantID,
		ID:       productUUID,
	})
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Parse form values
	weightValue, _ := strconv.Atoi(r.FormValue("weight_value"))
	weightUnit := r.FormValue("weight_unit")
	grind := r.FormValue("grind")
	basePriceStr := r.FormValue("base_price")
	inventoryQty, _ := strconv.Atoi(r.FormValue("inventory_quantity"))
	lowStockStr := r.FormValue("low_stock_threshold")
	inventoryPolicy := r.FormValue("inventory_policy")
	isActive := r.FormValue("is_active") == "true"

	// Parse base price
	basePriceDollars, err := strconv.ParseFloat(basePriceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}
	basePriceCents := int32(math.Round(basePriceDollars * 100))

	// Auto-generate SKU code if not provided
	skuCode := strings.TrimSpace(r.FormValue("sku"))
	if skuCode == "" {
		skuCode = generateSKUCode(product.Slug, weightValue, weightUnit, grind)
	}

	// Parse low stock threshold
	lowStockThreshold := pgtype.Int4{}
	if lowStockStr != "" {
		if threshold, err := strconv.Atoi(lowStockStr); err == nil {
			lowStockThreshold.Int32 = int32(threshold)
			lowStockThreshold.Valid = true
		}
	}

	// Calculate weight in grams for shipping
	weightGrams := calculateWeightGrams(weightValue, weightUnit)

	// Convert weight value to pgtype.Numeric
	var weightNumeric pgtype.Numeric
	if err := weightNumeric.Scan(weightValue); err != nil {
		http.Error(w, "Invalid weight value", http.StatusBadRequest)
		return
	}

	skuID := r.PathValue("sku_id")
	if skuID != "" {
		// Update existing SKU
		var skuUUID pgtype.UUID
		if err := skuUUID.Scan(skuID); err != nil {
			http.Error(w, "Invalid SKU ID", http.StatusBadRequest)
			return
		}

		_, err = h.repo.UpdateProductSKU(r.Context(), repository.UpdateProductSKUParams{
			TenantID:          h.tenantID,
			ID:                skuUUID,
			Sku:               skuCode,
			WeightValue:       weightNumeric,
			WeightUnit:        weightUnit,
			Grind:             grind,
			BasePriceCents:    basePriceCents,
			InventoryQuantity: int32(inventoryQty),
			InventoryPolicy:   inventoryPolicy,
			LowStockThreshold: lowStockThreshold,
			IsActive:          isActive,
			WeightGrams:       weightGrams,
			RequiresShipping:  true,
		})
		if err != nil {
			http.Error(w, "Failed to update SKU: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
	} else {
		// Create new SKU
		newSKU, err := h.repo.CreateProductSKU(r.Context(), repository.CreateProductSKUParams{
			TenantID:          h.tenantID,
			ProductID:         productUUID,
			Sku:               skuCode,
			WeightValue:       weightNumeric,
			WeightUnit:        weightUnit,
			Grind:             grind,
			BasePriceCents:    basePriceCents,
			InventoryQuantity: int32(inventoryQty),
			InventoryPolicy:   inventoryPolicy,
			LowStockThreshold: lowStockThreshold,
			IsActive:          isActive,
			WeightGrams:       weightGrams,
			RequiresShipping:  true,
		})
		if err != nil {
			http.Error(w, "Failed to create SKU: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Auto-create price list entry for default (retail) price list
		if err := h.createDefaultPricing(r.Context(), newSKU.ID, basePriceCents); err != nil {
			// Log error but don't fail - admin can add prices manually
			fmt.Printf("Warning: failed to create default pricing: %v\n", err)
		}

		http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
	}
}

// createDefaultPricing automatically creates a price list entry for the default retail price list
func (h *SKUFormHandler) createDefaultPricing(ctx context.Context, skuID pgtype.UUID, priceCents int32) error {
	// Get default price list
	priceList, err := h.repo.GetDefaultPriceList(ctx, h.tenantID)
	if err != nil {
		return err
	}

	// Create price list entry
	_, err = h.repo.CreatePriceListEntry(ctx, repository.CreatePriceListEntryParams{
		TenantID:      h.tenantID,
		PriceListID:   priceList.ID,
		ProductSkuID:  skuID,
		PriceCents:    priceCents,
		CompareAtPriceCents: pgtype.Int4{Valid: false},
		IsAvailable:   true,
	})

	return err
}

// generateSKUCode creates a SKU code from product slug, size, and grind
func generateSKUCode(productSlug string, weight int, unit, grind string) string {
	grindAbbrev := grind
	if grind == "whole_bean" {
		grindAbbrev = "wb"
	} else if grind == "espresso" {
		grindAbbrev = "esp"
	} else if grind == "french_press" {
		grindAbbrev = "fp"
	} else if grind == "pour_over" {
		grindAbbrev = "po"
	} else if grind == "cold_brew" {
		grindAbbrev = "cb"
	}

	return fmt.Sprintf("%s-%d%s-%s", productSlug, weight, unit, grindAbbrev)
}

// calculateSuggestedPrice returns a suggested retail price based on weight
// Simple formula: $1.50/oz for coffee
func calculateSuggestedPrice(weight int, unit string) string {
	weightOz := float64(weight)

	// Convert to oz if needed
	switch unit {
	case "lb":
		weightOz = float64(weight) * 16
	case "g":
		weightOz = float64(weight) / 28.35
	case "kg":
		weightOz = float64(weight) * 35.274
	}

	// $1.50 per oz base, with volume discounts
	pricePerOz := 1.50
	if weightOz >= 16 {
		pricePerOz = 1.35 // Discount for 1lb+
	} else if weightOz >= 32 {
		pricePerOz = 1.20 // Discount for 2lb+
	}

	suggestedPrice := weightOz * pricePerOz
	return fmt.Sprintf("%.2f", suggestedPrice)
}

// calculateWeightGrams converts weight to grams for shipping calculations
func calculateWeightGrams(weight int, unit string) pgtype.Int4 {
	var grams int32
	switch unit {
	case "oz":
		grams = int32(float64(weight) * 28.35)
	case "lb":
		grams = int32(float64(weight) * 453.592)
	case "g":
		grams = int32(weight)
	case "kg":
		grams = int32(weight * 1000)
	default:
		grams = int32(float64(weight) * 28.35) // Default to oz
	}
	return pgtype.Int4{Int32: grams, Valid: true}
}
