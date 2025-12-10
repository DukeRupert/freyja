package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionHandler handles all subscription-related admin routes
type SubscriptionHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *SubscriptionHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// List handles GET /admin/subscriptions
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.repo.GetSubscriptionStats(ctx, h.tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	subscriptions, err := h.repo.ListSubscriptions(ctx, repository.ListSubscriptionsParams{
		TenantID: h.tenantID,
		Limit:    100,
		Offset:   0,
	})

	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"CurrentPath":   r.URL.Path,
		"Stats":         stats,
		"Subscriptions": subscriptions,
	}

	h.renderer.RenderHTTP(w, "admin/subscriptions", data)
}

// Detail handles GET /admin/subscriptions/{id}
func (h *SubscriptionHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subscriptionIDStr := r.PathValue("id")
	if subscriptionIDStr == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Subscription ID required"))
		return
	}

	var subscriptionID pgtype.UUID
	if err := subscriptionID.Scan(subscriptionIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid subscription ID"))
		return
	}

	subscription, err := h.repo.GetSubscriptionWithDetails(ctx, repository.GetSubscriptionWithDetailsParams{
		TenantID: h.tenantID,
		ID:       subscriptionID,
	})

	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	items, err := h.repo.ListSubscriptionItemsForSubscription(ctx, repository.ListSubscriptionItemsForSubscriptionParams{
		TenantID:       h.tenantID,
		SubscriptionID: subscriptionID,
	})

	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"CurrentPath":  r.URL.Path,
		"Subscription": subscription,
		"Items":        items,
		"Orders":       nil,
	}

	h.renderer.RenderHTTP(w, "admin/subscription_detail", data)
}
