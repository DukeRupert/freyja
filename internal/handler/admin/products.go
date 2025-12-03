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

// ProductHandler handles all product-related admin routes
type ProductHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewProductHandler creates a new product handler
func NewProductHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *ProductHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// List handles GET /admin/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.ListAllProducts(r.Context(), h.tenantID)
	if err != nil {
		http.Error(w, "Failed to load products", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Products":    products,
	}

	h.renderer.RenderHTTP(w, "admin/products", data)
}

// Detail handles GET /admin/products/{id}
func (h *ProductHandler) Detail(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
		TenantID: h.tenantID,
		ID:       productUUID,
	})
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

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
		weightFormatted := ""
		if sku.WeightValue.Valid {
			f, err := sku.WeightValue.Float64Value()
			if err == nil && f.Valid {
				weightStr := fmt.Sprintf("%.2f", f.Float64)
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
			TrackInventory:       sku.InventoryPolicy == "deny",
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

// ShowForm handles GET /admin/products/new and GET /admin/products/{id}/edit
func (h *ProductHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")

	var product repository.Product
	var tastingNotesString string

	if productID != "" {
		var productUUID pgtype.UUID
		if err := productUUID.Scan(productID); err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		p, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
			TenantID: h.tenantID,
			ID:       productUUID,
		})
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		product = p

		if len(product.TastingNotes) > 0 {
			tastingNotesString = strings.Join(product.TastingNotes, ", ")
		}
	} else {
		product.Status = "draft"
		product.Visibility = "public"
	}

	data := map[string]interface{}{
		"CurrentPath":        r.URL.Path,
		"Product":            product,
		"TastingNotesString": tastingNotesString,
	}

	h.renderer.RenderHTTP(w, "admin/product_form", data)
}

