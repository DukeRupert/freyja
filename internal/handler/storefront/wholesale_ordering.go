package storefront

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/hiri/internal/cookie"
	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/telemetry"
	"github.com/jackc/pgx/v5/pgtype"
)

// WholesaleOrderingHandler handles the wholesale ordering matrix view
type WholesaleOrderingHandler struct {
	repo         repository.Querier
	cartService  domain.CartService
	renderer     *handler.Renderer
	tenantID     pgtype.UUID
	cookieConfig *cookie.Config
}

// NewWholesaleOrderingHandler creates a new wholesale ordering handler
func NewWholesaleOrderingHandler(
	repo repository.Querier,
	cartService domain.CartService,
	renderer *handler.Renderer,
	tenantID string,
	cookieConfig *cookie.Config,
) *WholesaleOrderingHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &WholesaleOrderingHandler{
		repo:         repo,
		cartService:  cartService,
		renderer:     renderer,
		tenantID:     tenantUUID,
		cookieConfig: cookieConfig,
	}
}

// ProductGroup represents a product with its variants for the matrix view
type ProductGroup struct {
	ProductID   string
	ProductName string
	ProductSlug string
	Origin      string
	ImageURL    string
	SKUs        []WholesaleSKU
}

// WholesaleSKU represents a single SKU row in the matrix
type WholesaleSKU struct {
	SKUID            string
	SKUCode          string
	Weight           string
	Grind            string
	PriceCents       int32
	StockStatus      string
	InventoryQty     int32
}

