package storefront

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/jackc/pgx/v5/pgtype"
)

// CheckoutHandler handles all checkout-related storefront routes
type CheckoutHandler struct {
	renderer             *handler.Renderer
	cartService          service.CartService
	checkoutService      service.CheckoutService
	orderService         service.OrderService
	repo                 repository.Querier
	stripePublishableKey string
	tenantID             pgtype.UUID
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(
	renderer *handler.Renderer,
	cartService service.CartService,
	checkoutService service.CheckoutService,
	orderService service.OrderService,
	repo repository.Querier,
	stripePublishableKey string,
	tenantID string,
) *CheckoutHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		slog.Error("Failed to parse tenant ID", "error", err)
	}

	return &CheckoutHandler{
		renderer:             renderer,
		cartService:          cartService,
		checkoutService:      checkoutService,
		orderService:         orderService,
		repo:                 repo,
		stripePublishableKey: stripePublishableKey,
		tenantID:             tenantUUID,
	}
}

// Page handles GET /checkout
func (h *CheckoutHandler) Page(w http.ResponseWriter, r *http.Request) {
	sessionID := GetSessionIDFromCookie(r)
	if sessionID == "" {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	cart, err := h.cartService.GetCart(r.Context(), sessionID)
	if err != nil {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	cartSummary, err := h.cartService.GetCartSummary(r.Context(), cart.ID.String())
	if err != nil {
		http.Error(w, "Failed to load cart details", http.StatusInternalServerError)
		return
	}

	if len(cartSummary.Items) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	data := BaseTemplateData(r)
	data["Cart"] = cartSummary
	data["CartID"] = cart.ID.String()
	data["StripePublishableKey"] = h.stripePublishableKey

	h.renderer.RenderHTTP(w, "storefront/checkout", data)
}

// ValidateAddress handles POST /checkout/validate-address
func (h *CheckoutHandler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShippingAddress address.Address `json:"shipping_address"`
		BillingAddress  address.Address `json:"billing_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode validate address request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("Validating address", "shipping", req.ShippingAddress, "billing", req.BillingAddress)

	shippingResult, err := h.checkoutService.ValidateAndNormalizeAddress(r.Context(), req.ShippingAddress)
	if err != nil {
		slog.Error("Shipping address validation failed", "error", err, "address", req.ShippingAddress)
		http.Error(w, fmt.Sprintf("Address validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	billingResult, err := h.checkoutService.ValidateAndNormalizeAddress(r.Context(), req.BillingAddress)
	if err != nil {
		slog.Error("Billing address validation failed", "error", err, "address", req.BillingAddress)
		http.Error(w, fmt.Sprintf("Address validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := struct {
		ShippingResult *address.ValidationResult `json:"shipping_result"`
		BillingResult  *address.ValidationResult `json:"billing_result"`
	}{
		ShippingResult: shippingResult,
		BillingResult:  billingResult,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode validate address response", "error", err)
	}
}

// GetShippingRates handles POST /checkout/shipping-rates
func (h *CheckoutHandler) GetShippingRates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID          string          `json:"cart_id"`
		ShippingAddress address.Address `json:"shipping_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode shipping rates request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("Getting shipping rates", "cart_id", req.CartID, "address", req.ShippingAddress)

	rates, err := h.checkoutService.GetShippingRates(r.Context(), req.CartID, req.ShippingAddress)
	if err != nil {
		slog.Error("Failed to get shipping rates", "error", err, "cart_id", req.CartID)
		http.Error(w, fmt.Sprintf("Failed to get shipping rates: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("Retrieved shipping rates", "count", len(rates))

	resp := struct {
		Rates []shipping.Rate `json:"rates"`
	}{
		Rates: rates,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode shipping rates response", "error", err)
	}
}

// CalculateTotal handles POST /checkout/calculate-total
func (h *CheckoutHandler) CalculateTotal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID               string          `json:"cart_id"`
		ShippingAddress      address.Address `json:"shipping_address"`
		BillingAddress       address.Address `json:"billing_address"`
		SelectedShippingRate shipping.Rate   `json:"selected_shipping_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode calculate total request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("Calculating order total", "cart_id", req.CartID)

	params := service.OrderTotalParams{
		CartID:               req.CartID,
		ShippingAddress:      req.ShippingAddress,
		BillingAddress:       req.BillingAddress,
		SelectedShippingRate: req.SelectedShippingRate,
	}

	total, err := h.checkoutService.CalculateOrderTotal(r.Context(), params)
	if err != nil {
		slog.Error("Failed to calculate order total", "error", err, "cart_id", req.CartID)
		http.Error(w, fmt.Sprintf("Failed to calculate total: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("Calculated order total", "total_cents", total.TotalCents)

	resp := struct {
		OrderTotal *service.OrderTotal `json:"order_total"`
	}{
		OrderTotal: total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode calculate total response", "error", err)
	}
}

// CreatePaymentIntent handles POST /checkout/create-payment-intent
func (h *CheckoutHandler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID          string              `json:"cart_id"`
		OrderTotal      *service.OrderTotal `json:"order_total"`
		ShippingAddress address.Address     `json:"shipping_address"`
		BillingAddress  address.Address     `json:"billing_address"`
		CustomerEmail   string              `json:"customer_email"`
		IdempotencyKey  string              `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode payment intent request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("Creating payment intent", "cart_id", req.CartID, "email", req.CustomerEmail)

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
		slog.Error("Failed to create payment intent", "error", err, "cart_id", req.CartID)
		http.Error(w, fmt.Sprintf("Failed to create payment intent: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("Payment intent created", "payment_intent_id", paymentIntent.ID, "amount_cents", paymentIntent.AmountCents)

	resp := struct {
		PaymentIntentID string `json:"payment_intent_id"`
		ClientSecret    string `json:"client_secret"`
		AmountCents     int32  `json:"amount_cents"`
	}{
		PaymentIntentID: paymentIntent.ID,
		ClientSecret:    paymentIntent.ClientSecret,
		AmountCents:     paymentIntent.AmountCents,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode payment intent response", "error", err)
	}
}

// OrderConfirmation handles GET /order-confirmation
func (h *CheckoutHandler) OrderConfirmation(w http.ResponseWriter, r *http.Request) {
	paymentIntentID := r.URL.Query().Get("payment_intent")
	redirectStatus := r.URL.Query().Get("redirect_status")

	if redirectStatus != "succeeded" {
		data := BaseTemplateData(r)
		data["PaymentIntentID"] = paymentIntentID
		data["Status"] = redirectStatus
		h.renderer.RenderHTTP(w, "storefront/order-confirmation", data)
		return
	}

	order, err := h.repo.GetOrderByPaymentIntentID(r.Context(), repository.GetOrderByPaymentIntentIDParams{
		TenantID:          h.tenantID,
		ProviderPaymentID: paymentIntentID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("Order not yet created for payment intent (webhook pending)", "payment_intent", paymentIntentID)
			data := BaseTemplateData(r)
			data["PaymentIntentID"] = paymentIntentID
			data["Status"] = "processing"
			h.renderer.RenderHTTP(w, "storefront/order-confirmation", data)
			return
		}
		slog.Error("Failed to get order", "error", err, "payment_intent", paymentIntentID)
		http.Error(w, "Failed to load order", http.StatusInternalServerError)
		return
	}

	orderDetails, err := h.repo.GetOrderWithDetails(r.Context(), repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       order.ID,
	})
	if err != nil {
		slog.Error("Failed to get order details", "error", err, "order_id", order.ID)
		http.Error(w, "Failed to load order details", http.StatusInternalServerError)
		return
	}

	orderItems, err := h.repo.GetOrderItems(r.Context(), order.ID)
	if err != nil {
		slog.Error("Failed to get order items", "error", err, "order_id", order.ID)
		http.Error(w, "Failed to load order details", http.StatusInternalServerError)
		return
	}

	sessionID := GetSessionIDFromCookie(r)
	if sessionID != "" {
		cart, err := h.cartService.GetCart(r.Context(), sessionID)
		if err == nil {
			if err := h.cartService.ClearCart(r.Context(), cart.ID.String()); err != nil {
				slog.Error("Failed to clear cart after successful payment", "error", err, "cart_id", cart.ID.String())
			} else {
				slog.Info("Cart cleared after successful payment", "cart_id", cart.ID.String(), "payment_intent", paymentIntentID)
			}
		}
	}

	type OrderItem struct {
		ProductName    string
		SKU            string
		Quantity       int32
		UnitPriceCents int32
		LineSubtotal   int32
	}

	type Address struct {
		Name       string
		Address1   string
		Address2   string
		City       string
		State      string
		PostalCode string
	}

	type OrderData struct {
		OrderNumber                  string
		Email                        string
		CreatedAt                    time.Time
		SubtotalCents                int32
		ShippingCents                int32
		TaxCents                     int32
		TotalCents                   int32
		BillingAddressSameAsShipping bool
	}

	items := make([]OrderItem, 0, len(orderItems))
	for _, item := range orderItems {
		items = append(items, OrderItem{
			ProductName:    item.ProductName,
			SKU:            item.Sku,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
			LineSubtotal:   item.Quantity * item.UnitPriceCents,
		})
	}

	billingAddressSameAsShipping := orderDetails.ShippingAddressLine1.String == orderDetails.BillingAddressLine1.String

	data := BaseTemplateData(r)
	data["Status"] = "succeeded"
	data["Order"] = OrderData{
		OrderNumber:                  orderDetails.OrderNumber,
		Email:                        orderDetails.CustomerEmail.String,
		CreatedAt:                    orderDetails.CreatedAt.Time,
		SubtotalCents:                orderDetails.SubtotalCents,
		ShippingCents:                orderDetails.ShippingCents,
		TaxCents:                     orderDetails.TaxCents,
		TotalCents:                   orderDetails.TotalCents,
		BillingAddressSameAsShipping: billingAddressSameAsShipping,
	}
	data["Items"] = items
	data["ShippingAddress"] = Address{
		Name:       orderDetails.ShippingName.String,
		Address1:   orderDetails.ShippingAddressLine1.String,
		Address2:   orderDetails.ShippingAddressLine2.String,
		City:       orderDetails.ShippingCity.String,
		State:      orderDetails.ShippingState.String,
		PostalCode: orderDetails.ShippingPostalCode.String,
	}
	data["BillingAddress"] = Address{
		Name:       orderDetails.BillingName.String,
		Address1:   orderDetails.BillingAddressLine1.String,
		Address2:   orderDetails.BillingAddressLine2.String,
		City:       orderDetails.BillingCity.String,
		State:      orderDetails.BillingState.String,
		PostalCode: orderDetails.BillingPostalCode.String,
	}

	h.renderer.RenderHTTP(w, "storefront/order-confirmation", data)
}
