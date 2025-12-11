package saas

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// DevBypassHandler provides a development-only bypass for the Stripe checkout flow.
// This allows creating tenants directly without payment for local development.
// NEVER enable in production!
type DevBypassHandler struct {
	onboardingService service.OnboardingService
	operatorService   service.OperatorService
	repo              repository.Querier
	baseURL           string
}

// NewDevBypassHandler creates a new dev bypass handler
func NewDevBypassHandler(
	onboardingService service.OnboardingService,
	operatorService service.OperatorService,
	repo repository.Querier,
	baseURL string,
) *DevBypassHandler {
	return &DevBypassHandler{
		onboardingService: onboardingService,
		operatorService:   operatorService,
		repo:              repo,
		baseURL:           baseURL,
	}
}

// ShowDevSignupForm handles GET /dev/signup
// Shows a simple form to create a tenant without Stripe
func (h *DevBypassHandler) ShowDevSignupForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dev Signup - Hiri</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-neutral-50 min-h-screen flex items-center justify-center">
    <div class="max-w-md mx-auto px-4">
        <div class="bg-white rounded-2xl shadow-sm ring-1 ring-neutral-900/5 p-8">
            <div class="mb-6">
                <div class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800 mb-4">
                    Development Only
                </div>
                <h1 class="text-2xl font-semibold text-neutral-900">Quick Tenant Setup</h1>
                <p class="mt-2 text-sm text-neutral-600">
                    Bypass Stripe checkout and create a tenant directly for development.
                </p>
            </div>

            <form method="POST" action="/dev/signup" class="space-y-4">
                <div>
                    <label for="business_name" class="block text-sm font-medium text-neutral-700">Business Name</label>
                    <input type="text" name="business_name" id="business_name" required
                        class="mt-1 block w-full rounded-md border-neutral-300 shadow-sm focus:border-teal-500 focus:ring-teal-500 sm:text-sm px-3 py-2 border"
                        placeholder="Acme Coffee Roasters">
                </div>

                <div>
                    <label for="email" class="block text-sm font-medium text-neutral-700">Email</label>
                    <input type="email" name="email" id="email" required
                        class="mt-1 block w-full rounded-md border-neutral-300 shadow-sm focus:border-teal-500 focus:ring-teal-500 sm:text-sm px-3 py-2 border"
                        placeholder="owner@example.com">
                </div>

                <div>
                    <label for="password" class="block text-sm font-medium text-neutral-700">Password</label>
                    <input type="password" name="password" id="password" required minlength="8"
                        class="mt-1 block w-full rounded-md border-neutral-300 shadow-sm focus:border-teal-500 focus:ring-teal-500 sm:text-sm px-3 py-2 border"
                        placeholder="Min 8 characters">
                </div>

                <button type="submit"
                    class="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-teal-600 hover:bg-teal-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-teal-500">
                    Create Tenant & Login
                </button>
            </form>

            <p class="mt-6 text-xs text-neutral-500 text-center">
                This creates a tenant with status "active" and logs you in immediately.
            </p>
        </div>
    </div>
</body>
</html>`))
}

// HandleDevSignup handles POST /dev/signup
// Creates tenant, operator, sets password, and redirects to admin
func (h *DevBypassHandler) HandleDevSignup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	businessName := r.FormValue("business_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if businessName == "" || email == "" || password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if len(password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Simulate checkout completed - creates tenant and operator
	checkoutData := service.CheckoutSession{
		ID:           fmt.Sprintf("dev_bypass_%d", r.Context().Value("request_id")),
		CustomerID:   "dev_customer",
		Email:        email,
		BusinessName: businessName,
		AmountTotal:  0, // Free for dev
	}

	tenantID, operatorID, err := h.onboardingService.ProcessCheckoutCompleted(ctx, checkoutData)
	if err != nil {
		slog.Error("dev bypass: failed to create tenant",
			"error", err,
			"email", email,
		)
		http.Error(w, fmt.Sprintf("Failed to create tenant: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("dev bypass: tenant created",
		"tenant_id", tenantID,
		"operator_id", operatorID,
		"email", email,
	)

	// Activate tenant directly (bypass Stripe subscription confirmation)
	var tenantPgUUID pgtype.UUID
	_ = tenantPgUUID.Scan(tenantID.String())
	err = h.repo.SetTenantStatus(ctx, repository.SetTenantStatusParams{
		ID:     tenantPgUUID,
		Status: "active",
	})
	if err != nil {
		slog.Error("dev bypass: failed to activate tenant",
			"tenant_id", tenantID,
			"error", err,
		)
		// Continue anyway - operator can still log in
	} else {
		slog.Info("dev bypass: tenant activated", "tenant_id", tenantID)
	}

	// Set password directly (bypasses setup email flow)
	err = h.operatorService.SetPassword(ctx, operatorID, password)
	if err != nil {
		slog.Error("dev bypass: failed to set password",
			"operator_id", operatorID,
			"error", err,
		)
		http.Error(w, fmt.Sprintf("Failed to set password: %v", err), http.StatusInternalServerError)
		return
	}

	// Create session and log in
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()
	sessionToken, err := h.operatorService.CreateSession(ctx, operatorID, userAgent, ipAddress)
	if err != nil {
		slog.Error("dev bypass: failed to create session",
			"operator_id", operatorID,
			"error", err,
		)
		// Redirect to login instead - account is created
		http.Redirect(w, r, "/login?created=true", http.StatusSeeOther)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     OperatorCookieName,
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   OperatorSessionMaxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("dev bypass: signup complete, redirecting to admin",
		"tenant_id", tenantID,
		"operator_id", operatorID,
	)

	// Redirect to admin dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
