package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// DashboardHandler handles the admin dashboard page
type DashboardHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *DashboardHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &DashboardHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get order stats (last 30 days)
	thirtyDaysAgo := pgtype.Timestamptz{}
	if err := thirtyDaysAgo.Scan(time.Now().AddDate(0, 0, -30)); err != nil {
		http.Error(w, "Failed to calculate date range", http.StatusInternalServerError)
		return
	}

	orderStats, err := h.repo.GetOrderStats(r.Context(), repository.GetOrderStatsParams{
		TenantID:  h.tenantID,
		CreatedAt: thirtyDaysAgo,
	})
	if err != nil {
		http.Error(w, "Failed to load order stats", http.StatusInternalServerError)
		return
	}

	// Get user stats
	userStats, err := h.repo.GetUserStats(r.Context(), h.tenantID)
	if err != nil {
		http.Error(w, "Failed to load user stats", http.StatusInternalServerError)
		return
	}

	// Get recent orders
	recentOrders, err := h.repo.ListOrders(r.Context(), repository.ListOrdersParams{
		TenantID: h.tenantID,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		http.Error(w, "Failed to load recent orders", http.StatusInternalServerError)
		return
	}

	// Format recent orders for display
	type DisplayOrder struct{
		ID                   pgtype.UUID
		OrderNumber          string
		OrderType            string
		Status               string
		TotalCents           int32
		TotalDollars         string
		Currency             string
		CreatedAt            pgtype.Timestamptz
		CreatedAtFormatted   string
		CustomerEmail        pgtype.Text
		CustomerName         string
		ShippingAddressLine1 pgtype.Text
		ShippingCity         pgtype.Text
		ShippingState        pgtype.Text
	}

	displayOrders := make([]DisplayOrder, len(recentOrders))
	for i, order := range recentOrders {
		createdAtFormatted := ""
		if order.CreatedAt.Valid {
			createdAtFormatted = order.CreatedAt.Time.Format("Jan 2, 2006")
		}

		customerName := ""
		if str, ok := order.CustomerName.(string); ok {
			customerName = str
		}

		displayOrders[i] = DisplayOrder{
			ID:                   order.ID,
			OrderNumber:          order.OrderNumber,
			OrderType:            order.OrderType,
			Status:               order.Status,
			TotalCents:           order.TotalCents,
			TotalDollars:         fmt.Sprintf("%.2f", float64(order.TotalCents)/100),
			Currency:             order.Currency,
			CreatedAt:            order.CreatedAt,
			CreatedAtFormatted:   createdAtFormatted,
			CustomerEmail:        order.CustomerEmail,
			CustomerName:         customerName,
			ShippingAddressLine1: order.ShippingAddressLine1,
			ShippingCity:         order.ShippingCity,
			ShippingState:        order.ShippingState,
		}
	}

	// Convert revenue cents to dollars
	revenueCents := int64(0)
	if rc, ok := orderStats.TotalRevenueCents.(int64); ok {
		revenueCents = rc
	} else if rc, ok := orderStats.TotalRevenueCents.(int32); ok {
		revenueCents = int64(rc)
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"OrderStats": map[string]interface{}{
			"TotalOrders":          orderStats.TotalOrders,
			"PendingOrders":        orderStats.PendingOrders,
			"ProcessingOrders":     orderStats.ProcessingOrders,
			"ShippedOrders":        orderStats.ShippedOrders,
			"TotalRevenueDollars":  fmt.Sprintf("%.2f", float64(revenueCents)/100),
		},
		"UserStats": map[string]interface{}{
			"TotalUsers":           userStats.TotalUsers,
			"RetailUsers":          userStats.RetailUsers,
			"WholesaleUsers":       userStats.WholesaleUsers,
			"PendingApplications":  userStats.PendingApplications,
		},
		"RecentOrders": displayOrders,
	}

	h.renderer.RenderHTTP(w, "admin/dashboard", data)
}
