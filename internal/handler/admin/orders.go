package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderListHandler shows all orders for admin
type OrderListHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewOrderListHandler creates a new order list handler
func NewOrderListHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *OrderListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &OrderListHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

func (h *OrderListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all orders with pagination (always use ListOrders for consistent display)
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