// HandleForm handles POST /admin/products/new and POST /admin/products/{id}/edit
func (h *ProductHandler) HandleForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	productID := r.PathValue("id")
	isEdit := productID != ""

	tastingNotesStr := strings.TrimSpace(r.FormValue("tasting_notes"))
	var tastingNotes []string
	if tastingNotesStr != "" {
		parts := strings.Split(tastingNotesStr, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				tastingNotes = append(tastingNotes, trimmed)
			}
		}
	}

	if isEdit {
		var productUUID pgtype.UUID
		if err := productUUID.Scan(productID); err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		elevationMin := pgtype.Int4{}
		if minStr := r.FormValue("elevation_min"); minStr != "" {
			if min, err := strconv.Atoi(minStr); err == nil {
				elevationMin.Int32 = int32(min)
				elevationMin.Valid = true
			}
		}

		elevationMax := pgtype.Int4{}
		if maxStr := r.FormValue("elevation_max"); maxStr != "" {
			if max, err := strconv.Atoi(maxStr); err == nil {
				elevationMax.Int32 = int32(max)
				elevationMax.Valid = true
			}
		}

		sortOrder := int32(0)
		if sortStr := r.FormValue("sort_order"); sortStr != "" {
			if sort, err := strconv.Atoi(sortStr); err == nil {
				sortOrder = int32(sort)
			}
		}

		_, err := h.repo.UpdateProduct(r.Context(), repository.UpdateProductParams{
			TenantID:         h.tenantID,
			ID:               productUUID,
			Name:             r.FormValue("name"),
			Slug:             r.FormValue("slug"),
			ShortDescription: makePgText(r.FormValue("short_description")),
			Description:      makePgText(r.FormValue("description")),
			Status:           r.FormValue("status"),
			Visibility:       r.FormValue("visibility"),
			Origin:           makePgText(r.FormValue("origin")),
			Region:           makePgText(r.FormValue("region")),
			Producer:         makePgText(r.FormValue("producer")),
			Process:          makePgText(r.FormValue("process")),
			RoastLevel:       makePgText(r.FormValue("roast_level")),
			TastingNotes:     tastingNotes,
			ElevationMin:     elevationMin,
			ElevationMax:     elevationMax,
			SortOrder:        sortOrder,
		})
		if err != nil {
			http.Error(w, "Failed to update product: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
	} else {
		elevationMin := pgtype.Int4{}
		if minStr := r.FormValue("elevation_min"); minStr != "" {
			if min, err := strconv.Atoi(minStr); err == nil {
				elevationMin.Int32 = int32(min)
				elevationMin.Valid = true
			}
		}

		elevationMax := pgtype.Int4{}
		if maxStr := r.FormValue("elevation_max"); maxStr != "" {
			if max, err := strconv.Atoi(maxStr); err == nil {
				elevationMax.Int32 = int32(max)
				elevationMax.Valid = true
			}
		}

		sortOrder := int32(0)
		if sortStr := r.FormValue("sort_order"); sortStr != "" {
			if sort, err := strconv.Atoi(sortStr); err == nil {
				sortOrder = int32(sort)
			}
		}

		isWhiteLabel := false
		baseProductID := pgtype.UUID{Valid: false}
		whiteLabelCustomerID := pgtype.UUID{Valid: false}

		_, err := h.repo.CreateProduct(r.Context(), repository.CreateProductParams{
			TenantID:             h.tenantID,
			Name:                 r.FormValue("name"),
			Slug:                 r.FormValue("slug"),
			ShortDescription:     makePgText(r.FormValue("short_description")),
			Description:          makePgText(r.FormValue("description")),
			Status:               r.FormValue("status"),
			Visibility:           r.FormValue("visibility"),
			Origin:               makePgText(r.FormValue("origin")),
			Region:               makePgText(r.FormValue("region")),
			Producer:             makePgText(r.FormValue("producer")),
			Process:              makePgText(r.FormValue("process")),
			RoastLevel:           makePgText(r.FormValue("roast_level")),
			TastingNotes:         tastingNotes,
			ElevationMin:         elevationMin,
			ElevationMax:         elevationMax,
			IsWhiteLabel:         isWhiteLabel,
			BaseProductID:        baseProductID,
			WhiteLabelCustomerID: whiteLabelCustomerID,
			SortOrder:            sortOrder,
		})
		if err != nil {
			http.Error(w, "Failed to create product: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
	}
}

// ShowSKUForm handles GET /admin/products/{product_id}/skus/new and GET /admin/products/{product_id}/skus/{sku_id}/edit
func (h *ProductHandler) ShowSKUForm(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
		TenantID: h.tenantID,
		ID:       productUUID,
	})
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	skuID := r.PathValue("sku_id")
	var sku repository.ProductSku
	var basePriceDollars string

	if skuID != "" {
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

	suggestedPrice := ""
	if skuID == "" {
		suggestedPrice = calculateSuggestedPrice(12, "oz")
	}

	data := map[string]interface{}{
		"CurrentPath":      r.URL.Path,
		"Product":          product,
		"SKU":              sku,
		"BasePriceDollars": basePriceDollars,
		"SuggestedPrice":   suggestedPrice,
	}

	h.renderer.RenderHTTP(w, "admin/sku_form", data)
}

// HandleSKUForm handles POST /admin/products/{product_id}/skus/new and POST /admin/products/{product_id}/skus/{sku_id}/edit
func (h *ProductHandler) HandleSKUForm(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.repo.GetProductByID(r.Context(), repository.GetProductByIDParams{
		TenantID: h.tenantID,
		ID:       productUUID,
	})
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	weightValue, _ := strconv.Atoi(r.FormValue("weight_value"))
	weightUnit := r.FormValue("weight_unit")
	grind := r.FormValue("grind")
	basePriceStr := r.FormValue("base_price")
	inventoryQty, _ := strconv.Atoi(r.FormValue("inventory_quantity"))
	lowStockStr := r.FormValue("low_stock_threshold")
	inventoryPolicy := r.FormValue("inventory_policy")
	isActive := r.FormValue("is_active") == "true"

	basePriceDollars, err := strconv.ParseFloat(basePriceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}
	basePriceCents := int32(math.Round(basePriceDollars * 100))

	skuCode := strings.TrimSpace(r.FormValue("sku"))
	if skuCode == "" {
		skuCode = generateSKUCode(product.Slug, weightValue, weightUnit, grind)
	}

	lowStockThreshold := pgtype.Int4{}
	if lowStockStr != "" {
		if threshold, err := strconv.Atoi(lowStockStr); err == nil {
			lowStockThreshold.Int32 = int32(threshold)
			lowStockThreshold.Valid = true
		}
	}

	weightGrams := calculateWeightGrams(weightValue, weightUnit)

	var weightNumeric pgtype.Numeric
	if err := weightNumeric.Scan(weightValue); err != nil {
		http.Error(w, "Invalid weight value", http.StatusBadRequest)
		return
	}

	skuID := r.PathValue("sku_id")
	if skuID != "" {
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

		if err := h.createDefaultPricing(r.Context(), newSKU.ID, basePriceCents); err != nil {
			fmt.Printf("Warning: failed to create default pricing: %v\n", err)
		}

		http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
	}
}

// createDefaultPricing automatically creates a price list entry for the default retail price list
func (h *ProductHandler) createDefaultPricing(ctx context.Context, skuID pgtype.UUID, priceCents int32) error {
	priceList, err := h.repo.GetDefaultPriceList(ctx, h.tenantID)
	if err != nil {
		return err
	}

	_, err = h.repo.CreatePriceListEntry(ctx, repository.CreatePriceListEntryParams{
		TenantID:            h.tenantID,
		PriceListID:         priceList.ID,
		ProductSkuID:        skuID,
		PriceCents:          priceCents,
		CompareAtPriceCents: pgtype.Int4{Valid: false},
		IsAvailable:         true,
	})

	return err
}

// makePgText creates a pgtype.Text from a string
func makePgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
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
func calculateSuggestedPrice(weight int, unit string) string {
	weightOz := float64(weight)

	switch unit {
	case "lb":
		weightOz = float64(weight) * 16
	case "g":
		weightOz = float64(weight) / 28.35
	case "kg":
		weightOz = float64(weight) * 35.274
	}

	pricePerOz := 1.50
	if weightOz >= 16 {
		pricePerOz = 1.35
	} else if weightOz >= 32 {
		pricePerOz = 1.20
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
		grams = int32(float64(weight) * 28.35)
	}
	return pgtype.Int4{Int32: grams, Valid: true}
}
