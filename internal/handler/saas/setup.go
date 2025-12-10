package saas

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/service"
)

// SetupHandler handles account setup after checkout
type SetupHandler struct {
	operatorService   service.OperatorService
	onboardingService service.OnboardingService
	renderer          *handler.Renderer
	baseURL           string
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(
	operatorService service.OperatorService,
	onboardingService service.OnboardingService,
	renderer *handler.Renderer,
	baseURL string,
) *SetupHandler {
	return &SetupHandler{
		operatorService:   operatorService,
		onboardingService: onboardingService,
		renderer:          renderer,
		baseURL:           baseURL,
	}
}

// ShowSetupForm handles GET /setup?token=xxx
func (h *SetupHandler) ShowSetupForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/resend-setup", http.StatusSeeOther)
		return
	}

	// Validate token
	ctx := r.Context()
	operator, err := h.operatorService.ValidateSetupToken(ctx, token)
	if err != nil {
		slog.Debug("setup: invalid token",
			"error", err,
		)
		errMsg := "This setup link is invalid or has expired. Please request a new one."
		h.showResendSetupFormWithMessage(w, r, nil, &errMsg)
		return
	}

	h.showSetupFormWithError(w, r, token, operator.Email, nil)
}

func (h *SetupHandler) showSetupFormWithError(w http.ResponseWriter, r *http.Request, token, email string, formError *string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Token":       token,
		"Email":       email,
	}

	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if formError != nil {
		data["Error"] = *formError
	}

	h.renderer.RenderHTTP(w, "saas/setup", data)
}

// HandleSetup handles POST /setup
func (h *SetupHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showSetupFormWithError(w, r, "", "", &errMsg)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if token == "" {
		http.Redirect(w, r, "/resend-setup", http.StatusSeeOther)
		return
	}

	// Validate token first
	operator, err := h.operatorService.ValidateSetupToken(ctx, token)
	if err != nil {
		errMsg := "This setup link is invalid or has expired. Please request a new one."
		h.showResendSetupFormWithMessage(w, r, nil, &errMsg)
		return
	}

	if password == "" {
		errMsg := "Password is required"
		h.showSetupFormWithError(w, r, token, operator.Email, &errMsg)
		return
	}

	if password != confirmPassword {
		errMsg := "Passwords do not match"
		h.showSetupFormWithError(w, r, token, operator.Email, &errMsg)
		return
	}

	if len(password) < 8 {
		errMsg := "Password must be at least 8 characters"
		h.showSetupFormWithError(w, r, token, operator.Email, &errMsg)
		return
	}

	// Set password and activate account
	operatorID := saasConvertUUID(operator.ID.Bytes)
	err = h.operatorService.SetPassword(ctx, operatorID, password)
	if err != nil {
		slog.Error("setup: failed to set password",
			"operator_id", operatorID,
			"error", err,
		)

		var errMsg string
		if errors.Is(err, service.ErrWeakPassword) {
			errMsg = "Password is too weak. Please choose a stronger password."
		} else {
			errMsg = "Failed to set password. Please try again."
		}
		h.showSetupFormWithError(w, r, token, operator.Email, &errMsg)
		return
	}

	// Create session
	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()
	sessionToken, err := h.operatorService.CreateSession(ctx, operatorID, userAgent, ipAddress)
	if err != nil {
		slog.Error("setup: failed to create session",
			"operator_id", operatorID,
			"error", err,
		)
		// Redirect to login - account is set up but session creation failed
		http.Redirect(w, r, "/login?setup=success", http.StatusSeeOther)
		return
	}

	slog.Info("setup: account setup complete",
		"operator_id", operatorID,
		"email", operator.Email,
		"ip", ipAddress,
		"user_agent", userAgent,
	)

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     OperatorCookieName,
		Value:    sessionToken,
		Path:     OperatorCookiePath,
		MaxAge:   OperatorSessionMaxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to admin dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// ShowResendSetupForm handles GET /resend-setup
func (h *SetupHandler) ShowResendSetupForm(w http.ResponseWriter, r *http.Request) {
	h.showResendSetupFormWithMessage(w, r, nil, nil)
}

func (h *SetupHandler) showResendSetupFormWithMessage(w http.ResponseWriter, r *http.Request, successMsg, errorMsg *string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
	}

	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if successMsg != nil {
		data["Success"] = *successMsg
	}
	if errorMsg != nil {
		data["Error"] = *errorMsg
	}

	h.renderer.RenderHTTP(w, "saas/resend-setup", data)
}

// HandleResendSetup handles POST /resend-setup
func (h *SetupHandler) HandleResendSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showResendSetupFormWithMessage(w, r, nil, &errMsg)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		errMsg := "Email is required"
		h.showResendSetupFormWithMessage(w, r, nil, &errMsg)
		return
	}

	// Resend setup token (this never reveals whether email exists)
	_, _, err := h.operatorService.ResendSetupToken(ctx, email)
	if err != nil {
		// Log but don't reveal to user
		slog.Debug("setup: resend token request failed",
			"email", email,
			"error", err,
		)
	}

	// Always show success message to prevent email enumeration
	successMsg := "If an account exists for that email and setup is pending, we've sent new setup instructions."
	h.showResendSetupFormWithMessage(w, r, &successMsg, nil)
}
