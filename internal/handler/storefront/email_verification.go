package storefront

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// VerifyEmailHandler handles the email verification page
type VerifyEmailHandler struct {
	verificationService service.EmailVerificationService
	renderer            *handler.Renderer
	tenantID            uuid.UUID
}

// NewVerifyEmailHandler creates a new email verification handler
func NewVerifyEmailHandler(
	verificationService service.EmailVerificationService,
	renderer *handler.Renderer,
	tenantID uuid.UUID,
) *VerifyEmailHandler {
	return &VerifyEmailHandler{
		verificationService: verificationService,
		renderer:            renderer,
		tenantID:            tenantID,
	}
}

// HandleVerify handles GET /verify-email - verifies the email using a token
func (h *VerifyEmailHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
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

	// Verify the token
	err := h.verificationService.VerifyEmail(ctx, h.tenantID, token)
	if err != nil {
		logger.Warn("verify email: verification failed", "error", err)
		data := BaseTemplateData(r)
		if errors.Is(err, service.ErrVerificationTokenInvalid) {
			data["Error"] = "This verification link is invalid or has expired. Please request a new verification email."
		} else if errors.Is(err, service.ErrEmailAlreadyVerified) {
			data["Success"] = "Your email has already been verified. You can now log in."
		} else {
			data["Error"] = "An error occurred while verifying your email. Please try again."
		}
		h.renderer.RenderHTTP(w, "verify_email", data)
		return
	}

	logger.Info("verify email: email verified successfully")

	// Render success page
	data := BaseTemplateData(r)
	data["Success"] = "Your email has been verified successfully! You can now log in to your account."
	h.renderer.RenderHTTP(w, "verify_email", data)
}

// ResendVerificationHandler handles resending verification emails
type ResendVerificationHandler struct {
	verificationService service.EmailVerificationService
	repo                repository.Querier
	renderer            *handler.Renderer
	tenantID            uuid.UUID
}

// NewResendVerificationHandler creates a new resend verification handler
func NewResendVerificationHandler(
	verificationService service.EmailVerificationService,
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID uuid.UUID,
) *ResendVerificationHandler {
	return &ResendVerificationHandler{
		verificationService: verificationService,
		repo:                repo,
		renderer:            renderer,
		tenantID:            tenantID,
	}
}

// ShowForm handles GET /resend-verification - displays the resend form
func (h *ResendVerificationHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData(r)

	// Check for email in query params (from signup redirect)
	email := r.URL.Query().Get("email")
	if email != "" {
		data["Email"] = email
	}

	h.renderer.RenderHTTP(w, "resend_verification", data)
}

// HandleSubmit handles POST /resend-verification - resends the verification email
func (h *ResendVerificationHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.GetLogger(ctx)

	// Parse form data
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

	// Convert tenant ID to pgtype.UUID
	var tenantUUID pgtype.UUID
	_ = tenantUUID.Scan(h.tenantID.String())

	// Try to get the user by email
	user, err := h.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: tenantUUID,
		Email:    email,
	})
	if err != nil {
		// Always show success to prevent user enumeration
		logger.Info("resend verification: email not found (showing success anyway)", "email", email)
		data := BaseTemplateData(r)
		data["Success"] = "If an account exists with that email, we've sent a new verification link."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	// Check if already verified
	if user.EmailVerified {
		logger.Info("resend verification: email already verified", "email", email)
		data := BaseTemplateData(r)
		data["Success"] = "Your email has already been verified. You can log in."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	// Get IP and user agent for rate limiting
	ipAddress := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	// Convert user ID from pgtype to uuid.UUID
	userID, err := pgtypeToGoogleUUID(user.ID)
	if err != nil {
		logger.Error("resend verification: failed to convert user ID", "error", err)
		data := BaseTemplateData(r)
		data["Success"] = "If an account exists with that email, we've sent a new verification link."
		data["Email"] = email
		h.renderer.RenderHTTP(w, "resend_verification", data)
		return
	}

	// Send verification email
	err = h.verificationService.SendVerificationEmail(ctx, h.tenantID, userID, email, user.FirstName.String, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, service.ErrVerificationRateLimitExceeded) {
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

	// Show success message
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
