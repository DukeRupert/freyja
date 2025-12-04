package storefront

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderHistoryHandler shows order history for the authenticated user
type OrderHistoryHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewOrderHistoryHandler creates a new order history handler
func NewOrderHistoryHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *OrderHistoryHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &OrderHistoryHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// OrderSummary represents an order for display in the order history
type OrderSummary struct {
	ID                string
	OrderNumber       string
	OrderType         string
	Status            string
	FulfillmentStatus string
	TotalCents        int32
	Currency          string
	CreatedAt         time.Time
	PaymentStatus     string
	TrackingNumber    string
	Carrier           string
	ShipmentStatus    string
	ShippedAt         *time.Time
	IsSubscription    bool
	StatusColor       string
	StatusLabel       string
}

// List handles GET /account/orders
func (h *OrderHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/orders", http.StatusSeeOther)
		return
	}

	// Parse pagination params
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Parse status filter
	statusFilter := r.URL.Query().Get("status")

	limit := int32(20)
	offset := int32((page - 1) * int(limit))

	// Fetch orders for user
	orders, err := h.repo.ListOrdersForUser(ctx, repository.ListOrdersForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		http.Error(w, "Failed to load orders", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	totalCount, err := h.repo.CountOrdersForUser(ctx, repository.CountOrdersForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		totalCount = 0
	}

	// Transform orders for display
	displayOrders := make([]OrderSummary, 0, len(orders))
	for _, o := range orders {
		// Apply status filter if provided
		if statusFilter != "" && o.Status != statusFilter {
			continue
		}

		summary := OrderSummary{
			OrderNumber:       o.OrderNumber,
			OrderType:         o.OrderType,
			Status:            o.Status,
			FulfillmentStatus: o.FulfillmentStatus,
			TotalCents:        o.TotalCents,
			Currency:          o.Currency,
			CreatedAt:         o.CreatedAt.Time,
			IsSubscription:    o.SubscriptionID.Valid,
			ShipmentStatus:    o.ShipmentStatus,
		}

		// Format UUID as string
		summary.ID = fmt.Sprintf("%x-%x-%x-%x-%x",
			o.ID.Bytes[0:4], o.ID.Bytes[4:6], o.ID.Bytes[6:8],
			o.ID.Bytes[8:10], o.ID.Bytes[10:16])

		// Set payment status
		if o.PaymentStatus.Valid {
			summary.PaymentStatus = o.PaymentStatus.String
		}

		// Set shipment info
		if o.TrackingNumber.Valid {
			summary.TrackingNumber = o.TrackingNumber.String
		}
		if o.Carrier.Valid {
			summary.Carrier = o.Carrier.String
		}
		if o.ShippedAt.Valid {
			t := o.ShippedAt.Time
			summary.ShippedAt = &t
		}

		// Set status display properties
		summary.StatusLabel, summary.StatusColor = getOrderStatusDisplay(o.Status, o.FulfillmentStatus)

		displayOrders = append(displayOrders, summary)
	}

	// Calculate pagination
	totalPages := int(totalCount) / int(limit)
	if int(totalCount)%int(limit) > 0 {
		totalPages++
	}

	data := BaseTemplateData(r)
	data["Orders"] = displayOrders
	data["CurrentPage"] = page
	data["TotalPages"] = totalPages
	data["TotalCount"] = totalCount
	data["StatusFilter"] = statusFilter
	data["HasPrevPage"] = page > 1
	data["HasNextPage"] = page < totalPages
	data["PrevPage"] = page - 1
	data["NextPage"] = page + 1

	h.renderer.RenderHTTP(w, "storefront/orders", data)
}

// getOrderStatusDisplay returns the display label and color for an order status
func getOrderStatusDisplay(status, fulfillmentStatus string) (label, color string) {
	switch status {
	case "pending":
		return "Processing", "amber"
	case "paid":
		if fulfillmentStatus == "fulfilled" {
			return "Shipped", "blue"
		}
		return "Confirmed", "teal"
	case "processing":
		return "Processing", "amber"
	case "shipped":
		return "Shipped", "blue"
	case "delivered":
		return "Delivered", "green"
	case "cancelled":
		return "Cancelled", "red"
	case "refunded":
		return "Refunded", "neutral"
	default:
		return status, "neutral"
	}
}
