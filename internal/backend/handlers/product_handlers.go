// internal/backend/handlers/product_handlers.go
package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/freyja/internal/backend/client"
	"github.com/dukerupert/freyja/internal/backend/templates/component"
	page_views "github.com/dukerupert/freyja/internal/backend/templates/page"
	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"

	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	freyjaClient *client.FreyjaClient
	queries      *database.Queries
}

func NewProductHandler(freyjaClient *client.FreyjaClient, queries *database.Queries) *ProductHandler {
	return &ProductHandler{
		freyjaClient: freyjaClient,
		queries:      queries,
	}
}

// ShowProductsPage renders the full products page
func (h *ProductHandler) ShowProductsPage(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse query parameters
	pageParam := c.QueryParam("page")
	page := 1
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	
	status := c.QueryParam("status")
	stockStatus := c.QueryParam("stock_status")
	search := c.QueryParam("search")
	
	limit := int32(20)
	offset := int32((page - 1) * int(limit))
	
	// Get products using your existing ListAllProducts query
	allProducts, err := h.queries.ListAllProducts(ctx, database.ListAllProductsParams{
		Limit:  limit * 5, // Get more to account for filtering
		Offset: 0,         // Start from beginning for filtering
	})
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}
	
	// Apply filters in memory (you can optimize this with SQL later)
	var filtered []database.ProductStockSummary
	for _, product := range allProducts {
		if h.matchesFilters(product, status, stockStatus, search) {
			filtered = append(filtered, product)
		}
	}
	
	// Apply pagination to filtered results
	total := int64(len(filtered))
	start := int(offset)
	end := start + int(limit)
	if start >= len(filtered) {
		filtered = []database.ProductStockSummary{}
	} else {
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}
	
	// Build pagination data
	var pagination *component.PaginationData
	if total > int64(limit) {
		pagination = &component.PaginationData{
			CurrentPage:  page,
			Total:        int(total),
			Start:        start + 1,
			End:          min(start+len(filtered), int(total)),
			HasPrevious:  page > 1,
			HasNext:      start+int(limit) < int(total),
			PreviousPage: page - 1,
			NextPage:     page + 1,
			Pages:        generatePageNumbers(page, int(total), int(limit)),
		}
	}
	
	data := page_views.ProductsPageData{
		Products:   filtered,
		Pagination: pagination,
	}
	
	// Render the Templ component directly
	component := page_views.ProductsPage(data)
	return component.Render(context.Background(), c.Response().Writer)
}

// matchesFilters checks if a product matches the given filters
func (h *ProductHandler) matchesFilters(product database.ProductStockSummary, status, stockStatus, search string) bool {
	// Status filter
	if status != "" {
		if status == "active" && !product.ProductActive {
			return false
		}
		if status == "inactive" && product.ProductActive {
			return false
		}
	}
	
	// Stock status filter
	if stockStatus != "" {
		switch stockStatus {
		case "in_stock":
			if !product.HasStock {
				return false
			}
		case "low_stock":
			if product.StockStatus != "low" {
				return false
			}
		case "out_of_stock":
			if product.HasStock {
				return false
			}
		}
	}
	
	// Search filter (case-insensitive)
	if search != "" {
		searchLower := strings.ToLower(search)
		nameLower := strings.ToLower(product.Name)
		descLower := ""
		if product.Description.Valid {
			descLower = strings.ToLower(product.Description.String)
		}
		
		if !strings.Contains(nameLower, searchLower) && !strings.Contains(descLower, searchLower) {
			return false
		}
	}
	
	return true
}

// GetProductDetail handles GET /admin/products/:id
// Returns HTML for HTMX to inject into #main-content
func (h *ProductHandler) GetProductDetail(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse product ID
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return page_views.ErrorState("Invalid product ID", "bg-red-50 border-red-200 text-red-800").Render(ctx, c.Response().Writer)
	}

	// Get product with full details
	productSummary, err := h.queries.GetProductWithSummary(ctx, int32(id))
	if err != nil {
		if err.Error() == "product not found" {
			return page_views.ErrorState("Product not found", "bg-yellow-50 border-yellow-200 text-yellow-800").Render(ctx, c.Response().Writer)
		}

		c.Logger().Error("Failed to get product details: ", err)
		return page_views.ErrorState("Failed to load product details", "bg-red-50 border-red-200 text-red-800").Render(ctx, c.Response().Writer)
	}

	// Get variants for this product
	variants, err := h.queries.GetActiveVariantsByProduct(ctx, int32(id))
	if err != nil {
		c.Logger().Error("Failed to get product variants: ", err)
		// Continue without variants rather than failing completely
		variants = []database.ProductVariants{}
	}

	// Set content type to HTML
	c.Response().Header().Set("Content-Type", "text/html")

	// convert productSummary 
	product := h.convertToProductSummary(productSummary)
	
	// Render the templ component
	return page_views.ProductDetailPage(product, variants).Render(ctx, c.Response().Writer)
}

func (h *ProductHandler) convertToProductSummary(dbSummary database.ProductStockSummary) *interfaces.ProductSummary {
	summary := &interfaces.ProductSummary{
		ProductID:        dbSummary.ProductID,
		Name:             dbSummary.Name,
		Description:      dbSummary.Description,
		ProductActive:    dbSummary.ProductActive,
		HasStock:         dbSummary.HasStock,
		StockStatus:      dbSummary.StockStatus,
		AvailableOptions: dbSummary.AvailableOptions,
	}

	// Handle nullable interface{} fields safely
	if dbSummary.TotalStock != nil {
		if val, ok := dbSummary.TotalStock.(int64); ok {
			summary.TotalStock = int32(val)
		}
	}

	if dbSummary.VariantsInStock != nil {
		if val, ok := dbSummary.VariantsInStock.(int64); ok {
			summary.VariantsInStock = int32(val)
		}
	}

	if dbSummary.TotalVariants != nil {
		if val, ok := dbSummary.TotalVariants.(int64); ok {
			summary.TotalVariants = int32(val)
		}
	}

	if dbSummary.MinPrice != nil {
		if val, ok := dbSummary.MinPrice.(int64); ok {
			summary.MinPrice = int32(val)
		}
	}

	if dbSummary.MaxPrice != nil {
		if val, ok := dbSummary.MaxPrice.(int64); ok {
			summary.MaxPrice = int32(val)
		}
	}

	return summary
}

// Helper function to generate page numbers for pagination
func generatePageNumbers(currentPage, total, limit int) []int {
	totalPages := (total + limit - 1) / limit
	
	// Show max 7 pages
	start := max(1, currentPage-3)
	end := min(totalPages, start+6)
	
	// Adjust start if we're near the end
	if end-start < 6 {
		start = max(1, end-6)
	}
	
	var pages []int
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	
	return pages
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}