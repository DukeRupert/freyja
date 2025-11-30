package storefront

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
)

// CheckoutPageHandler displays the checkout page with cart summary
type CheckoutPageHandler struct {
	renderer    *handler.Renderer
	cartService service.CartService
}

// NewCheckoutPageHandler creates a new checkout page handler
func NewCheckoutPageHandler(renderer *handler.Renderer, cartService service.CartService) *CheckoutPageHandler {
	return &CheckoutPageHandler{
		renderer:    renderer,
		cartService: cartService,
	}
}

func (h *CheckoutPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session ID from cookie
	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		// No session, redirect to cart page
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	// Get cart by session ID
	cart, err := h.cartService.GetCart(r.Context(), sessionID)
	if err != nil {
		// No cart found, redirect to cart page
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	// Load cart summary
	cartSummary, err := h.cartService.GetCartSummary(r.Context(), cart.ID.String())
	if err != nil {
		http.Error(w, "Failed to load cart details", http.StatusInternalServerError)
		return
	}

	// Check if cart is empty
	if len(cartSummary.Items) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Cart": cartSummary,
	}

	h.renderer.RenderHTTP(w, "storefront/checkout", data)
}

// ValidateAddressHandler validates shipping and billing addresses
type ValidateAddressHandler struct {
	checkoutService service.CheckoutService
}

// NewValidateAddressHandler creates a new address validation handler
func NewValidateAddressHandler(checkoutService service.CheckoutService) *ValidateAddressHandler {
	return &ValidateAddressHandler{
		checkoutService: checkoutService,
	}
}

type ValidateAddressRequest struct {
	ShippingAddress address.Address `json:"shipping_address"`
	BillingAddress  address.Address `json:"billing_address"`
}

type ValidateAddressResponse struct {
	ShippingResult *address.ValidationResult `json:"shipping_result"`
	BillingResult  *address.ValidationResult `json:"billing_result"`
}

func (h *ValidateAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate shipping address
	shippingResult, err := h.checkoutService.ValidateAndNormalizeAddress(r.Context(), req.ShippingAddress)
	if err != nil {
		http.Error(w, fmt.Sprintf("Address validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Validate billing address
	billingResult, err := h.checkoutService.ValidateAndNormalizeAddress(r.Context(), req.BillingAddress)
	if err != nil {
		http.Error(w, fmt.Sprintf("Address validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := ValidateAddressResponse{
		ShippingResult: shippingResult,
		BillingResult:  billingResult,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetShippingRatesHandler calculates shipping rates for the cart
type GetShippingRatesHandler struct {
	checkoutService service.CheckoutService
}

// NewGetShippingRatesHandler creates a new shipping rates handler
func NewGetShippingRatesHandler(checkoutService service.CheckoutService) *GetShippingRatesHandler {
	return &GetShippingRatesHandler{
		checkoutService: checkoutService,
	}
}

type GetShippingRatesRequest struct {
	CartID          string          `json:"cart_id"`
	ShippingAddress address.Address `json:"shipping_address"`
}

type GetShippingRatesResponse struct {
	Rates []shipping.Rate `json:"rates"`
}

func (h *GetShippingRatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetShippingRatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get shipping rates
	rates, err := h.checkoutService.GetShippingRates(r.Context(), req.CartID, req.ShippingAddress)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get shipping rates: %v", err), http.StatusInternalServerError)
		return
	}

	resp := GetShippingRatesResponse{
		Rates: rates,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CalculateTotalHandler calculates the complete order total
type CalculateTotalHandler struct {
	checkoutService service.CheckoutService
}

// NewCalculateTotalHandler creates a new calculate total handler
func NewCalculateTotalHandler(checkoutService service.CheckoutService) *CalculateTotalHandler {
	return &CalculateTotalHandler{
		checkoutService: checkoutService,
	}
}

type CalculateTotalRequest struct {
	CartID               string          `json:"cart_id"`
	ShippingAddress      address.Address `json:"shipping_address"`
	BillingAddress       address.Address `json:"billing_address"`
	SelectedShippingRate shipping.Rate   `json:"selected_shipping_rate"`
}

type CalculateTotalResponse struct {
	OrderTotal *service.OrderTotal `json:"order_total"`
}

func (h *CalculateTotalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CalculateTotalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Calculate order total
	params := service.OrderTotalParams{
		CartID:               req.CartID,
		ShippingAddress:      req.ShippingAddress,
		BillingAddress:       req.BillingAddress,
		SelectedShippingRate: req.SelectedShippingRate,
	}

	total, err := h.checkoutService.CalculateOrderTotal(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate total: %v", err), http.StatusInternalServerError)
		return
	}

	resp := CalculateTotalResponse{
		OrderTotal: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CreatePaymentIntentHandler creates a Stripe payment intent
type CreatePaymentIntentHandler struct {
	checkoutService service.CheckoutService
}

// NewCreatePaymentIntentHandler creates a new payment intent handler
func NewCreatePaymentIntentHandler(checkoutService service.CheckoutService) *CreatePaymentIntentHandler {
	return &CreatePaymentIntentHandler{
		checkoutService: checkoutService,
	}
}

type CreatePaymentIntentRequest struct {
	CartID          string              `json:"cart_id"`
	OrderTotal      *service.OrderTotal `json:"order_total"`
	ShippingAddress address.Address     `json:"shipping_address"`
	BillingAddress  address.Address     `json:"billing_address"`
	CustomerEmail   string              `json:"customer_email"`
	IdempotencyKey  string              `json:"idempotency_key"`
}

type CreatePaymentIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	AmountCents     int32  `json:"amount_cents"`
}

func (h *CreatePaymentIntentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePaymentIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create payment intent
	params := service.PaymentIntentParams{
		CartID:          req.CartID,
		OrderTotal:      req.OrderTotal,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		CustomerEmail:   req.CustomerEmail,
		IdempotencyKey:  req.IdempotencyKey,
	}

	paymentIntent, err := h.checkoutService.CreatePaymentIntent(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create payment intent: %v", err), http.StatusInternalServerError)
		return
	}

	resp := CreatePaymentIntentResponse{
		PaymentIntentID: paymentIntent.ID,
		ClientSecret:    paymentIntent.ClientSecret,
		AmountCents:     paymentIntent.AmountCents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
