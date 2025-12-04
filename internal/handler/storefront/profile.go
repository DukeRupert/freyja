package storefront

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProfileHandler handles user profile settings
type ProfileHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
	logger   *slog.Logger
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *ProfileHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &ProfileHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
		logger:   slog.Default().With("handler", "profile"),
	}
}

// Show handles GET /account/settings
func (h *ProfileHandler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/settings", http.StatusSeeOther)
		return
	}

	data := BaseTemplateData(r)
	data["Success"] = r.URL.Query().Get("success")
	data["Error"] = r.URL.Query().Get("error")

	h.renderer.RenderHTTP(w, "storefront/settings", data)
}

// UpdateProfile handles POST /account/settings/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err, "userID", user.ID)
		h.redirectWithError(w, r, "Invalid form data")
		return
	}

	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate input lengths
	if len(firstName) > 100 {
		h.redirectWithError(w, r, "First name is too long")
		return
	}
	if len(lastName) > 100 {
		h.redirectWithError(w, r, "Last name is too long")
		return
	}
	if len(phone) > 50 {
		h.redirectWithError(w, r, "Phone number is too long")
		return
	}

	// Update profile (tenant-scoped for security)
	err := h.repo.UpdateUserProfile(ctx, repository.UpdateUserProfileParams{
		ID:        user.ID,
		TenantID:  h.tenantID,
		FirstName: pgtype.Text{String: firstName, Valid: firstName != ""},
		LastName:  pgtype.Text{String: lastName, Valid: lastName != ""},
		Phone:     pgtype.Text{String: phone, Valid: phone != ""},
	})
	if err != nil {
		h.logger.Error("failed to update profile", "error", err, "userID", user.ID)
		h.redirectWithError(w, r, "Failed to update profile")
		return
	}

	h.logger.Info("profile updated", "userID", user.ID)

	// Handle htmx request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/settings?success=profile")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/settings?success=profile", http.StatusSeeOther)
}

// ChangePassword handles POST /account/settings/password
func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err, "userID", user.ID)
		h.redirectWithError(w, r, "Invalid form data")
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate new password format FIRST (prevents timing attacks)
	if newPassword != confirmPassword {
		h.redirectWithError(w, r, "New passwords do not match")
		return
	}

	if len(newPassword) < auth.MinPasswordLength {
		h.redirectWithError(w, r, fmt.Sprintf("Password must be at least %d characters", auth.MinPasswordLength))
		return
	}

	// THEN verify current password (more expensive operation)
	if !user.PasswordHash.Valid {
		h.redirectWithError(w, r, "Password change not available for this account")
		return
	}

	if err := auth.VerifyPassword(currentPassword, user.PasswordHash.String); err != nil {
		h.logger.Warn("incorrect current password attempt", "userID", user.ID)
		h.redirectWithError(w, r, "Current password is incorrect")
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err, "userID", user.ID)
		h.redirectWithError(w, r, "Failed to update password")
		return
	}

	// Update password (tenant-scoped for security)
	err = h.repo.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           user.ID,
		TenantID:     h.tenantID,
		PasswordHash: pgtype.Text{String: newHash, Valid: true},
	})
	if err != nil {
		h.logger.Error("failed to update password", "error", err, "userID", user.ID)
		h.redirectWithError(w, r, "Failed to update password")
		return
	}

	h.logger.Info("password changed", "userID", user.ID)

	// Handle htmx request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/settings?success=password")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/settings?success=password", http.StatusSeeOther)
}

// redirectWithError redirects to settings page with an error message
func (h *ProfileHandler) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	encodedMsg := url.QueryEscape(message)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/settings?error="+encodedMsg)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/account/settings?error="+encodedMsg, http.StatusSeeOther)
}
