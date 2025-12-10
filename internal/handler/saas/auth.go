package saas

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/service"
)

const (
	// OperatorCookieName is the cookie name for operator sessions
	OperatorCookieName = "freyja_operator"

	// OperatorCookiePath restricts cookie to admin routes
	OperatorCookiePath = "/admin"

	// OperatorSessionMaxAge is 7 days in seconds
	OperatorSessionMaxAge = 7 * 24 * 60 * 60
)

// AuthHandler handles operator authentication flows
type AuthHandler struct {
	operatorService service.OperatorService
	renderer        *handler.Renderer
	baseURL         string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	operatorService service.OperatorService,
	renderer *handler.Renderer,
	baseURL string,
) *AuthHandler {
	return &AuthHandler{
		operatorService: operatorService,
		renderer:        renderer,
		baseURL:         baseURL,
	}
}

// ShowLoginForm handles GET /login
func (h *AuthHandler) ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	h.showLoginFormWithError(w, r, nil, "")
}

func (h *AuthHandler) showLoginFormWithError(w http.ResponseWriter, r *http.Request, formError *string, email string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
	}

	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if formError != nil {
		data["Error"] = *formError
	}
	if email != "" {
		data["Email"] = email
	}

	h.renderer.RenderHTTP(w, "saas/login", data)
}

// HandleLogin handles POST /login
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showLoginFormWithError(w, r, &errMsg, "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	tenantSlug := r.FormValue("tenant_slug") // Optional: for multi-tenant login

	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showLoginFormWithError(w, r, &errMsg, email)
		return
	}

	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	// For now, we don't require tenant_id for operator login
	// since operators are unique by email across the platform
	// In a true multi-tenant setup, tenant would be determined by subdomain

	// Authenticate operator
	operator, err := h.operatorService.Authenticate(ctx, email, password)
	if err != nil {
		slog.Warn("operator: login failed",
			"email", email,
			"reason", err.Error(),
			"ip", ipAddress,
			"user_agent", userAgent,
		)

		var errMsg string
		if errors.Is(err, service.ErrOperatorInvalidPassword) || errors.Is(err, service.ErrOperatorNotFound) {
			errMsg = "Invalid email or password"
		} else if errors.Is(err, service.ErrOperatorSuspended) {
			errMsg = "Your account has been suspended"
		} else if errors.Is(err, service.ErrOperatorPending) {
			errMsg = "Please complete account setup first"
		} else {
			errMsg = "Login failed. Please try again."
		}
		h.showLoginFormWithError(w, r, &errMsg, email)
		return
	}

	// Create session
	operatorID := saasConvertUUID(operator.ID.Bytes)
	token, err := h.operatorService.CreateSession(ctx, operatorID, userAgent, ipAddress)
	if err != nil {
		slog.Error("operator: failed to create session",
			"email", email,
			"operator_id", operatorID,
			"error", err,
		)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINTERNAL, "", "Failed to create session"))
		return
	}

	slog.Info("operator: login successful",
		"email", email,
		"operator_id", operatorID,
		"ip", ipAddress,
		"user_agent", userAgent,
		"tenant_slug", tenantSlug,
	)

	// Set session cookie (scoped to /admin path)
	http.SetCookie(w, &http.Cookie{
		Name:     OperatorCookieName,
		Value:    token,
		Path:     OperatorCookiePath,
		MaxAge:   OperatorSessionMaxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to admin dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// HandleLogout handles POST /logout
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session token from cookie
	cookie, err := r.Cookie(OperatorCookieName)
	if err == nil && cookie.Value != "" {
		// Delete session
		if err := h.operatorService.DeleteSession(ctx, cookie.Value); err != nil {
			slog.Warn("operator: failed to delete session during logout",
				"error", err,
			)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     OperatorCookieName,
		Value:    "",
		Path:     OperatorCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ShowForgotPasswordForm handles GET /forgot-password
func (h *AuthHandler) ShowForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	h.showForgotPasswordFormWithMessage(w, r, nil, nil)
}

func (h *AuthHandler) showForgotPasswordFormWithMessage(w http.ResponseWriter, r *http.Request, successMsg *string, errorMsg *string) {
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

	h.renderer.RenderHTTP(w, "saas/forgot-password", data)
}

// HandleForgotPassword handles POST /forgot-password
func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showForgotPasswordFormWithMessage(w, r, nil, &errMsg)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		errMsg := "Email is required"
		h.showForgotPasswordFormWithMessage(w, r, nil, &errMsg)
		return
	}

	// Request password reset (this never reveals whether email exists)
	_, _, err := h.operatorService.RequestPasswordReset(ctx, email)
	if err != nil {
		// Log but don't reveal to user
		slog.Debug("operator: password reset request failed",
			"email", email,
			"error", err,
		)
	}

	// Always show success message to prevent email enumeration
	successMsg := "If an account exists for that email, we've sent password reset instructions."
	h.showForgotPasswordFormWithMessage(w, r, &successMsg, nil)
}

// ShowResetPasswordForm handles GET /reset-password?token=xxx
func (h *AuthHandler) ShowResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/forgot-password", http.StatusSeeOther)
		return
	}

	// Validate token
	ctx := r.Context()
	_, err := h.operatorService.ValidateResetToken(ctx, token)
	if err != nil {
		errMsg := "This password reset link is invalid or has expired. Please request a new one."
		h.showForgotPasswordFormWithMessage(w, r, nil, &errMsg)
		return
	}

	h.showResetPasswordFormWithError(w, r, token, nil)
}

func (h *AuthHandler) showResetPasswordFormWithError(w http.ResponseWriter, r *http.Request, token string, formError *string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Token":       token,
	}

	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if formError != nil {
		data["Error"] = *formError
	}

	h.renderer.RenderHTTP(w, "saas/reset-password", data)
}

// HandleResetPassword handles POST /reset-password
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showResetPasswordFormWithError(w, r, "", &errMsg)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if token == "" {
		http.Redirect(w, r, "/forgot-password", http.StatusSeeOther)
		return
	}

	if password == "" {
		errMsg := "Password is required"
		h.showResetPasswordFormWithError(w, r, token, &errMsg)
		return
	}

	if password != confirmPassword {
		errMsg := "Passwords do not match"
		h.showResetPasswordFormWithError(w, r, token, &errMsg)
		return
	}

	if len(password) < 8 {
		errMsg := "Password must be at least 8 characters"
		h.showResetPasswordFormWithError(w, r, token, &errMsg)
		return
	}

	// Reset password
	err := h.operatorService.ResetPassword(ctx, token, password)
	if err != nil {
		slog.Warn("operator: password reset failed",
			"error", err,
		)

		var errMsg string
		if errors.Is(err, service.ErrOperatorInvalidToken) {
			errMsg = "This password reset link is invalid or has expired. Please request a new one."
		} else if errors.Is(err, service.ErrWeakPassword) {
			errMsg = "Password is too weak. Please choose a stronger password."
		} else {
			errMsg = "Failed to reset password. Please try again."
		}
		h.showResetPasswordFormWithError(w, r, token, &errMsg)
		return
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
}

// Helper functions

// saasConvertUUID converts pgtype.UUID Bytes to uuid.UUID
func saasConvertUUID(bytes [16]byte) uuid.UUID {
	return uuid.UUID(bytes)
}
