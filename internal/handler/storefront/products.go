package storefront

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/telemetry"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductHandler handles all product-related pages:
// - Product listing with filters
// - Product detail view
// - Subscription product selection (public)
type ProductHandler struct {
	productService domain.ProductService
	repo           repository.Querier
	renderer       *handler.Renderer
	tenantID       pgtype.UUID
}

// NewProductHandler creates a new consolidated product handler
func NewProductHandler(
	productService domain.ProductService,
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID string,
) *ProductHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductHandler{
		productService: productService,
		repo:           repo,
		renderer:       renderer,
		tenantID:       tenantUUID,
	}
}

// =============================================================================
// Product List
// =============================================================================

// ProductDisplay wraps product data for template rendering
type ProductDisplay struct {
	ID               pgtype.UUID
	TenantID         pgtype.UUID
	Name             string
	Slug             string
	ShortDescription pgtype.Text
	Origin           pgtype.Text
	RoastLevel       pgtype.Text
	TastingNotes     []string
	SortOrder        int32
	ImageURL         pgtype.Text
	ImageAlt         pgtype.Text
	BasePrice        pgtype.Int4
}

// convertToInt4 converts an interface{} (from subquery result) to pgtype.Int4
func convertToInt4(v interface{}) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	switch val := v.(type) {
	case int32:
		return pgtype.Int4{Int32: val, Valid: true}
	case int64:
		return pgtype.Int4{Int32: int32(val), Valid: true}
	case int:
		return pgtype.Int4{Int32: int32(val), Valid: true}
	default:
		return pgtype.Int4{Valid: false}
	}
}

// List handles GET /products - shows product listing with filters
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := h.tenantID.String()

	// Parse filter params
	roastLevel := r.URL.Query().Get("roast")
	origin := r.URL.Query().Get("origin")
	tastingNote := r.URL.Query().Get("note")

	// Track product searches/filters
	if telemetry.Business != nil {
		filterType := "none"
		if roastLevel != "" {
			filterType = "roast"
		} else if origin != "" {
			filterType = "origin"
		} else if tastingNote != "" {
			filterType = "note"
		}
		telemetry.Business.ProductSearches.WithLabelValues(tenantID, filterType).Inc()
	}

	// Get filter options for the UI
	filterOptions, err := h.repo.GetProductFilterOptions(ctx, h.tenantID)
	if err != nil {
		filterOptions = repository.GetProductFilterOptionsRow{}
	}

	var products []ProductDisplay

	// Check if any filters are applied
	if roastLevel != "" || origin != "" || tastingNote != "" {
		// Use filtered query
		var roastLevelParam, originParam, tastingNoteParam pgtype.Text
		if roastLevel != "" {
			roastLevelParam = pgtype.Text{String: roastLevel, Valid: true}
		}
		if origin != "" {
			originParam = pgtype.Text{String: origin, Valid: true}
		}
		if tastingNote != "" {
			tastingNoteParam = pgtype.Text{String: tastingNote, Valid: true}
		}

		filteredProducts, err := h.repo.ListActiveProductsFiltered(ctx, repository.ListActiveProductsFilteredParams{
			TenantID:    h.tenantID,
			RoastLevel:  roastLevelParam,
			Origin:      originParam,
			TastingNote: tastingNoteParam,
		})
		if err != nil {
			slog.Error("failed to list filtered products", "error", err, "roast", roastLevel, "origin", origin, "note", tastingNote)
			handler.InternalErrorResponse(w, r, err)
			return
		}

		products = make([]ProductDisplay, len(filteredProducts))
		for i, p := range filteredProducts {
			products[i] = ProductDisplay{
				ID:               p.ID,
				TenantID:         p.TenantID,
				Name:             p.Name,
				Slug:             p.Slug,
				ShortDescription: p.ShortDescription,
				Origin:           p.Origin,
				RoastLevel:       p.RoastLevel,
				TastingNotes:     p.TastingNotes,
				SortOrder:        p.SortOrder,
				ImageURL:         p.PrimaryImageUrl,
				ImageAlt:         p.PrimaryImageAlt,
				BasePrice:        convertToInt4(p.BasePrice),
			}
		}
	} else {
		// No filters - use original query
		allProducts, err := h.productService.ListProducts(ctx)
		if err != nil {
			handler.InternalErrorResponse(w, r, err)
			return
		}

		products = make([]ProductDisplay, len(allProducts))
		for i, p := range allProducts {
			products[i] = ProductDisplay{
				ID:               p.ID,
				TenantID:         p.TenantID,
				Name:             p.Name,
				Slug:             p.Slug,
				ShortDescription: p.ShortDescription,
				Origin:           p.Origin,
				RoastLevel:       p.RoastLevel,
				TastingNotes:     p.TastingNotes,
				SortOrder:        p.SortOrder,
				ImageURL:         p.PrimaryImageURL,
				ImageAlt:         p.PrimaryImageAlt,
			}
		}
	}

	// Calculate active filter count
	activeFilters := 0
	if roastLevel != "" {
		activeFilters++
	}
	if origin != "" {
		activeFilters++
	}
	if tastingNote != "" {
		activeFilters++
	}

	data := BaseTemplateData(r)
	data["Products"] = products
	data["RoastLevels"] = filterOptions.RoastLevels
	data["Origins"] = filterOptions.Origins
	data["TastingNotes"] = filterOptions.TastingNotes
	data["SelectedRoast"] = roastLevel
	data["SelectedOrigin"] = origin
	data["SelectedNote"] = tastingNote
	data["ActiveFilterCount"] = activeFilters
	data["HasFilters"] = activeFilters > 0

	h.renderer.RenderHTTP(w, "storefront/products", data)
}

