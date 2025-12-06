package saas

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/dukerupert/freyja/internal/service"
)

// BillingHandler handles billing portal access
type BillingHandler struct {
	onboardingService service.OnboardingService
	baseURL           string
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(
	onboardingService service.OnboardingService,
	baseURL string,
) *BillingHandler {
	return &BillingHandler{
		onboardingService: onboardingService,
		baseURL:           baseURL,
	}
}

// RedirectToBillingPortal handles GET /admin/billing
// Creates Stripe Customer Portal session and redirects operator
// Requires RequireOperator middleware to populate context with operator
func (h *BillingHandler) RedirectToBillingPortal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from operator context (set by middleware)
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		slog.Error("billing: tenant_id not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	returnURL := h.baseURL + "/admin"

	portalURL, err := h.onboardingService.CreateBillingPortalSession(ctx, tenantID, returnURL)
	if err != nil {
		slog.Error("billing: failed to create portal session",
			"tenant_id", tenantID,
			"error", err,
		)
		http.Error(w, "Failed to access billing portal", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}
