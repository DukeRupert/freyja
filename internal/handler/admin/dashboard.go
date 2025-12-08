package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/onboarding"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DashboardHandler handles the admin dashboard page
type DashboardHandler struct {
	repo              repository.Querier
	renderer          *handler.Renderer
	tenantID          pgtype.UUID
	onboardingService *onboarding.Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string, onboardingService *onboarding.Service) *DashboardHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &DashboardHandler{
		repo:              repo,
		renderer:          renderer,
		tenantID:          tenantUUID,
		onboardingService: onboardingService,
	}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Method not allowed"))
		return
	}

	// Get order stats (last 30 days)
	thirtyDaysAgo := pgtype.Timestamptz{}
	if err := thirtyDaysAgo.Scan(time.Now().AddDate(0, 0, -30)); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	orderStats, err := h.repo.GetOrderStats(r.Context(), repository.GetOrderStatsParams{
		TenantID:  h.tenantID,
		CreatedAt: thirtyDaysAgo,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get user stats
	userStats, err := h.repo.GetUserStats(r.Context(), h.tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get recent orders
	recentOrders, err := h.repo.ListOrders(r.Context(), repository.ListOrdersParams{
		TenantID: h.tenantID,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Format recent orders for display
	type DisplayOrder struct {
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
			CustomerName:         order.CustomerName,
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

	// Get onboarding status
	tenantUUID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	onboardingStatus, err := h.onboardingService.GetStatus(r.Context(), tenantUUID)
	if err != nil {
		// Log but don't fail - dashboard should still work without onboarding
		onboardingStatus = nil
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"OrderStats": map[string]interface{}{
			"TotalOrders":         orderStats.TotalOrders,
			"PendingOrders":       orderStats.PendingOrders,
			"ProcessingOrders":    orderStats.ProcessingOrders,
			"ShippedOrders":       orderStats.ShippedOrders,
			"TotalRevenueDollars": fmt.Sprintf("%.2f", float64(revenueCents)/100),
		},
		"UserStats": map[string]interface{}{
			"TotalUsers":          userStats.TotalUsers,
			"RetailUsers":         userStats.RetailUsers,
			"WholesaleUsers":      userStats.WholesaleUsers,
			"PendingApplications": userStats.PendingApplications,
		},
		"RecentOrders": displayOrders,
		"Onboarding":   onboardingStatus,
	}

	h.renderer.RenderHTTP(w, "admin/dashboard", data)
}
