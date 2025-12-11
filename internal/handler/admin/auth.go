package admin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/dukerupert/hiri/internal/cookie"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/service"
)

const (
	// OperatorCookieName is the cookie name for operator sessions
	OperatorCookieName = "hiri_operator"

	// OperatorSessionMaxAge is 7 days in seconds
	OperatorSessionMaxAge = 7 * 24 * 60 * 60
)

// LoginHandler handles the admin login page and form submission
// Authenticates against tenant_operators table via OperatorService
type LoginHandler struct {
	operatorService service.OperatorService
	renderer        *handler.Renderer
	cookieConfig    *cookie.Config
}

// NewLoginHandler creates a new admin login handler using operators
func NewLoginHandler(operatorService service.OperatorService, renderer *handler.Renderer, cookieConfig *cookie.Config) *LoginHandler {
	return &LoginHandler{
		operatorService: operatorService,
		renderer:        renderer,
		cookieConfig:    cookieConfig,
	}
}

// ShowForm handles GET /admin/login - displays the admin login form
func (h *LoginHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	// Check for success messages from password reset
	successMsg := ""
	if r.URL.Query().Get("reset") == "success" {
		successMsg = "Password reset successfully. Please log in."
	}
	h.showFormWithError(w, r, nil, "", successMsg)
}

func (h *LoginHandler) showFormWithError(w http.ResponseWriter, r *http.Request, formError *string, email string, success string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
	}

	// Add CSRF token for form protection
	if csrfToken := middleware.GetCSRFToken(r.Context()); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	if formError != nil {
		data["Error"] = *formError
	}
	if email != "" {
		data["Email"] = email
	}
	if success != "" {
		data["Success"] = success
	}

	h.renderer.RenderHTTP(w, "admin/login", data)
}

// HandleSubmit handles POST /admin/login - processes the admin login form
func (h *LoginHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showFormWithError(w, r, &errMsg, "", "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate required fields
	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showFormWithError(w, r, &errMsg, email, "")
		return
	}

	// Get client info for audit logging
	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	// Authenticate operator
	operator, err := h.operatorService.Authenticate(ctx, email, password)
	if err != nil {
		// Audit log: failed login attempt
		slog.Warn("admin: login failed",
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
		h.showFormWithError(w, r, &errMsg, email, "")
		return
	}

	// Create session
	operatorID := uuid.UUID(operator.ID.Bytes)
	token, err := h.operatorService.CreateSession(ctx, operatorID, userAgent, ipAddress)
	if err != nil {
		slog.Error("admin: failed to create session",
			"email", email,
			"operator_id", operatorID,
			"error", err,
		)
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Audit log: successful admin login
	slog.Info("admin: login successful",
		"email", email,
		"operator_id", operatorID,
		"ip", ipAddress,
		"user_agent", userAgent,
	)

	// Set session cookie
	h.cookieConfig.SetSession(w, OperatorCookieName, token, OperatorSessionMaxAge)

	// Redirect to admin dashboard
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// LogoutHandler handles admin logout
type LogoutHandler struct {
	operatorService service.OperatorService
	cookieConfig    *cookie.Config
}

// NewLogoutHandler creates a new admin logout handler
func NewLogoutHandler(operatorService service.OperatorService, cookieConfig *cookie.Config) *LogoutHandler {
	return &LogoutHandler{
		operatorService: operatorService,
		cookieConfig:    cookieConfig,
	}
}

// HandleSubmit handles POST /admin/logout - logs out the operator
func (h *LogoutHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session cookie
	cookie, err := r.Cookie(OperatorCookieName)
	if err == nil && cookie.Value != "" {
		// Delete session from database
		if err := h.operatorService.DeleteSession(ctx, cookie.Value); err != nil {
			slog.Warn("admin: failed to delete session during logout",
				"error", err,
			)
		}
	}

	// Clear session cookie
	h.cookieConfig.ClearSession(w, OperatorCookieName)

	// Redirect to admin login page
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// ForgotPasswordHandler handles the forgot password form
type ForgotPasswordHandler struct {
	operatorService service.OperatorService
	renderer        *handler.Renderer
}

// NewForgotPasswordHandler creates a new forgot password handler
func NewForgotPasswordHandler(operatorService service.OperatorService, renderer *handler.Renderer) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		operatorService: operatorService,
		renderer:        renderer,
	}
}

// ShowForm handles GET /admin/forgot-password
func (h *ForgotPasswordHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	h.showFormWithMessage(w, r, nil, nil)
}

func (h *ForgotPasswordHandler) showFormWithMessage(w http.ResponseWriter, r *http.Request, successMsg *string, errorMsg *string) {
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

	h.renderer.RenderHTTP(w, "admin/forgot-password", data)
}

// HandleSubmit handles POST /admin/forgot-password
func (h *ForgotPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showFormWithMessage(w, r, nil, &errMsg)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		errMsg := "Email is required"
		h.showFormWithMessage(w, r, nil, &errMsg)
		return
	}

	// Request password reset (this never reveals whether email exists)
	_, _, err := h.operatorService.RequestPasswordReset(ctx, email)
	if err != nil {
		// Log but don't reveal to user
		slog.Debug("admin: password reset request failed",
			"email", email,
			"error", err,
		)
	}

	// Always show success message to prevent email enumeration
	successMsg := "If an account exists for that email, we've sent password reset instructions."
	h.showFormWithMessage(w, r, &successMsg, nil)
}

// ResetPasswordHandler handles the password reset form
type ResetPasswordHandler struct {
	operatorService service.OperatorService
	renderer        *handler.Renderer
}

// NewResetPasswordHandler creates a new reset password handler
func NewResetPasswordHandler(operatorService service.OperatorService, renderer *handler.Renderer) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		operatorService: operatorService,
		renderer:        renderer,
	}
}

// ShowForm handles GET /admin/reset-password?token=xxx
func (h *ResetPasswordHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/admin/forgot-password", http.StatusSeeOther)
		return
	}

	// Validate token
	ctx := r.Context()
	_, err := h.operatorService.ValidateResetToken(ctx, token)
	if err != nil {
		slog.Debug("admin: invalid reset token",
			"error", err,
		)
		// Redirect to forgot password with error
		http.Redirect(w, r, "/admin/forgot-password?error=invalid_token", http.StatusSeeOther)
		return
	}

	h.showFormWithError(w, r, token, nil)
}

func (h *ResetPasswordHandler) showFormWithError(w http.ResponseWriter, r *http.Request, token string, formError *string) {
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

	h.renderer.RenderHTTP(w, "admin/reset-password", data)
}

// HandleSubmit handles POST /admin/reset-password
func (h *ResetPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showFormWithError(w, r, "", &errMsg)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if token == "" {
		http.Redirect(w, r, "/admin/forgot-password", http.StatusSeeOther)
		return
	}

	if password == "" {
		errMsg := "Password is required"
		h.showFormWithError(w, r, token, &errMsg)
		return
	}

	if password != confirmPassword {
		errMsg := "Passwords do not match"
		h.showFormWithError(w, r, token, &errMsg)
		return
	}

	if len(password) < 8 {
		errMsg := "Password must be at least 8 characters"
		h.showFormWithError(w, r, token, &errMsg)
		return
	}

	// Reset password
	err := h.operatorService.ResetPassword(ctx, token, password)
	if err != nil {
		slog.Warn("admin: password reset failed",
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
		h.showFormWithError(w, r, token, &errMsg)
		return
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/admin/login?reset=success", http.StatusSeeOther)
}
