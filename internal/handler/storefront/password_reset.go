package storefront

import (
	"errors"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/google/uuid"
)

// ForgotPasswordHandler handles the "forgot password" flow
// GET /forgot-password - Shows the form to request a password reset
// POST /forgot-password - Processes the reset request and sends email
type ForgotPasswordHandler struct {
	renderer             *handler.Renderer
	passwordResetService service.PasswordResetService
	tenantID             uuid.UUID
	// TODO: Add email service dependency when email abstraction is implemented
}

// NewForgotPasswordHandler creates a new forgot password handler
func NewForgotPasswordHandler(
	renderer *handler.Renderer,
	passwordResetService service.PasswordResetService,
	tenantID uuid.UUID,
) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		renderer:             renderer,
		passwordResetService: passwordResetService,
		tenantID:             tenantID,
	}
}

// ServeHTTP handles GET and POST requests for forgot password
func (h *ForgotPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.ShowForm(w, r)
		return
	}

	if r.Method == http.MethodPost {
		h.HandleSubmit(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// ShowForm displays the forgot password form
func (h *ForgotPasswordHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)

	// Check for success message from query params
	if r.URL.Query().Get("success") == "true" {
		data["Success"] = "If an account exists with that email, you will receive a password reset link shortly."
	}

	h.renderer.RenderHTTP(w, "forgot_password", data)
}

// HandleSubmit processes the forgot password form submission
func (h *ForgotPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/forgot-password", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")

	// Basic validation
	if email == "" {
		data := BaseTemplateData(r)
		data["Error"] = "Email is required"
		data["Email"] = email
		h.renderer.RenderHTTP(w, "forgot_password", data)
		return
	}

	// Get IP address and user agent
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	// Request password reset (always returns nil to prevent enumeration)
	_, _ = h.passwordResetService.RequestPasswordReset(ctx, h.tenantID, email, ipAddress, userAgent)

	// Always redirect to success page (security: don't reveal if email exists)
	http.Redirect(w, r, "/forgot-password?success=true", http.StatusSeeOther)
}

// ResetPasswordHandler handles the password reset completion flow
// GET /reset-password?token=xxx - Shows the form to enter new password
// POST /reset-password - Processes the new password
type ResetPasswordHandler struct {
	renderer             *handler.Renderer
	passwordResetService service.PasswordResetService
	userService          service.UserService
	tenantID             uuid.UUID
}

// NewResetPasswordHandler creates a new reset password handler
func NewResetPasswordHandler(
	renderer *handler.Renderer,
	passwordResetService service.PasswordResetService,
	userService service.UserService,
	tenantID uuid.UUID,
) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		renderer:             renderer,
		passwordResetService: passwordResetService,
		userService:          userService,
		tenantID:             tenantID,
	}
}

// ServeHTTP handles GET and POST requests for password reset
func (h *ResetPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.ShowForm(w, r)
		return
	}

	if r.Method == http.MethodPost {
		h.HandleSubmit(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// ShowForm displays the reset password form
func (h *ResetPasswordHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := BaseTemplateData(r)

	token := r.URL.Query().Get("token")
	if token == "" {
		data["Error"] = "Invalid or missing reset token"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	// Validate token immediately to provide early feedback
	_, err := h.passwordResetService.ValidateResetToken(ctx, h.tenantID, token)
	if err != nil {
		data["Error"] = "This password reset link is invalid or has expired"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	data["Token"] = token
	h.renderer.RenderHTTP(w, "reset_password", data)
}

// HandleSubmit processes the password reset form submission
func (h *ResetPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		data := BaseTemplateData(r)
		data["Error"] = "Invalid form data"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	token := r.FormValue("token")
	newPassword := r.FormValue("password")
	confirmPassword := r.FormValue("password_confirm")

	data := BaseTemplateData(r)
	data["Token"] = token

	// Validate required fields
	if token == "" || newPassword == "" || confirmPassword == "" {
		data["Error"] = "All fields are required"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	// Validate passwords match
	if newPassword != confirmPassword {
		data["Error"] = "Passwords do not match"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	// Validate password length (minimum 8 characters)
	if len(newPassword) < 8 {
		data["Error"] = "Password must be at least 8 characters"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	// Reset password
	err := h.passwordResetService.ResetPassword(ctx, h.tenantID, token, newPassword)
	if err != nil {
		var errMsg string
		if errors.Is(err, service.ErrInvalidToken) {
			errMsg = "This password reset link is invalid or has expired"
		} else {
			errMsg = "Failed to reset password. Please try again."
		}
		data["Error"] = errMsg
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
}
