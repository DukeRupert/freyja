package storefront

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/service"
)

// CartViewHandler handles the cart view page
type CartViewHandler struct {
	cartService service.CartService
	templates   *template.Template
	secure      bool // For cookie security (HTTPS)
}

// NewCartViewHandler creates a new cart view handler
func NewCartViewHandler(cartService service.CartService, templates *template.Template, secure bool) *CartViewHandler {
	return &CartViewHandler{
		cartService: cartService,
		templates:   templates,
		secure:      secure,
	}
}

// ServeHTTP handles GET /cart
func (h *CartViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		// No session = empty cart
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `
			<!DOCTYPE html>
			<html>
			<head><title>Cart - Freyja Coffee</title></head>
			<body>
				<h1>Your Cart</h1>
				<p>Your cart is empty.</p>
				<a href="/products">Continue Shopping</a>
			</body>
			</html>
		`)
		return
	}

	cart, err := h.cartService.GetCart(ctx, sessionID)
	if err != nil {
		if errors.Is(err, service.ErrCartNotFound) {
			// Session exists but no cart, show empty
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `
				<!DOCTYPE html>
				<html>
				<head><title>Cart - Freyja Coffee</title></head>
				<body>
					<h1>Your Cart</h1>
					<p>Your cart is empty.</p>
					<a href="/products">Continue Shopping</a>
				</body>
				</html>
			`)
			return
		}
		// TODO: Log error
		http.Error(w, "Failed to load cart", http.StatusInternalServerError)
		return
	}

	summary, err := h.cartService.GetCartSummary(ctx, cart.ID.String())
	if err != nil {
		// TODO: Log error
		http.Error(w, "Failed to load cart details", http.StatusInternalServerError)
		return
	}

	// TODO: Render template with cart summary
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head><title>Cart - Freyja Coffee</title></head>
		<body>
			<h1>Your Cart</h1>
			<p>%d items</p>
			<ul>
	`, summary.ItemCount)

	for _, item := range summary.Items {
		fmt.Fprintf(w, `<li>%s - %s %s (%s) x %d = $%.2f</li>`,
			item.ProductName,
			item.WeightValue,
			item.Grind,
			item.SKU,
			item.Quantity,
			float64(item.LineSubtotal)/100.0,
		)
	}

	fmt.Fprintf(w, `
			</ul>
			<p><strong>Subtotal: $%.2f</strong></p>
			<a href="/products">Continue Shopping</a>
		</body>
		</html>
	`, float64(summary.Subtotal)/100.0)
}

// AddToCartHandler handles adding items to cart
type AddToCartHandler struct {
	cartService service.CartService
	templates   *template.Template
	secure      bool
}

// NewAddToCartHandler creates a new add to cart handler
func NewAddToCartHandler(cartService service.CartService, templates *template.Template, secure bool) *AddToCartHandler {
	return &AddToCartHandler{
		cartService: cartService,
		templates:   templates,
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
	summary, err := h.cartService.AddItem(ctx, cart.ID.String(), skuID, quantity)
	if err != nil {
		if errors.Is(err, service.ErrSKUNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidQuantity) {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}
		// TODO: Log error
		http.Error(w, "Failed to add item", http.StatusInternalServerError)
		return
	}

	// TODO: Return htmx partial with updated cart summary
	// For now, return simple HTML response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div>Item added! Cart has %d items ($%.2f)</div>`,
		summary.ItemCount,
		float64(summary.Subtotal)/100.0,
	)
}

// UpdateCartItemHandler handles updating cart item quantities
type UpdateCartItemHandler struct {
	cartService service.CartService
	templates   *template.Template
}

// NewUpdateCartItemHandler creates a new update cart item handler
func NewUpdateCartItemHandler(cartService service.CartService, templates *template.Template) *UpdateCartItemHandler {
	return &UpdateCartItemHandler{
		cartService: cartService,
		templates:   templates,
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
		// TODO: Log error
		http.Error(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	// TODO: Return htmx partial with updated cart
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div>Cart updated! %d items ($%.2f)</div>`,
		summary.ItemCount,
		float64(summary.Subtotal)/100.0,
	)
}

// RemoveCartItemHandler handles removing items from cart
type RemoveCartItemHandler struct {
	cartService service.CartService
	templates   *template.Template
}

// NewRemoveCartItemHandler creates a new remove cart item handler
func NewRemoveCartItemHandler(cartService service.CartService, templates *template.Template) *RemoveCartItemHandler {
	return &RemoveCartItemHandler{
		cartService: cartService,
		templates:   templates,
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

	summary, err := h.cartService.RemoveItem(ctx, cart.ID.String(), skuID)
	if err != nil {
		// TODO: Log error
		http.Error(w, "Failed to remove item", http.StatusInternalServerError)
		return
	}

	// TODO: Return htmx partial with updated cart
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div>Item removed! %d items ($%.2f)</div>`,
		summary.ItemCount,
		float64(summary.Subtotal)/100.0,
	)
}
