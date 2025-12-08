package storefront

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/telemetry"
)

// CartHandler handles all cart-related storefront routes
type CartHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
	secure      bool
	tenantID    string
}

// NewCartHandler creates a new cart handler
func NewCartHandler(cartService service.CartService, renderer *handler.Renderer, secure bool, tenantID string) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		renderer:    renderer,
		secure:      secure,
		tenantID:    tenantID,
	}
}

// View handles GET /cart
func (h *CartHandler) View(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionID := GetSessionIDFromCookie(r)
	var summary *service.CartSummary

	if sessionID != "" {
		cart, err := h.cartService.GetCart(ctx, sessionID)
		if err != nil && domain.ErrorCode(err) != domain.ENOTFOUND {
			handler.InternalErrorResponse(w, r, err)
			return
		}

		if cart != nil {
			cartSummary, err := h.cartService.GetCartSummary(ctx, cart.ID.String())
			if err != nil {
				handler.InternalErrorResponse(w, r, err)
				return
			}
			summary = cartSummary
		}
	}

	data := BaseTemplateData(r)
	data["Summary"] = summary

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderer.RenderHTTP(w, "cart", data)
}

// Add handles POST /cart/add
func (h *CartHandler) Add(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	skuID := r.FormValue("sku_id")
	if skuID == "" {
		h.renderCartError(w, "Please select a size and grind option")
		return
	}

	quantityStr := r.FormValue("quantity")

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		h.renderCartError(w, "Invalid quantity")
		return
	}

	sessionID := GetSessionIDFromCookie(r)
	cart, newSessionID, err := h.cartService.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if newSessionID != sessionID {
		SetSessionCookie(w, newSessionID, h.secure)
	}

	_, err = h.cartService.AddItem(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		errCode := domain.ErrorCode(err)
		if errCode == domain.ENOTFOUND {
			h.renderCartError(w, "Product not found")
			return
		}
		if errCode == domain.EINVALID {
			h.renderCartError(w, domain.ErrorMessage(err))
			return
		}
		h.renderCartError(w, "Failed to add item")
		return
	}

	// Track add to cart
	if telemetry.Business != nil {
		telemetry.Business.CartUpdated.WithLabelValues(h.tenantID, "add").Inc()
		telemetry.Business.CartItemsAdd.WithLabelValues(h.tenantID).Add(float64(quantity))
	}

	tmpl, err := h.renderer.Execute("cart_added")
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "cart_added", nil); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}
}

// Update handles POST /cart/update
func (h *CartHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	skuID := r.FormValue("sku_id")
	quantityStr := r.FormValue("quantity")

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 0 {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid quantity"))
		return
	}

	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		handler.NotFoundResponse(w, r)
		return
	}

	cart, err := h.cartService.GetCart(ctx, sessionID)
	if err != nil {
		handler.ErrorResponse(w, r, err)
		return
	}

	summary, err := h.cartService.UpdateItemQuantity(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		h.renderCartError(w, "Failed to update item")
		return
	}

	// Track cart update
	if telemetry.Business != nil {
		telemetry.Business.CartUpdated.WithLabelValues(h.tenantID, "update_quantity").Inc()
	}

	tmpl, err := h.renderer.Execute("cart_summary")
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"Summary": summary,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "cart_summary", data); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}
}

// Remove handles POST /cart/remove
func (h *CartHandler) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	skuID := r.FormValue("sku_id")

	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		handler.NotFoundResponse(w, r)
		return
	}

	cart, err := h.cartService.GetCart(ctx, sessionID)
	if err != nil {
		handler.ErrorResponse(w, r, err)
		return
	}

	_, err = h.cartService.RemoveItem(ctx, cart.ID.String(), skuID)
	if err != nil {
		h.renderCartError(w, "Failed to remove item")
		return
	}

	// Track cart item removal
	if telemetry.Business != nil {
		telemetry.Business.CartUpdated.WithLabelValues(h.tenantID, "remove").Inc()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (h *CartHandler) renderCartError(w http.ResponseWriter, message string) {
	tmpl, err := h.renderer.Execute("cart_error")
	if err != nil {
		http.Error(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]interface{}{
		"Message": message,
	}
	if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
		http.Error(w, "An unexpected error occurred", http.StatusInternalServerError)
	}
}
