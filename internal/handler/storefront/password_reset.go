package storefront

import (
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/google/uuid"
)

// ForgotPasswordHandler handles the "forgot password" flow
// GET /forgot-password - Shows the form to request a password reset
// POST /forgot-password - Processes the reset request and sends email
type ForgotPasswordHandler struct {
	renderer            *handler.Renderer
	passwordResetService service.PasswordResetService
	tenantID            uuid.UUID
	// TODO: Add email service dependency when email abstraction is implemented
}

// NewForgotPasswordHandler creates a new forgot password handler
func NewForgotPasswordHandler(
	renderer *handler.Renderer,
	passwordResetService service.PasswordResetService,
	tenantID uuid.UUID,
) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{
		renderer:            renderer,
		passwordResetService: passwordResetService,
		tenantID:            tenantID,
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
	// TODO: Implement form rendering
	// 1. Create template data with BaseTemplateData(r)
	// 2. Add any error/success messages from query params
	// 3. Render "forgot_password" template
	data := BaseTemplateData(r)
	h.renderer.RenderHTTP(w, "forgot_password", data)
}

// HandleSubmit processes the forgot password form submission
func (h *ForgotPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement password reset request logic
	// 1. Parse form data (email)
	// 2. Validate email format
	// 3. Get IP address from request (r.RemoteAddr)
	// 4. Get user agent from request (r.UserAgent())
	// 5. Call passwordResetService.RequestPasswordReset()
	// 6. Handle rate limiting errors gracefully
	// 7. Send reset email with token (when email service available)
	// 8. Always show success message (even if email not found - security)
	// 9. Redirect to success page or re-render with success message

	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ResetPasswordHandler handles the password reset completion flow
// GET /reset-password?token=xxx - Shows the form to enter new password
// POST /reset-password - Processes the new password
type ResetPasswordHandler struct {
	renderer            *handler.Renderer
	passwordResetService service.PasswordResetService
	userService         service.UserService
	tenantID            uuid.UUID
}

// NewResetPasswordHandler creates a new reset password handler
func NewResetPasswordHandler(
	renderer *handler.Renderer,
	passwordResetService service.PasswordResetService,
	userService service.UserService,
	tenantID uuid.UUID,
) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		renderer:            renderer,
		passwordResetService: passwordResetService,
		userService:         userService,
		tenantID:            tenantID,
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
	// TODO: Implement form rendering
	// 1. Get token from query params
	// 2. Validate token exists
	// 3. Optionally validate token immediately (or defer to POST)
	// 4. Create template data with BaseTemplateData(r)
	// 5. Add token to template data
	// 6. Add any error messages
	// 7. Render "reset_password" template

	data := BaseTemplateData(r)
	token := r.URL.Query().Get("token")
	data["Token"] = token
	h.renderer.RenderHTTP(w, "reset_password", data)
}

// HandleSubmit processes the password reset form submission
func (h *ResetPasswordHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement password reset completion logic
	// 1. Parse form data (token, new_password, confirm_password)
	// 2. Validate passwords match
	// 3. Validate password strength (min length, etc.)
	// 4. Call passwordResetService.ResetPassword()
	// 5. Handle errors (invalid token, expired, etc.)
	// 6. On success:
	//    a. Show success message
	//    b. Optionally auto-login user (create session)
	//    c. Redirect to login or dashboard
	// 7. On error:
	//    a. Re-render form with error message

	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
