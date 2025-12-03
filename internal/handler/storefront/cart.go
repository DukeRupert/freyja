package storefront

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
)

// CartHandler handles all cart-related storefront routes
type CartHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
	secure      bool
}

// NewCartHandler creates a new cart handler
func NewCartHandler(cartService service.CartService, renderer *handler.Renderer, secure bool) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		renderer:    renderer,
		secure:      secure,
	}
}

// View handles GET /cart
func (h *CartHandler) View(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionID := GetSessionIDFromCookie(r)
	var summary *service.CartSummary

	if sessionID != "" {
		cart, err := h.cartService.GetCart(ctx, sessionID)
		if err != nil && !errors.Is(err, service.ErrCartNotFound) {
			http.Error(w, "Failed to load cart", http.StatusInternalServerError)
			return
		}

		if cart != nil {
			cartSummary, err := h.cartService.GetCartSummary(ctx, cart.ID.String())
			if err != nil {
				http.Error(w, "Failed to load cart details", http.StatusInternalServerError)
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
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	skuID := r.FormValue("sku_id")
	quantityStr := r.FormValue("quantity")

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	sessionID := GetSessionIDFromCookie(r)
	cart, newSessionID, err := h.cartService.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		http.Error(w, "Cart error", http.StatusInternalServerError)
		return
	}

	if newSessionID != sessionID {
		SetSessionCookie(w, newSessionID, h.secure)
	}

	_, err = h.cartService.AddItem(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		if errors.Is(err, service.ErrSKUNotFound) {
			h.renderCartError(w, "Product not found")
			return
		}
		if errors.Is(err, service.ErrInvalidQuantity) {
			h.renderCartError(w, "Invalid quantity")
			return
		}
		h.renderCartError(w, "Failed to add item")
		return
	}

	tmpl, err := h.renderer.Execute("cart_added")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "cart_added", nil); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Update handles POST /cart/update
func (h *CartHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	skuID := r.FormValue("sku_id")
	quantityStr := r.FormValue("quantity")

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		http.Error(w, "No cart found", http.StatusNotFound)
		return
	}

	cart, err := h.cartService.GetCart(ctx, sessionID)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	summary, err := h.cartService.UpdateItemQuantity(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		h.renderCartError(w, "Failed to update item")
		return
	}

	tmpl, err := h.renderer.Execute("cart_summary")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Summary": summary,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "cart_summary", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Remove handles POST /cart/remove
func (h *CartHandler) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	skuID := r.FormValue("sku_id")

	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		http.Error(w, "No cart found", http.StatusNotFound)
		return
	}

	cart, err := h.cartService.GetCart(ctx, sessionID)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	_, err = h.cartService.RemoveItem(ctx, cart.ID.String(), skuID)
	if err != nil {
		h.renderCartError(w, "Failed to remove item")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (h *CartHandler) renderCartError(w http.ResponseWriter, message string) {
	tmpl, err := h.renderer.Execute("cart_error")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]interface{}{
		"Message": message,
	}
	if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}
