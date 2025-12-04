package storefront

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// PaymentMethodHandler handles payment method listing and management
type PaymentMethodHandler struct {
	accountService      service.AccountService
	subscriptionService service.SubscriptionService
	repo                repository.Querier
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
}

// NewPaymentMethodHandler creates a new payment method handler
func NewPaymentMethodHandler(
	accountService service.AccountService,
	subscriptionService service.SubscriptionService,
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID string,
) *PaymentMethodHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &PaymentMethodHandler{
		accountService:      accountService,
		subscriptionService: subscriptionService,
		repo:                repo,
		renderer:            renderer,
		tenantID:            tenantUUID,
	}
}

// List handles GET /account/payment-methods
func (h *PaymentMethodHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/payment-methods", http.StatusSeeOther)
		return
	}

	// Get payment methods
	paymentMethods, err := h.accountService.ListPaymentMethods(ctx, h.tenantID, user.ID)
	if err != nil {
		http.Error(w, "Failed to load payment methods", http.StatusInternalServerError)
		return
	}

	data := BaseTemplateData(r)
	data["PaymentMethods"] = paymentMethods

	h.renderer.RenderHTTP(w, "storefront/payment_methods", data)
}

// SetDefault handles POST /account/payment-methods/{id}/default
func (h *PaymentMethodHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get payment method ID from path
	paymentMethodIDStr := r.PathValue("id")
	if paymentMethodIDStr == "" {
		http.Error(w, "Payment method ID required", http.StatusBadRequest)
		return
	}

	var paymentMethodID pgtype.UUID
	if err := paymentMethodID.Scan(paymentMethodIDStr); err != nil {
		http.Error(w, "Invalid payment method ID", http.StatusBadRequest)
		return
	}

	// Verify ownership and get billing_customer_id
	pm, err := h.repo.GetPaymentMethodByID(ctx, repository.GetPaymentMethodByIDParams{
		ID:       paymentMethodID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "Payment method not found", http.StatusNotFound)
		return
	}

	// Set as default (this updates all payment methods for the billing customer)
	err = h.repo.SetDefaultPaymentMethod(ctx, repository.SetDefaultPaymentMethodParams{
		BillingCustomerID: pm.BillingCustomerID,
		ID:                paymentMethodID,
	})
	if err != nil {
		http.Error(w, "Failed to set default payment method", http.StatusInternalServerError)
		return
	}

	// Handle htmx request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/payment-methods")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/payment-methods", http.StatusSeeOther)
}

// Portal handles GET /account/payment-methods/portal
// Redirects to Stripe Customer Portal for managing payment methods
func (h *PaymentMethodHandler) Portal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get authenticated user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/payment-methods/portal", http.StatusSeeOther)
		return
	}

	// Create portal session with return URL to payment methods page
	portalURL, err := h.subscriptionService.CreateCustomerPortalSession(ctx, service.PortalSessionParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		ReturnURL: "/account/payment-methods",
	})

	if err != nil {
		// If user doesn't have a Stripe customer, redirect back with message
		http.Redirect(w, r, "/account/payment-methods?error=no_stripe_customer", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}
