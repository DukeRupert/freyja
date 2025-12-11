package admin

import (
	"context"
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductHandler handles all product-related admin routes
type ProductHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	storage  storage.Storage
}

// NewProductHandler creates a new product handler
func NewProductHandler(repo repository.Querier, renderer *handler.Renderer, storage storage.Storage) *ProductHandler {
	return &ProductHandler{
		repo:     repo,
		renderer: renderer,
		storage:  storage,
	}
}

// List handles GET /admin/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	products, err := h.repo.ListAllProducts(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
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
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("id")
	if productID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Product ID required"))
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}

	product, err := h.repo.GetProductByID(ctx, repository.GetProductByIDParams{
		TenantID: tenantID,
		ID:       productUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	skus, err := h.repo.GetProductSKUs(r.Context(), productUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Fetch product images
	images, err := h.repo.GetProductImages(r.Context(), productUUID)
	if err != nil {
		// Non-fatal - just log and continue with empty images
		images = []repository.ProductImage{}
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
		"Images":      images,
		"CSRFToken":   middleware.GetCSRFToken(r.Context()),
	}

	h.renderer.RenderHTTP(w, "admin/product_detail", data)
}

// ShowForm handles GET /admin/products/new and GET /admin/products/{id}/edit
func (h *ProductHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("id")

	var product repository.Product
	var tastingNotesString string

	if productID != "" {
		var productUUID pgtype.UUID
		if err := productUUID.Scan(productID); err != nil {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
			return
		}

		p, err := h.repo.GetProductByID(ctx, repository.GetProductByIDParams{
			TenantID: tenantID,
			ID:       productUUID,
		})
		if err != nil {
			handler.NotFoundResponse(w, r)
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
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
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
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
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

		_, err := h.repo.UpdateProduct(ctx, repository.UpdateProductParams{
			TenantID:         tenantID,
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
			handler.InternalErrorResponse(w, r, err)
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

		_, err := h.repo.CreateProduct(ctx, repository.CreateProductParams{
			TenantID:             tenantID,
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
			handler.InternalErrorResponse(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
	}
}

// ShowSKUForm handles GET /admin/products/{product_id}/skus/new and GET /admin/products/{product_id}/skus/{sku_id}/edit
func (h *ProductHandler) ShowSKUForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("product_id")
	if productID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Product ID required"))
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}

	product, err := h.repo.GetProductByID(ctx, repository.GetProductByIDParams{
		TenantID: tenantID,
		ID:       productUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	skuID := r.PathValue("sku_id")
	var sku repository.ProductSku
	var basePriceDollars string

	if skuID != "" {
		var skuUUID pgtype.UUID
		if err := skuUUID.Scan(skuID); err != nil {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid SKU ID"))
			return
		}

		sku, err = h.repo.GetSKUByID(r.Context(), skuUUID)
		if err != nil {
			handler.NotFoundResponse(w, r)
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
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	productID := r.PathValue("product_id")
	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}

	product, err := h.repo.GetProductByID(ctx, repository.GetProductByIDParams{
		TenantID: tenantID,
		ID:       productUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid price"))
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid weight value"))
		return
	}

	skuID := r.PathValue("sku_id")
	if skuID != "" {
		var skuUUID pgtype.UUID
		if err := skuUUID.Scan(skuID); err != nil {
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid SKU ID"))
			return
		}

		_, err = h.repo.UpdateProductSKU(ctx, repository.UpdateProductSKUParams{
			TenantID:          tenantID,
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
			handler.InternalErrorResponse(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
	} else {
		newSKU, err := h.repo.CreateProductSKU(ctx, repository.CreateProductSKUParams{
			TenantID:          tenantID,
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
			handler.InternalErrorResponse(w, r, err)
			return
		}

		if err := h.createDefaultPricing(ctx, tenantID, newSKU.ID, basePriceCents); err != nil {
			fmt.Printf("Warning: failed to create default pricing: %v\n", err)
		}

		http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
	}
}

// createDefaultPricing automatically creates a price list entry for the default retail price list
func (h *ProductHandler) createDefaultPricing(ctx context.Context, tenantID pgtype.UUID, skuID pgtype.UUID, priceCents int32) error {
	priceList, err := h.repo.GetDefaultPriceList(ctx, tenantID)
	if err != nil {
		return err
	}

	_, err = h.repo.CreatePriceListEntry(ctx, repository.CreatePriceListEntryParams{
		TenantID:            tenantID,
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

// validateImageUpload checks file type and size limits
func validateImageUpload(fileHeader *multipart.FileHeader) error {
	const maxSize = 5 * 1024 * 1024
	if fileHeader.Size > maxSize {
		return fmt.Errorf("image must be smaller than 5MB (current: %.1fMB)", float64(fileHeader.Size)/(1024*1024))
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExts[ext] {
		return fmt.Errorf("only JPEG, PNG, and WebP images are supported")
	}

	return nil
}

// extractImageMetadata reads image dimensions
func extractImageMetadata(file multipart.File) (width, height int, err error) {
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}

// generateImageKey creates storage key with pattern: products/{tenant_id}/{product_id}/{uuid}.{ext}
func generateImageKey(tenantID, productID pgtype.UUID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	// Sanitize extension - only allow known safe extensions
	allowedExts := map[string]string{".jpg": ".jpg", ".jpeg": ".jpg", ".png": ".png", ".webp": ".webp"}
	safeExt, ok := allowedExts[ext]
	if !ok {
		safeExt = ".jpg" // default to safe extension
	}
	return fmt.Sprintf("products/%s/%s/%s%s",
		formatUUID(tenantID), formatUUID(productID), uuid.New().String(), safeExt)
}

// UploadImage handles POST /admin/products/{id}/images/upload
func (h *ProductHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("id")
	if productID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Product ID required"))
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}

	_, err := h.repo.GetProductByID(ctx, repository.GetProductByIDParams{
		TenantID: tenantID,
		ID:       productUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "No image file provided"))
		return
	}
	defer file.Close()

	if err := validateImageUpload(fileHeader); err != nil {
		h.renderImageError(w, err.Error())
		return
	}

	// Check max images per product
	existingImages, _ := h.repo.GetProductImages(ctx, productUUID)
	const maxImagesPerProduct = 20
	if len(existingImages) >= maxImagesPerProduct {
		h.renderImageError(w, fmt.Sprintf("Maximum of %d images per product reached", maxImagesPerProduct))
		return
	}

	width, height, metaErr := extractImageMetadata(file)
	_, _ = file.Seek(0, io.SeekStart)

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])
	_, _ = file.Seek(0, io.SeekStart)

	key := generateImageKey(tenantID, productUUID, fileHeader.Filename)

	url, err := h.storage.Put(ctx, key, file, contentType)
	if err != nil {
		h.renderImageError(w, "Failed to store image. Please try again.")
		return
	}

	// Use existingImages from max check above for sort order and primary determination
	sortOrder := int32(len(existingImages))
	isPrimary := len(existingImages) == 0

	widthInt := pgtype.Int4{Int32: int32(width), Valid: metaErr == nil}
	heightInt := pgtype.Int4{Int32: int32(height), Valid: metaErr == nil}
	fileSizeInt := pgtype.Int4{Int32: int32(fileHeader.Size), Valid: true}

	_, err = h.repo.CreateProductImage(ctx, repository.CreateProductImageParams{
		TenantID:  tenantID,
		ProductID: productUUID,
		Url:       url,
		AltText:   pgtype.Text{Valid: false},
		Width:     widthInt,
		Height:    heightInt,
		FileSize:  fileSizeInt,
		SortOrder: sortOrder,
		IsPrimary: isPrimary,
	})
	if err != nil {
		// Clean up orphaned file (best-effort, ignore error)
		_ = h.storage.Delete(ctx, key)
		h.renderImageError(w, "Failed to save image. Please try again.")
		return
	}

	h.renderImageGallery(w, r, tenantID, productUUID)
}

// DeleteImage handles DELETE /admin/products/{product_id}/images/{image_id}
func (h *ProductHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("product_id")
	imageID := r.PathValue("image_id")

	var productUUID, imageUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}
	if err := imageUUID.Scan(imageID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid image ID"))
		return
	}

	images, err := h.repo.GetProductImages(ctx, productUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	var imageURL string
	for _, img := range images {
		if img.ID == imageUUID {
			imageURL = img.Url
			break
		}
	}

	if err := h.repo.DeleteProductImage(ctx, repository.DeleteProductImageParams{
		TenantID: tenantID,
		ID:       imageUUID,
	}); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if imageURL != "" {
		key := strings.TrimPrefix(imageURL, "/uploads/")
		_ = h.storage.Delete(ctx, key)
	}

	h.renderImageGallery(w, r, tenantID, productUUID)
}

// SetPrimary handles POST /admin/products/{product_id}/images/{image_id}/primary
func (h *ProductHandler) SetPrimary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("product_id")
	imageID := r.PathValue("image_id")

	var productUUID, imageUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}
	if err := imageUUID.Scan(imageID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid image ID"))
		return
	}

	if err := h.repo.SetPrimaryImage(ctx, repository.SetPrimaryImageParams{
		TenantID: tenantID,
		ID:       imageUUID,
	}); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	h.renderImageGallery(w, r, tenantID, productUUID)
}

// UpdateImageMetadata handles POST /admin/products/{product_id}/images/{image_id}/metadata
// Updates alt text, width, and height for an image
func (h *ProductHandler) UpdateImageMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	productID := r.PathValue("product_id")
	imageID := r.PathValue("image_id")

	var productUUID, imageUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid product ID"))
		return
	}
	if err := imageUUID.Scan(imageID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid image ID"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	altText := r.FormValue("alt_text")
	widthStr := r.FormValue("width")
	heightStr := r.FormValue("height")

	images, err := h.repo.GetProductImages(ctx, productUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	var currentImage repository.ProductImage
	for _, img := range images {
		if img.ID == imageUUID {
			currentImage = img
			break
		}
	}

	// Parse width/height, keeping existing values if not provided or invalid
	width := currentImage.Width
	if widthStr != "" {
		if w, err := strconv.Atoi(widthStr); err == nil && w > 0 {
			width = pgtype.Int4{Int32: int32(w), Valid: true}
		}
	}

	height := currentImage.Height
	if heightStr != "" {
		if h, err := strconv.Atoi(heightStr); err == nil && h > 0 {
			height = pgtype.Int4{Int32: int32(h), Valid: true}
		}
	}

	_, err = h.repo.UpdateProductImage(ctx, repository.UpdateProductImageParams{
		TenantID:  tenantID,
		ID:        imageUUID,
		Url:       currentImage.Url,
		AltText:   pgtype.Text{String: altText, Valid: altText != ""},
		Width:     width,
		Height:    height,
		FileSize:  currentImage.FileSize,
		SortOrder: currentImage.SortOrder,
		IsPrimary: currentImage.IsPrimary,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	h.renderImageGallery(w, r, tenantID, productUUID)
}

// renderImageGallery renders just the image gallery section for htmx swap
func (h *ProductHandler) renderImageGallery(w http.ResponseWriter, r *http.Request, tenantID pgtype.UUID, productUUID pgtype.UUID) {
	ctx := r.Context()
	images, err := h.repo.GetProductImages(ctx, productUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	csrfToken := middleware.GetCSRFToken(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(images) == 0 {
		fmt.Fprint(w, `<p class="text-sm text-zinc-500 dark:text-zinc-400">No images yet. Upload your first product image above.</p>`)
		return
	}

	fmt.Fprint(w, `<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">`)
	for _, img := range images {
		productIDStr := formatUUID(productUUID)
		imageIDStr := formatUUID(img.ID)

		// Image card
		fmt.Fprint(w, `<div class="group relative flex flex-col overflow-hidden rounded-lg border border-zinc-200 bg-white dark:border-zinc-700 dark:bg-zinc-800">`)

		// Image container
		fmt.Fprint(w, `<div class="relative aspect-square">`)

		// Image - escape user-provided content to prevent XSS
		altText := ""
		if img.AltText.Valid {
			altText = html.EscapeString(img.AltText.String)
		}
		escapedURL := html.EscapeString(img.Url)
		fmt.Fprintf(w, `<img src="%s" alt="%s" class="h-full w-full object-cover">`, escapedURL, altText)

		// Default badge (positioned in top-right corner of image)
		if img.IsPrimary {
			fmt.Fprint(w, `<div class="absolute right-2 top-2"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 ring-1 ring-inset ring-blue-600/20 dark:bg-blue-900/50 dark:text-blue-300 dark:ring-blue-500/30">Default</span></div>`)
		}

		// Actions overlay
		fmt.Fprint(w, `<div class="absolute inset-0 flex items-center justify-center gap-2 bg-black/60 opacity-0 transition-opacity group-hover:opacity-100">`)
		if !img.IsPrimary {
			fmt.Fprintf(w, `<button hx-post="/admin/products/%s/images/%s/default" hx-target="#image-gallery" hx-swap="innerHTML" hx-vals='{"csrf_token": "%s"}' class="rounded-lg bg-white px-3 py-1.5 text-xs font-medium text-zinc-900 hover:bg-zinc-100">Set Default</button>`,
				productIDStr, imageIDStr, csrfToken)
		}
		fmt.Fprintf(w, `<button hx-delete="/admin/products/%s/images/%s" hx-target="#image-gallery" hx-swap="innerHTML" hx-confirm="Delete this image?" hx-vals='{"csrf_token": "%s"}' class="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">Delete</button>`,
			productIDStr, imageIDStr, csrfToken)
		fmt.Fprint(w, `</div>`)

		fmt.Fprint(w, `</div>`) // Close image container

		// Image info and alt text form
		fmt.Fprint(w, `<div class="flex flex-col gap-2 border-t border-zinc-200 p-3 dark:border-zinc-700">`)

		// Dimensions display
		if img.Width.Valid && img.Height.Valid {
			fmt.Fprintf(w, `<p class="text-xs text-zinc-500 dark:text-zinc-400">%d × %d px</p>`, img.Width.Int32, img.Height.Int32)
		}

		// Alt text form with label and save feedback
		fmt.Fprintf(w, `<form x-data="{ saved: false }" hx-post="/admin/products/%s/images/%s/metadata" hx-swap="none" @htmx:after-request="saved = true; setTimeout(() => saved = false, 2000)" class="space-y-1">`,
			productIDStr, imageIDStr)
		fmt.Fprintf(w, `<input type="hidden" name="csrf_token" value="%s">`, csrfToken)
		fmt.Fprintf(w, `<label for="alt-%s" class="block text-xs font-medium text-zinc-600 dark:text-zinc-400">Alt text</label>`, imageIDStr)
		fmt.Fprintf(w, `<div class="flex gap-2">`)
		fmt.Fprintf(w, `<input type="text" id="alt-%s" name="alt_text" value="%s" placeholder="Describe the image" class="flex-1 rounded-md border border-zinc-300 bg-white px-2 py-1 text-xs text-zinc-900 placeholder:text-zinc-400 dark:border-zinc-600 dark:bg-zinc-800 dark:text-white dark:placeholder:text-zinc-500">`,
			imageIDStr, altText)
		fmt.Fprint(w, `<button type="submit" class="rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors" :class="saved ? 'bg-green-600 text-white' : 'bg-zinc-900 text-white hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100'"><span x-show="!saved">Save</span><span x-show="saved" x-cloak>Saved!</span></button>`)
		fmt.Fprint(w, `</div>`)
		fmt.Fprint(w, `</form>`)

		fmt.Fprint(w, `</div>`) // Close info section
		fmt.Fprint(w, `</div>`) // Close card
	}
	fmt.Fprint(w, `</div>`)
}

// formatUUID returns the string representation of a pgtype.UUID
func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// renderImageError renders an error message for htmx swap
func (h *ProductHandler) renderImageError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<div class="rounded-lg bg-red-50 p-4 text-sm text-red-800 dark:bg-red-900/50 dark:text-red-200">Error: %s</div>`, message)
}
