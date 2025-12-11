package saas

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/service"
)

// CheckoutHandler handles SaaS checkout session creation
type CheckoutHandler struct {
	onboardingService service.OnboardingService
	baseURL           string
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(
	onboardingService service.OnboardingService,
	baseURL string,
) *CheckoutHandler {
	return &CheckoutHandler{
		onboardingService: onboardingService,
		baseURL:           baseURL,
	}
}

// CheckoutResponse is the response from POST /api/saas/checkout
type CheckoutResponse struct {
	URL string `json:"url"`
}

// HandleCreateCheckoutSession handles POST /api/saas/checkout
// Creates a Stripe Checkout session for new tenant signup
// Returns JSON with the checkout URL to redirect the client
func (h *CheckoutHandler) HandleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Build success and cancel URLs
	// Success URL will receive session_id query param from Stripe
	successURL := h.baseURL + "/setup/success"
	cancelURL := h.baseURL + "/pricing"

	checkoutURL, err := h.onboardingService.CreateCheckoutSession(ctx, service.CreateCheckoutParams{
		SuccessURL: successURL,
		CancelURL:  cancelURL,
	})
	if err != nil {
		slog.Error("checkout: failed to create checkout session",
			"error", err,
		)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINTERNAL, "", "Failed to create checkout session"))
		return
	}

	slog.Info("checkout: session created",
		"redirect_url", checkoutURL,
	)

	// Return JSON response with checkout URL
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow cross-origin from marketing site
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(CheckoutResponse{URL: checkoutURL}); err != nil {
		slog.Error("checkout: failed to encode response",
			"error", err,
		)
	}
}

// HandleCheckoutOptions handles OPTIONS /api/saas/checkout for CORS preflight
func (h *CheckoutHandler) HandleCheckoutOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}