// Order handles GET /wholesale/order - shows the wholesale ordering matrix
func (h *WholesaleOrderingHandler) Order(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := h.tenantID.String()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/wholesale/order", http.StatusSeeOther)
		return
	}

	// Check if user is wholesale or has approved wholesale application
	if user.AccountType != "wholesale" {
		// Check if they have a pending application
		if user.WholesaleApplicationStatus.Valid && user.WholesaleApplicationStatus.String == "pending" {
			http.Redirect(w, r, "/wholesale/status", http.StatusSeeOther)
			return
		}
		// Redirect to apply
		http.Redirect(w, r, "/wholesale/apply", http.StatusSeeOther)
		return
	}

	// Track wholesale page view
	if telemetry.Business != nil {
		telemetry.Business.ProductViews.WithLabelValues(tenantID, "wholesale_ordering").Inc()
	}

	// Get user's price list (or default if not set)
	var priceListID pgtype.UUID
	userPriceListID, err := h.repo.GetPriceListForUser(ctx, user.ID)
	if err == nil && userPriceListID.Valid {
		priceListID = userPriceListID
	} else {
		// Fall back to default price list
		priceList, err := h.repo.GetDefaultPriceList(ctx, h.tenantID)
		if err != nil {
			handler.InternalErrorResponse(w, r, err)
			return
		}
		priceListID = priceList.ID
	}

	// Get products with SKUs for wholesale ordering
	rows, err := h.repo.ListProductsWithSKUsForWholesale(ctx, repository.ListProductsWithSKUsForWholesaleParams{
		TenantID:    h.tenantID,
		PriceListID: priceListID,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Group SKUs by product
	productGroups := h.groupByProduct(rows)

	// Get current cart summary
	sessionID := GetSessionIDFromCookie(r)
	var cartSummary *domain.CartSummary
	if sessionID != "" {
		cart, err := h.cartService.GetCart(ctx, sessionID)
		if err == nil && cart != nil {
			cartSummary, _ = h.cartService.GetCartSummary(ctx, cart.ID.String())
		}
	}

	data := BaseTemplateData(r)
	data["Products"] = productGroups
	data["CartSummary"] = cartSummary
	data["User"] = user

	h.renderer.RenderHTTP(w, "storefront/wholesale_order", data)
}

// BatchAdd handles POST /wholesale/cart/batch - adds multiple items to cart
func (h *WholesaleOrderingHandler) BatchAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := h.tenantID.String()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	if user.AccountType != "wholesale" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EFORBIDDEN, "", "Wholesale account required"))
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data")
		return
	}

	// Get or create cart
	sessionID := GetSessionIDFromCookie(r)
	cart, newSessionID, err := h.cartService.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if newSessionID != sessionID {
		SetSessionCookie(w, newSessionID, h.cookieConfig)
	}

	// Process all quantity inputs
	// Form fields are named "qty[{sku_id}]"
	itemsAdded := 0
	var addErrors []string

	for key, values := range r.Form {
		if !strings.HasPrefix(key, "qty[") || !strings.HasSuffix(key, "]") {
			continue
		}

		// Extract SKU ID from key
		skuID := strings.TrimSuffix(strings.TrimPrefix(key, "qty["), "]")
		if skuID == "" {
			continue
		}

		// Get quantity
		qtyStr := values[0]
		if qtyStr == "" || qtyStr == "0" {
			continue
		}

		qty, err := strconv.Atoi(qtyStr)
		if err != nil || qty < 1 {
			continue
		}

		// Add to cart
		_, err = h.cartService.AddItem(ctx, cart.ID.String(), skuID, qty)
		if err != nil {
			addErrors = append(addErrors, fmt.Sprintf("Failed to add item: %v", err))
			continue
		}

		itemsAdded++
	}

	// Track wholesale cart additions
	if telemetry.Business != nil && itemsAdded > 0 {
		telemetry.Business.ProductAddToCart.WithLabelValues(tenantID, "wholesale_batch").Add(float64(itemsAdded))
	}

	if len(addErrors) > 0 && itemsAdded == 0 {
		h.renderError(w, r, "Failed to add items to cart")
		return
	}

	// Redirect or return success
	if r.Header.Get("HX-Request") == "true" {
		// Return updated cart summary for HTMX
		cartSummary, err := h.cartService.GetCartSummary(ctx, cart.ID.String())
		if err != nil {
			h.renderError(w, r, "Failed to get cart")
			return
		}

		w.Header().Set("HX-Trigger", "cartUpdated")
		h.renderer.RenderHTTP(w, "storefront/wholesale_cart_summary", map[string]interface{}{
			"CartSummary": cartSummary,
			"ItemsAdded":  itemsAdded,
		})
		return
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// groupByProduct groups the flat SKU rows into product groups
func (h *WholesaleOrderingHandler) groupByProduct(rows []repository.ListProductsWithSKUsForWholesaleRow) []ProductGroup {
	groupMap := make(map[string]*ProductGroup)
	var orderedKeys []string

	for _, row := range rows {
		productID := row.ProductID.String()

		group, exists := groupMap[productID]
		if !exists {
			group = &ProductGroup{
				ProductID:   productID,
				ProductName: row.ProductName,
				ProductSlug: row.ProductSlug,
				Origin:      row.ProductOrigin.String,
				ImageURL:    row.ProductImageUrl.String,
				SKUs:        []WholesaleSKU{},
			}
			groupMap[productID] = group
			orderedKeys = append(orderedKeys, productID)
		}

		// Format weight
		weight := ""
		if row.WeightValue.Valid {
			f, err := row.WeightValue.Float64Value()
			if err == nil && f.Valid {
				weight = fmt.Sprintf("%.0f %s", f.Float64, row.WeightUnit)
			}
		}

		// Determine stock status
		stockStatus := "In Stock"
		if row.InventoryQuantity <= 0 {
			if row.InventoryPolicy == "allow" {
				stockStatus = "Backorder"
			} else {
				stockStatus = "Out of Stock"
			}
		} else if row.LowStockThreshold.Valid && row.InventoryQuantity <= row.LowStockThreshold.Int32 {
			stockStatus = "Low Stock"
		}

		group.SKUs = append(group.SKUs, WholesaleSKU{
			SKUID:        row.SkuID.String(),
			SKUCode:      row.SkuCode,
			Weight:       weight,
			Grind:        row.Grind,
			PriceCents:   row.PriceCents,
			StockStatus:  stockStatus,
			InventoryQty: row.InventoryQuantity,
		})
	}

	// Return in original order
	result := make([]ProductGroup, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		result = append(result, *groupMap[key])
	}

	return result
}

// renderError sends an error response
func (h *WholesaleOrderingHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Retarget", "#error-message")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`<div class="rounded-md bg-red-50 p-4"><p class="text-sm text-red-700">%s</p></div>`, message)))
		return
	}
	handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "%s", message))
}
