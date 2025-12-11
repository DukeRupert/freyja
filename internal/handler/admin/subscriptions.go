package admin

import (
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
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(repo repository.Querier, renderer *handler.Renderer) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo:     repo,
		renderer: renderer,
	}
}

// List handles GET /admin/subscriptions
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	stats, err := h.repo.GetSubscriptionStats(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	subscriptions, err := h.repo.ListSubscriptions(ctx, repository.ListSubscriptionsParams{
		TenantID: tenantID,
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
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

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
		TenantID: tenantID,
		ID:       subscriptionID,
	})

	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	items, err := h.repo.ListSubscriptionItemsForSubscription(ctx, repository.ListSubscriptionItemsForSubscriptionParams{
		TenantID:       tenantID,
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
