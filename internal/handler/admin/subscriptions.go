package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionListHandler shows all subscriptions for the tenant
type SubscriptionListHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewSubscriptionListHandler creates a new subscription list handler
func NewSubscriptionListHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *SubscriptionListHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionListHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// ServeHTTP handles GET /admin/subscriptions
func (h *SubscriptionListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get subscription stats for dashboard
	stats, err := h.repo.GetSubscriptionStats(ctx, h.tenantID)
	if err != nil {
		http.Error(w, "Failed to load subscription stats", http.StatusInternalServerError)
		return
	}

	// Get all subscriptions with pagination
	subscriptions, err := h.repo.ListSubscriptions(ctx, repository.ListSubscriptionsParams{
		TenantID: h.tenantID,
		Limit:    100,
		Offset:   0,
	})

	if err != nil {
		http.Error(w, "Failed to load subscriptions", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentPath":   r.URL.Path,
		"Stats":         stats,
		"Subscriptions": subscriptions,
	}

	h.renderer.RenderHTTP(w, "admin/subscriptions", data)
}

// SubscriptionDetailHandler shows detailed subscription information
type SubscriptionDetailHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewSubscriptionDetailHandler creates a new subscription detail handler
func NewSubscriptionDetailHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *SubscriptionDetailHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &SubscriptionDetailHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// ServeHTTP handles GET /admin/subscriptions/{id}
func (h *SubscriptionDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get subscription ID from path
	subscriptionIDStr := r.PathValue("id")
	if subscriptionIDStr == "" {
		http.Error(w, "Subscription ID required", http.StatusBadRequest)
		return
	}

	var subscriptionID pgtype.UUID
	if err := subscriptionID.Scan(subscriptionIDStr); err != nil {
		http.Error(w, "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	// Get subscription with full details
	subscription, err := h.repo.GetSubscriptionWithDetails(ctx, repository.GetSubscriptionWithDetailsParams{
		TenantID: h.tenantID,
		ID:       subscriptionID,
	})

	if err != nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}

	// Get subscription items
	items, err := h.repo.ListSubscriptionItemsForSubscription(ctx, repository.ListSubscriptionItemsForSubscriptionParams{
		TenantID:       h.tenantID,
		SubscriptionID: subscriptionID,
	})

	if err != nil {
		http.Error(w, "Failed to load subscription items", http.StatusInternalServerError)
		return
	}

	// TODO: Add ListOrdersBySubscription query to show order history
	// For now, orders will be nil in the template
	// orders, err := h.repo.ListOrdersBySubscription(ctx, ...)

	data := map[string]interface{}{
		"CurrentPath":  r.URL.Path,
		"Subscription": subscription,
		"Items":        items,
		"Orders":       nil, // TODO: implement ListOrdersBySubscription query
	}

	h.renderer.RenderHTTP(w, "admin/subscription_detail", data)
}
