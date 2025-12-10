package admin

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/jobs"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/google/uuid"
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
		handler.InternalErrorResponse(w, r, err)
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Order ID required"))
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid order ID"))
		return
	}

	order, err := h.repo.GetOrderWithDetails(r.Context(), repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	items, err := h.repo.GetOrderItems(r.Context(), orderUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	shipments, err := h.repo.GetShipmentsByOrderID(r.Context(), orderUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
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
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Order ID required"))
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid order ID"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	status := r.FormValue("status")
	if status == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Status required"))
		return
	}

	err := h.repo.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   status,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}

// CreateShipment handles POST /admin/orders/{id}/shipments
func (h *OrderHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx, slog.Default())

	orderID := r.PathValue("id")
	if orderID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Order ID required"))
		return
	}

	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid order ID"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	carrier := r.FormValue("carrier")
	trackingNumber := r.FormValue("tracking_number")

	if carrier == "" || trackingNumber == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Carrier and tracking number required"))
		return
	}

	// Get order details for the shipping confirmation email
	order, err := h.repo.GetOrderWithDetails(ctx, repository.GetOrderWithDetailsParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
	})
	if err != nil {
		logger.Error("failed to get order details for shipment", "error", err, "order_id", orderID)
		handler.NotFoundResponse(w, r)
		return
	}

	var carrierText pgtype.Text
	carrierText.String = carrier
	carrierText.Valid = true

	var trackingText pgtype.Text
	trackingText.String = trackingNumber
	trackingText.Valid = true

	_, err = h.repo.CreateShipment(ctx, repository.CreateShipmentParams{
		TenantID:         h.tenantID,
		OrderID:          orderUUID,
		Carrier:          carrierText,
		TrackingNumber:   trackingText,
		ShippingMethodID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	err = h.repo.UpdateOrderFulfillmentStatus(ctx, repository.UpdateOrderFulfillmentStatusParams{
		TenantID:          h.tenantID,
		ID:                orderUUID,
		FulfillmentStatus: "fulfilled",
	})
	if err != nil {
		logger.Warn("failed to update fulfillment status", "error", err, "order_id", orderID)
	}

	err = h.repo.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
		TenantID: h.tenantID,
		ID:       orderUUID,
		Status:   "shipped",
	})
	if err != nil {
		logger.Warn("failed to update order status", "error", err, "order_id", orderID)
	}

	// Enqueue shipping confirmation email
	if order.CustomerEmail.Valid && order.CustomerEmail.String != "" {
		tenantUUID, err := uuid.FromBytes(h.tenantID.Bytes[:])
		if err != nil {
			logger.Error("failed to convert tenant ID", "error", err)
		} else {
			orderUUIDGoogle, err := uuid.FromBytes(orderUUID.Bytes[:])
			if err != nil {
				logger.Error("failed to convert order ID", "error", err)
			} else {
				customerName := order.CustomerFirstName.String
				if order.CustomerLastName.Valid && order.CustomerLastName.String != "" {
					customerName += " " + order.CustomerLastName.String
				}

				// Build tracking URL based on carrier
				trackingURL := buildTrackingURL(carrier, trackingNumber)

				payload := jobs.ShippingConfirmationPayload{
					OrderID:        orderUUIDGoogle,
					Email:          order.CustomerEmail.String,
					CustomerName:   customerName,
					OrderNumber:    order.OrderNumber,
					Carrier:        carrier,
					TrackingNumber: trackingNumber,
					TrackingURL:    trackingURL,
				}

				if err := jobs.EnqueueShippingConfirmationEmail(ctx, h.repo, tenantUUID, payload); err != nil {
					logger.Error("failed to enqueue shipping confirmation email", "error", err, "order_id", orderID)
				} else {
					logger.Info("shipping confirmation email enqueued", "order_id", orderID, "email", order.CustomerEmail.String)
				}
			}
		}
	}

	http.Redirect(w, r, "/admin/orders/"+orderID, http.StatusSeeOther)
}

// buildTrackingURL constructs a tracking URL for common carriers
func buildTrackingURL(carrier, trackingNumber string) string {
	switch carrier {
	case "USPS", "usps":
		return "https://tools.usps.com/go/TrackConfirmAction?tLabels=" + trackingNumber
	case "UPS", "ups":
		return "https://www.ups.com/track?tracknum=" + trackingNumber
	case "FedEx", "fedex", "FEDEX":
		return "https://www.fedex.com/fedextrack/?trknbr=" + trackingNumber
	case "DHL", "dhl":
		return "https://www.dhl.com/en/express/tracking.html?AWB=" + trackingNumber
	default:
		// Return empty string if carrier not recognized
		return ""
	}
}