// =============================================================================
// Product Detail
// =============================================================================

// Detail handles GET /products/{slug} - shows product detail page
func (h *ProductHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	tenantID := h.tenantID.String()

	if slug == "" {
		handler.NotFoundResponse(w, r)
		return
	}

	detail, err := h.productService.GetProductDetail(ctx, slug)
	if err != nil {
		// Service now returns domain errors - ErrProductNotFound maps to 404
		if domain.ErrorCode(err) == domain.ENOTFOUND {
			handler.ErrorResponse(w, r, err)
			return
		}
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Track product view
	if telemetry.Business != nil {
		telemetry.Business.ProductViews.WithLabelValues(tenantID, slug).Inc()
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

// =============================================================================
// Subscription Products (Public)
// =============================================================================

// SubscriptionProduct wraps product data for subscription selection display
type SubscriptionProduct struct {
	ID               pgtype.UUID
	Name             string
	Slug             string
	ShortDescription string
	Origin           string
	RoastLevel       string
	ImageURL         string
	SKUs             []SubscriptionSKU
}

// SubscriptionSKU contains SKU info for subscription selection
type SubscriptionSKU struct {
	ID          pgtype.UUID
	SKU         string
	WeightValue string
	WeightUnit  string
	Grind       string
	PriceCents  int32
	DisplayName string // e.g., "12oz - Whole Bean"
}

// SubscribeProducts handles GET /subscribe - shows products available for subscription
func (h *ProductHandler) SubscribeProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := h.tenantID.String()

	// Track subscription page view
	if telemetry.Business != nil {
		telemetry.Business.SubscribePageView.WithLabelValues(tenantID).Inc()
	}

	// Get all active products
	products, err := h.productService.ListProducts(ctx)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get default price list
	priceList, err := h.repo.GetDefaultPriceList(ctx, h.tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Build subscription products with SKU options
	subscriptionProducts := make([]SubscriptionProduct, 0, len(products))

	for _, p := range products {
		// Get SKUs for this product
		skus, err := h.repo.GetProductSKUs(ctx, p.ID)
		if err != nil {
			continue // Skip products with no SKUs
		}

		if len(skus) == 0 {
			continue
		}

		subscriptionSKUs := make([]SubscriptionSKU, 0, len(skus))

		for _, sku := range skus {
			// Get price for this SKU
			price, err := h.repo.GetPriceForSKU(ctx, repository.GetPriceForSKUParams{
				PriceListID:  priceList.ID,
				ProductSkuID: sku.ID,
			})
			if err != nil {
				continue // Skip SKUs without pricing
			}

			// Format weight value
			weightStr := ""
			if sku.WeightValue.Valid {
				f, err := sku.WeightValue.Float64Value()
				if err == nil && f.Valid {
					weightStr = fmt.Sprintf("%.0f", f.Float64)
				}
			}

			// Build display name (e.g., "12oz - Whole Bean")
			displayName := fmt.Sprintf("%s%s - %s", weightStr, sku.WeightUnit, sku.Grind)

			subscriptionSKUs = append(subscriptionSKUs, SubscriptionSKU{
				ID:          sku.ID,
				SKU:         sku.Sku,
				WeightValue: weightStr,
				WeightUnit:  sku.WeightUnit,
				Grind:       sku.Grind,
				PriceCents:  price.PriceCents,
				DisplayName: displayName,
			})
		}

		if len(subscriptionSKUs) > 0 {
			subscriptionProducts = append(subscriptionProducts, SubscriptionProduct{
				ID:               p.ID,
				Name:             p.Name,
				Slug:             p.Slug,
				ShortDescription: p.ShortDescription.String,
				Origin:           p.Origin.String,
				RoastLevel:       p.RoastLevel.String,
				ImageURL:         p.PrimaryImageURL.String,
				SKUs:             subscriptionSKUs,
			})
		}
	}

	data := BaseTemplateData(r)
	data["Products"] = subscriptionProducts
	data["BillingIntervals"] = service.ValidBillingIntervals

	h.renderer.RenderHTTP(w, "storefront/subscription_products", data)
}
