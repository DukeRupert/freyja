package storefront

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/hiri/internal/cookie"
	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// _ asserts that domain packages are imported for type references
var _ = domain.ErrUserNotFound

const (
	sessionCookieName = "hiri_session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// AuthHandler handles all authentication-related flows:
// - Signup and signup success
// - Login and logout
// - Password reset (forgot and reset)
// - Email verification (verify and resend)
type AuthHandler struct {
	userService          domain.UserService
	verificationService  service.EmailVerificationService
	passwordResetService service.PasswordResetService
	repo                 repository.Querier
	renderer             *handler.Renderer
	tenantID             uuid.UUID
	cookieConfig         *cookie.Config
}

// NewAuthHandler creates a new consolidated auth handler
func NewAuthHandler(
	userService domain.UserService,
	verificationService service.EmailVerificationService,
	passwordResetService service.PasswordResetService,
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID uuid.UUID,
	cookieConfig *cookie.Config,
) *AuthHandler {
	return &AuthHandler{
		userService:          userService,
		verificationService:  verificationService,
		passwordResetService: passwordResetService,
		repo:                 repo,
		renderer:             renderer,
		tenantID:             tenantID,
		cookieConfig:         cookieConfig,
	}
}

// =============================================================================
// Signup
// =============================================================================

// ShowSignupForm handles GET /signup - displays the signup form
func (h *AuthHandler) ShowSignupForm(w http.ResponseWriter, r *http.Request) {
	h.showSignupFormWithError(w, r, nil)
}

func (h *AuthHandler) showSignupFormWithError(w http.ResponseWriter, r *http.Request, formError *string) {
	data := BaseTemplateData(r)
	if formError != nil {
		data["Error"] = *formError
	}
	h.renderer.RenderHTTP(w, "signup", data)
}

// HandleSignup handles POST /signup - processes the signup form
func (h *AuthHandler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx)

	if err := r.ParseForm(); err != nil {
		logger.Error("signup: failed to parse form", "error", err)
		errMsg := "Invalid form data"
		h.showSignupFormWithError(w, r, &errMsg)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")

	if email == "" || password == "" {
		logger.Warn("signup: missing required fields", "email", email, "hasPassword", password != "")
		errMsg := "Email and password are required"
		h.showSignupFormWithError(w, r, &errMsg)
		return
	}

	user, err := h.userService.Register(ctx, email, password, firstName, lastName)
	if err != nil {
		logger.Error("signup: registration failed", "email", email, "error", err)
		var errMsg string
		if domain.ErrorCode(err) == domain.ECONFLICT {
			errMsg = "An account with this email already exists"
		} else {
			errMsg = "Failed to create account. Please try again."
		}
		h.showSignupFormWithError(w, r, &errMsg)
		return
	}

	logger.Info("signup: user registered successfully", "email", email, "userID", user.ID)

	userID, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		logger.Error("signup: failed to convert user ID", "error", err)
		handler.InternalErrorResponse(w, r, err)
		return
	}

	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	err = h.verificationService.SendVerificationEmail(ctx, h.tenantID, userID, email, firstName, ipAddress, userAgent)
	if err != nil {
		logger.Error("signup: failed to send verification email", "error", err)
	}

	logger.Info("signup: verification email sent", "email", email)
	http.Redirect(w, r, "/signup-success?email="+email, http.StatusSeeOther)
}

// ShowSignupSuccess handles GET /signup-success - displays the verification pending page
func (h *AuthHandler) ShowSignupSuccess(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)
	email := r.URL.Query().Get("email")
	if email != "" {
		data["Email"] = email
	}
	h.renderer.RenderHTTP(w, "signup_success", data)
}

// =============================================================================
// Login
// =============================================================================

// ShowLoginForm handles GET /login - displays the login form
func (h *AuthHandler) ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	h.showLoginFormWithError(w, r, nil, "")
}

func (h *AuthHandler) showLoginFormWithError(w http.ResponseWriter, r *http.Request, formError *string, email string) {
	data := BaseTemplateData(r)
	if formError != nil {
		data["Error"] = *formError
	}
	if email != "" {
		data["Email"] = email
	}
	if r.URL.Query().Get("reset") == "success" {
		data["Success"] = "Your password has been reset successfully. Please log in with your new password."
	}
	h.renderer.RenderHTTP(w, "login", data)
}

