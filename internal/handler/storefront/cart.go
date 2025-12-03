package storefront

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
)

// CartViewHandler handles the cart view page
type CartViewHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
	secure      bool // For cookie security (HTTPS)
}

// NewCartViewHandler creates a new cart view handler
func NewCartViewHandler(cartService service.CartService, renderer *handler.Renderer, secure bool) *CartViewHandler {
	return &CartViewHandler{
		cartService: cartService,
		renderer:    renderer,
		secure:      secure,
	}
}

// ServeHTTP handles GET /cart
func (h *CartViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// AddToCartHandler handles adding items to cart
type AddToCartHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
	secure      bool
}

// NewAddToCartHandler creates a new add to cart handler
func NewAddToCartHandler(cartService service.CartService, renderer *handler.Renderer, secure bool) *AddToCartHandler {
	return &AddToCartHandler{
		cartService: cartService,
		renderer:    renderer,
		secure:      secure,
	}
}

// ServeHTTP handles POST /cart/add
func (h *AddToCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Get or create cart
	sessionID := GetSessionIDFromCookie(r)
	cart, newSessionID, err := h.cartService.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		// TODO: Log error
		http.Error(w, "Cart error", http.StatusInternalServerError)
		return
	}

	// Set session cookie if new session was created
	if newSessionID != sessionID {
		SetSessionCookie(w, newSessionID, h.secure)
	}

	// Add item to cart
	_, err = h.cartService.AddItem(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		if errors.Is(err, service.ErrSKUNotFound) {
			tmpl, err := h.renderer.Execute("cart_error")
			if err != nil {
				http.Error(w, "Failed to load template", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data := map[string]interface{}{
				"Message": "Product not found",
			}
			if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
			}
			return
		}
		if errors.Is(err, service.ErrInvalidQuantity) {
			tmpl, err := h.renderer.Execute("cart_error")
			if err != nil {
				http.Error(w, "Failed to load template", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data := map[string]interface{}{
				"Message": "Invalid quantity",
			}
			if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
			}
			return
		}
		tmpl, err := h.renderer.Execute("cart_error")
		if err != nil {
			http.Error(w, "Failed to load template", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := map[string]interface{}{
			"Message": "Failed to add item",
		}
		if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
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

// UpdateCartItemHandler handles updating cart item quantities
type UpdateCartItemHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
}

// NewUpdateCartItemHandler creates a new update cart item handler
func NewUpdateCartItemHandler(cartService service.CartService, renderer *handler.Renderer) *UpdateCartItemHandler {
	return &UpdateCartItemHandler{
		cartService: cartService,
		renderer:    renderer,
	}
}

// ServeHTTP handles POST /cart/update
func (h *UpdateCartItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		// TODO: Log error
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	summary, err := h.cartService.UpdateItemQuantity(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		tmpl, err := h.renderer.Execute("cart_error")
		if err != nil {
			http.Error(w, "Failed to load template", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := map[string]interface{}{
			"Message": "Failed to update item",
		}
		if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
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

// RemoveCartItemHandler handles removing items from cart
type RemoveCartItemHandler struct {
	cartService service.CartService
	renderer    *handler.Renderer
}

// NewRemoveCartItemHandler creates a new remove cart item handler
func NewRemoveCartItemHandler(cartService service.CartService, renderer *handler.Renderer) *RemoveCartItemHandler {
	return &RemoveCartItemHandler{
		cartService: cartService,
		renderer:    renderer,
	}
}

// ServeHTTP handles POST /cart/remove
func (h *RemoveCartItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		// TODO: Log error
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	_, err = h.cartService.RemoveItem(ctx, cart.ID.String(), skuID)
	if err != nil {
		tmpl, err := h.renderer.Execute("cart_error")
		if err != nil {
			http.Error(w, "Failed to load template", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := map[string]interface{}{
			"Message": "Failed to remove item",
		}
		if err := tmpl.ExecuteTemplate(w, "cart_error", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// Return empty response - htmx will remove the element via swap
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
