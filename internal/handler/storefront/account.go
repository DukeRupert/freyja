package storefront

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// AccountDashboardHandler displays the account dashboard/landing page
type AccountDashboardHandler struct {
	accountService      service.AccountService
	subscriptionService service.SubscriptionService
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewAccountDashboardHandler creates a new account dashboard handler
func NewAccountDashboardHandler(
	accountService service.AccountService,
	subscriptionService service.SubscriptionService,
	renderer *handler.Renderer,
	tenantID string,
) *AccountDashboardHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &AccountDashboardHandler{
		accountService:      accountService,
		subscriptionService: subscriptionService,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// ServeHTTP handles GET /account
func (h *AccountDashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context (RequireAuth middleware ensures this exists)
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account", http.StatusSeeOther)
		return
	}

	// Get account summary (addresses, payment methods, orders)
	accountSummary, err := h.accountService.GetAccountSummary(ctx, h.tenantID, user.ID)
	if err != nil {
		// Log error but continue with zero values
		accountSummary = service.AccountSummary{}
	}

	// Get subscription counts
	subscriptionCounts, err := h.subscriptionService.GetSubscriptionCountsForUser(ctx, h.tenantID, user.ID)
	if err != nil {
		// Log error but continue with zero values
		subscriptionCounts = service.SubscriptionCounts{}
	}

	data := BaseTemplateData(r)
	data["AccountSummary"] = accountSummary
	data["SubscriptionCounts"] = subscriptionCounts

	h.renderer.RenderHTTP(w, "storefront/account", data)
}
