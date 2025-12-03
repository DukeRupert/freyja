package storefront

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderConfirmationHandler displays the order confirmation page
type OrderConfirmationHandler struct {
	renderer     *handler.Renderer
	cartService  service.CartService
	orderService service.OrderService
	repo         repository.Querier
	tenantID     pgtype.UUID
}

// NewOrderConfirmationHandler creates a new order confirmation handler
func NewOrderConfirmationHandler(
	renderer *handler.Renderer,
	cartService service.CartService,
	orderService service.OrderService,
	repo repository.Querier,
	tenantID string,
) *OrderConfirmationHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		slog.Error("Failed to parse tenant ID", "error", err)
	}

	return &OrderConfirmationHandler{
		renderer:     renderer,
		cartService:  cartService,
		orderService: orderService,
		repo:         repo,
		tenantID:     tenantUUID,
	}
}

func (h *OrderConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get payment intent ID from query params
	paymentIntentID := r.URL.Query().Get("payment_intent")
	redirectStatus := r.URL.Query().Get("redirect_status")

	// If payment failed, show error page
	if redirectStatus != "succeeded" {
		data := BaseTemplateData(r)
		data["PaymentIntentID"] = paymentIntentID
		data["Status"] = redirectStatus
		h.renderer.RenderHTTP(w, "storefront/order-confirmation", data)
		return
	}

	// Get order by payment intent ID
	// Note: The order is created asynchronously by the Stripe webhook handler.
	// If the order doesn't exist yet, show a pending state message.
	order, err := h.repo.GetOrderByPaymentIntentID(r.Context(), repository.GetOrderByPaymentIntentIDParams{
		TenantID:          h.tenantID,
		ProviderPaymentID: paymentIntentID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("Order not yet created for payment intent (webhook pending)", "payment_intent", paymentIntentID)
			// Show a "processing" state to the user
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

	// Get order with full details including addresses
	orderDetails, err := h.repo.GetOrderWithDetails(r.Context(), repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       order.ID,
	})
	if err != nil {
		slog.Error("Failed to get order details", "error", err, "order_id", order.ID)
		http.Error(w, "Failed to load order details", http.StatusInternalServerError)
		return
	}

	// Get order items
	orderItems, err := h.repo.GetOrderItems(r.Context(), order.ID)
	if err != nil {
		slog.Error("Failed to get order items", "error", err, "order_id", order.ID)
		http.Error(w, "Failed to load order details", http.StatusInternalServerError)
		return
	}

	// Clear the cart after successful payment
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

	// Prepare template data
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

	// Determine if billing address is same as shipping
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