// HandleLogin handles POST /login - processes the login form
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		errMsg := "Invalid form data"
		h.showLoginFormWithError(w, r, &errMsg, "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		errMsg := "Email and password are required"
		h.showLoginFormWithError(w, r, &errMsg, email)
		return
	}

	user, err := h.userService.Authenticate(ctx, email, password)
	if err != nil {
		errCode := domain.ErrorCode(err)
		// Handle email not verified - redirect to resend verification
		if errCode == domain.EFORBIDDEN && domain.ErrorMessage(err) == "Email has not been verified" {
			http.Redirect(w, r, "/resend-verification?email="+email, http.StatusSeeOther)
			return
		}
		var errMsg string
		switch errCode {
		case domain.EUNAUTHORIZED:
			errMsg = "Invalid email or password"
		case domain.ENOTFOUND:
			errMsg = "Invalid email or password"
		case domain.EFORBIDDEN:
			// Could be suspended or pending
			errMsg = domain.ErrorMessage(err)
		default:
			errMsg = "Login failed. Please try again."
		}
		h.showLoginFormWithError(w, r, &errMsg, email)
		return
	}

	userIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		user.ID.Bytes[0:4], user.ID.Bytes[4:6], user.ID.Bytes[6:8],
		user.ID.Bytes[8:10], user.ID.Bytes[10:16])
	token, err := h.userService.CreateSession(ctx, userIDStr)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	h.cookieConfig.SetSession(w, sessionCookieName, token, sessionMaxAge)

	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

// =============================================================================
// Logout
// =============================================================================

// HandleLogout handles POST /logout - logs out the user
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.userService.DeleteSession(ctx, cookie.Value)
	}

	h.cookieConfig.ClearSession(w, sessionCookieName)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// =============================================================================
// Forgot Password
// =============================================================================

// ShowForgotPasswordForm displays the forgot password form
func (h *AuthHandler) ShowForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)
	if r.URL.Query().Get("success") == "true" {
		data["Success"] = "If an account exists with that email, you will receive a password reset link shortly."
	}
	h.renderer.RenderHTTP(w, "forgot_password", data)
}

// HandleForgotPassword processes the forgot password form submission
func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/forgot-password", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")

	if email == "" {
		data := BaseTemplateData(r)
		data["Error"] = "Email is required"
		data["Email"] = email
		h.renderer.RenderHTTP(w, "forgot_password", data)
		return
	}

	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	// Always returns nil to prevent enumeration
	_, _ = h.passwordResetService.RequestPasswordReset(ctx, h.tenantID, email, ipAddress, userAgent)

	http.Redirect(w, r, "/forgot-password?success=true", http.StatusSeeOther)
}

// =============================================================================
// Reset Password
// =============================================================================

