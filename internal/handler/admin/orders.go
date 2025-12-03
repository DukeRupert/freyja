package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderHandler handles all order-related admin routes
type OrderHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *OrderHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &OrderHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// List handles GET /admin/orders
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	orders, err := h.repo.ListOrders(r.Context(), repository.ListOrdersParams{
		TenantID: h.tenantID,
		Limit:    100,
		Offset:   0,
	})

	if err != nil {
		http.Error(w, "Failed to load orders", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Orders":      orders,
	}

	h.renderer.RenderHTTP(w, "admin/orders", data)
}

// Detail handles GET /admin/orders/{id}
func (h *OrderHandler) Detail(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.repo.GetOrderWithDetails(r.Context(), repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
	})
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	items, err := h.repo.GetOrderItems(r.Context(), orderUUID)
	if err != nil {
		http.Error(w, "Failed to load order items", http.StatusInternalServerError)
		return
	}

	shipments, err := h.repo.GetShipmentsByOrderID(r.Context(), orderUUID)
	if err != nil {
		http.Error(w, "Failed to load shipments", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Order":       order,
		"OrderItems":  items,
		"Shipments":   shipments,
	}

	h.renderer.RenderHTTP(w, "admin/order_detail", data)
}

// UpdateStatus handles POST /admin/orders/{id}/status
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status == "" {
		http.Error(w, "Status required", http.StatusBadRequest)
		return
	}

	err := h.repo.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   status,
	})
	if err != nil {
		http.Error(w, "Failed to update order status", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}

// CreateShipment handles POST /admin/orders/{id}/shipments
func (h *OrderHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	carrier := r.FormValue("carrier")
	trackingNumber := r.FormValue("tracking_number")

	if carrier == "" || trackingNumber == "" {
		http.Error(w, "Carrier and tracking number required", http.StatusBadRequest)
		return
	}

	var carrierText pgtype.Text
	carrierText.String = carrier
	carrierText.Valid = true

	var trackingText pgtype.Text
	trackingText.String = trackingNumber
	trackingText.Valid = true

	_, err := h.repo.CreateShipment(r.Context(), repository.CreateShipmentParams{
		TenantID:         h.tenantID,
		OrderID:          orderUUID,
		Carrier:          carrierText,
		TrackingNumber:   trackingText,
		ShippingMethodID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		http.Error(w, "Failed to create shipment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.repo.UpdateOrderFulfillmentStatus(r.Context(), repository.UpdateOrderFulfillmentStatusParams{
		TenantID:          h.tenantID,
		ID:                orderUUID,
		FulfillmentStatus: "fulfilled",
	})
	if err != nil {
		fmt.Printf("Warning: failed to update fulfillment status: %v\n", err)
	}

	err = h.repo.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   "shipped",
	})
	if err != nil {
		fmt.Printf("Warning: failed to update order status: %v\n", err)
	}

	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}
