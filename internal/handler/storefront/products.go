package storefront

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

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

// ProductListHandler handles the product listing page
type ProductListHandler struct {
	productService service.ProductService
	repo           repository.Querier
	tenantID       pgtype.UUID
	renderer       *handler.Renderer
}

// NewProductListHandler creates a new product list handler
func NewProductListHandler(productService service.ProductService, repo repository.Querier, tenantID string, renderer *handler.Renderer) *ProductListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProductListHandler{
		productService: productService,
		repo:           repo,
		tenantID:       tenantUUID,
		renderer:       renderer,
	}
}

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

// ServeHTTP handles GET /products
func (h *ProductListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filter params
	roastLevel := r.URL.Query().Get("roast")
	origin := r.URL.Query().Get("origin")

	// Get filter options for the UI
	filterOptions, err := h.repo.GetProductFilterOptions(ctx, h.tenantID)
	if err != nil {
		filterOptions = repository.GetProductFilterOptionsRow{}
	}

	var products []ProductDisplay

	// Check if any filters are applied
	if roastLevel != "" || origin != "" {
		// Use filtered query
		var roastLevelParam, originParam pgtype.Text
		if roastLevel != "" {
			roastLevelParam = pgtype.Text{String: roastLevel, Valid: true}
		}
		if origin != "" {
			originParam = pgtype.Text{String: origin, Valid: true}
		}

		filteredProducts, err := h.repo.ListActiveProductsFiltered(ctx, repository.ListActiveProductsFilteredParams{
			TenantID:   h.tenantID,
			RoastLevel: roastLevelParam,
			Origin:     originParam,
		})
		if err != nil {
			http.Error(w, "Failed to load products", http.StatusInternalServerError)
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
			http.Error(w, "Failed to load products", http.StatusInternalServerError)
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
				ImageURL:         p.PrimaryImageUrl,
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

	data := BaseTemplateData(r)
	data["Products"] = products
	data["RoastLevels"] = filterOptions.RoastLevels
	data["Origins"] = filterOptions.Origins
	data["SelectedRoast"] = roastLevel
	data["SelectedOrigin"] = origin
	data["ActiveFilterCount"] = activeFilters
	data["HasFilters"] = activeFilters > 0

	h.renderer.RenderHTTP(w, "storefront/products", data)
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