// ShowResetPasswordForm displays the reset password form
func (h *AuthHandler) ShowResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := BaseTemplateData(r)

	token := r.URL.Query().Get("token")
	if token == "" {
		data["Error"] = "Invalid or missing reset token"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	_, err := h.passwordResetService.ValidateResetToken(ctx, h.tenantID, token)
	if err != nil {
		data["Error"] = "This password reset link is invalid or has expired"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	data["Token"] = token
	h.renderer.RenderHTTP(w, "reset_password", data)
}

// HandleResetPassword processes the password reset form submission
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	if token == "" || newPassword == "" || confirmPassword == "" {
		data["Error"] = "All fields are required"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	if newPassword != confirmPassword {
		data["Error"] = "Passwords do not match"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	if len(newPassword) < 8 {
		data["Error"] = "Password must be at least 8 characters"
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	err := h.passwordResetService.ResetPassword(ctx, h.tenantID, token, newPassword)
	if err != nil {
		var errMsg string
		if domain.ErrorCode(err) == domain.EINVALID {
			errMsg = "This password reset link is invalid or has expired"
		} else {
			errMsg = "Failed to reset password. Please try again."
		}
		data["Error"] = errMsg
		h.renderer.RenderHTTP(w, "reset_password", data)
		return
	}

	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
}

// =============================================================================
// Email Verification
// =============================================================================

// HandleVerifyEmail handles GET /verify-email - verifies the email using a token
func (h *AuthHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx)

	token := r.URL.Query().Get("token")
	if token == "" {
		logger.Warn("verify email: missing token")
		data := BaseTemplateData(r)
		data["Error"] = "Invalid verification link. Please request a new verification email."
		h.renderer.RenderHTTP(w, "verify_email", data)
		return
	}

	err := h.verificationService.VerifyEmail(ctx, h.tenantID, token)
	if err != nil {
		logger.Warn("verify email: verification failed", "error", err)
		data := BaseTemplateData(r)
		errCode := domain.ErrorCode(err)
		switch errCode {
		case domain.EINVALID:
			data["Error"] = "This verification link is invalid or has expired. Please request a new verification email."
		case domain.ECONFLICT:
			data["Success"] = "Your email has already been verified. You can now log in."
		default:
			data["Error"] = "An error occurred while verifying your email. Please try again."
		}
		h.renderer.RenderHTTP(w, "verify_email", data)
		return
	}

	logger.Info("verify email: email verified successfully")

	data := BaseTemplateData(r)
	data["Success"] = "Your email has been verified successfully! You can now log in to your account."
	h.renderer.RenderHTTP(w, "verify_email", data)
}

// ShowResendVerificationForm handles GET /resend-verification - displays the resend form
func (h *AuthHandler) ShowResendVerificationForm(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)
	email := r.URL.Query().Get("email")
	if email != "" {
		data["Email"] = email
	}
	h.renderer.RenderHTTP(w, "resend_verification", data)
}

// HandleResendVerification handles POST /resend-verification - resends the verification email
func (h *AuthHandler) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx)

	if err := r.ParseForm(); err != nil {
		logger.Error("resend verification: failed to parse form", "error", err)
		data := BaseTemplateData(r)
		data["Error"] = "Invalid form data"
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		logger.Warn("resend verification: missing email")
		data := BaseTemplateData(r)
		data["Error"] = "Email address is required"
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	var tenantUUID pgtype.UUID
	_ = tenantUUID.Scan(h.tenantID.String())

	user, err := h.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: tenantUUID,
		Email:    email,
	})
	if err != nil {
		logger.Info("resend verification: email not found (showing success anyway)", "email", email)
		data := BaseTemplateData(r)
		data["Success"] = "If an account exists with that email, we've sent a new verification link."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	if user.EmailVerified {
		logger.Info("resend verification: email already verified", "email", email)
		data := BaseTemplateData(r)
		data["Success"] = "Your email has already been verified. You can log in."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	userID, err := pgtypeToGoogleUUID(user.ID)
	if err != nil {
		logger.Error("resend verification: failed to convert user ID", "error", err)
		data := BaseTemplateData(r)
		data["Success"] = "If an account exists with that email, we've sent a new verification link."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	err = h.verificationService.SendVerificationEmail(ctx, h.tenantID, userID, email, user.FirstName.String, ipAddress, userAgent)
	if err != nil {
		if domain.ErrorCode(err) == domain.ERATELIMIT {
			logger.Warn("resend verification: rate limit exceeded", "email", email)
			data := BaseTemplateData(r)
			data["Error"] = "Too many verification requests. Please try again in an hour."
			data["Email"] = email
			h.renderer.RenderHTTP(w, "resend_verification", data)
			return
		}
		logger.Error("resend verification: failed to send email", "error", err)
	}

	logger.Info("resend verification: verification email sent", "email", email)

	data := BaseTemplateData(r)
	data["Success"] = "If an account exists with that email, we've sent a new verification link."
	data["Email"] = email
	h.renderer.RenderHTTP(w, "resend_verification", data)
}

// pgtypeToGoogleUUID converts pgtype.UUID to google uuid.UUID
func pgtypeToGoogleUUID(pgtypeUUID pgtype.UUID) (uuid.UUID, error) {
	if !pgtypeUUID.Valid {
		return uuid.Nil, fmt.Errorf("invalid UUID")
	}
	return uuid.FromBytes(pgtypeUUID.Bytes[:])
}
