package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/dukerupert/hiri/internal/onboarding"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// OnboardingHandler handles onboarding checklist routes
type OnboardingHandler struct {
	service  *onboarding.Service
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(
	service *onboarding.Service,
	renderer *handler.Renderer,
	tenantID string,
) *OnboardingHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &OnboardingHandler{
		service:  service,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// GetStatus handles GET /admin/onboarding
// Returns the full onboarding checklist status.
// Can return JSON (for API) or HTML (for dashboard widget).
func (h *OnboardingHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	status, err := h.service.GetStatus(ctx, tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Check Accept header to decide format
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
		return
	}

	// Render HTML template
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Onboarding":  status,
	}

	if csrfToken := middleware.GetCSRFToken(ctx); csrfToken != "" {
		data["CSRFToken"] = csrfToken
	}

	h.renderer.RenderHTTP(w, "admin/onboarding", data)
}

// GetStatusJSON handles GET /admin/api/onboarding
// Always returns JSON response for API clients.
func (h *OnboardingHandler) GetStatusJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		writeJSONError(w, "Invalid tenant ID", http.StatusInternalServerError)
		return
	}

	status, err := h.service.GetStatus(ctx, tenantID)
	if err != nil {
		writeJSONError(w, "Failed to load onboarding status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// SkipItem handles POST /admin/onboarding/{item_id}/skip
// Marks an optional item as skipped.
func (h *OnboardingHandler) SkipItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID := r.PathValue("item_id")
	if itemID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Item ID required"))
		return
	}

	tenantID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get operator ID from session if available
	var operatorID *uuid.UUID
	if opID := middleware.GetOperatorID(ctx); opID != uuid.Nil {
		operatorID = &opID
	}

	if err := h.service.SkipItem(ctx, tenantID, itemID, operatorID); err != nil {
		switch err {
		case onboarding.ErrCannotSkipRequiredItem:
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Cannot skip required item"))
		case onboarding.ErrInvalidItemID:
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid item ID"))
		default:
			handler.InternalErrorResponse(w, r, err)
		}
		return
	}

	// For htmx requests, return the updated checklist widget
	if r.Header.Get("HX-Request") == "true" {
		h.GetStatus(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UnskipItem handles DELETE /admin/onboarding/{item_id}/skip
// Removes skip flag from an item.
func (h *OnboardingHandler) UnskipItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID := r.PathValue("item_id")
	if itemID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Item ID required"))
		return
	}

	tenantID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if err := h.service.UnskipItem(ctx, tenantID, itemID); err != nil {
		switch err {
		case onboarding.ErrInvalidItemID:
			handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid item ID"))
		default:
			handler.InternalErrorResponse(w, r, err)
		}
		return
	}

	// For htmx requests, return the updated checklist widget
	if r.Header.Get("HX-Request") == "true" {
		h.GetStatus(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// IsLaunchReady handles GET /admin/api/onboarding/launch-ready
// Returns a simple boolean indicating if the store is ready to launch.
func (h *OnboardingHandler) IsLaunchReady(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := uuid.FromBytes(h.tenantID.Bytes[:])
	if err != nil {
		writeJSONError(w, "Invalid tenant ID", http.StatusInternalServerError)
		return
	}

	ready, err := h.service.IsLaunchReady(ctx, tenantID)
	if err != nil {
		writeJSONError(w, "Failed to check launch readiness", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"launch_ready": ready})
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
