package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderDetailHandler shows order details with fulfillment actions
type OrderDetailHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewOrderDetailHandler creates a new order detail handler
func NewOrderDetailHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *OrderDetailHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &OrderDetailHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *OrderDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Get order with full details
	order, err := h.repo.GetOrderWithDetails(r.Context(), repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
	})
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get order items
	items, err := h.repo.GetOrderItems(r.Context(), orderUUID)
	if err != nil {
		http.Error(w, "Failed to load order items", http.StatusInternalServerError)
		return
	}

	// Get shipments
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

// UpdateOrderStatusHandler handles updating order status
type UpdateOrderStatusHandler struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewUpdateOrderStatusHandler creates a new order status update handler
func NewUpdateOrderStatusHandler(repo repository.Querier, tenantID string) *UpdateOrderStatusHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &UpdateOrderStatusHandler{
		repo:     repo,
		tenantID: tenantUUID,
	}
}

func (h *UpdateOrderStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Update order status
	err := h.repo.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   status,
	})
	if err != nil {
		http.Error(w, "Failed to update order status", http.StatusInternalServerError)
		return
	}

	// Redirect back to order detail
	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}

// CreateShipmentHandler handles creating a shipment for an order
type CreateShipmentHandler struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewCreateShipmentHandler creates a new shipment creation handler
func NewCreateShipmentHandler(repo repository.Querier, tenantID string) *CreateShipmentHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &CreateShipmentHandler{
		repo:     repo,
		tenantID: tenantUUID,
	}
}

func (h *CreateShipmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Create shipment
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
		ShippingMethodID: pgtype.UUID{Valid: false}, // Optional for now
	})
	if err != nil {
		http.Error(w, "Failed to create shipment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update order fulfillment status to fulfilled
	err = h.repo.UpdateOrderFulfillmentStatus(r.Context(), repository.UpdateOrderFulfillmentStatusParams{
		TenantID:          h.tenantID,
		ID:                orderUUID,
		FulfillmentStatus: "fulfilled",
	})
	if err != nil {
		// Log error but don't fail - shipment was created
		fmt.Printf("Warning: failed to update fulfillment status: %v\n", err)
	}

	// Update order status to shipped
	err = h.repo.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   "shipped",
	})
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to update order status: %v\n", err)
	}

	// Redirect back to order detail
	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}
