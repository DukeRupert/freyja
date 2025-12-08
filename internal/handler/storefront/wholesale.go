package storefront

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// WholesaleApplicationHandler handles the wholesale application form
type WholesaleApplicationHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewWholesaleApplicationHandler creates a new wholesale application handler
func NewWholesaleApplicationHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *WholesaleApplicationHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &WholesaleApplicationHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// businessTypes defines the available business types for wholesale applications
var businessTypes = []string{
	"Cafe",
	"Restaurant",
	"Retailer",
	"Hotel",
	"Office",
	"Other",
}

// Form handles GET /wholesale/apply - shows the application form
func (h *WholesaleApplicationHandler) Form(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/wholesale/apply", http.StatusSeeOther)
		return
	}

	// Check if user already has a pending or approved application
	if user.WholesaleApplicationStatus.Valid {
		status := user.WholesaleApplicationStatus.String
		if status == "pending" || status == "approved" {
			// Redirect to status page
			http.Redirect(w, r, "/wholesale/status", http.StatusSeeOther)
			return
		}
	}

	// Check if user is already wholesale
	if user.AccountType == "wholesale" {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}

	data := BaseTemplateData(r)
	data["BusinessTypes"] = businessTypes
	data["User"] = user

	h.renderer.RenderHTTP(w, "storefront/wholesale_apply", data)
}

// Submit handles POST /wholesale/apply - processes the application
func (h *WholesaleApplicationHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	// Check if user already has a pending application
	if user.WholesaleApplicationStatus.Valid && user.WholesaleApplicationStatus.String == "pending" {
		h.renderError(w, r, "You already have a pending application")
		return
	}

	// Check if already wholesale
	if user.AccountType == "wholesale" {
		h.renderError(w, r, "Your account is already a wholesale account")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data")
		return
	}

	// Validate required fields
	companyName := strings.TrimSpace(r.FormValue("company_name"))
	businessType := strings.TrimSpace(r.FormValue("business_type"))

	if companyName == "" {
		h.renderError(w, r, "Company name is required")
		return
	}

	if businessType == "" {
		h.renderError(w, r, "Business type is required")
		return
	}

	// Validate business type
	validBusinessType := false
	for _, bt := range businessTypes {
		if bt == businessType {
			validBusinessType = true
			break
		}
	}
	if !validBusinessType {
		h.renderError(w, r, "Invalid business type")
		return
	}

	// Build application notes from form fields
	notes := buildApplicationNotes(r)

	// Optional tax ID
	taxID := pgtype.Text{
		String: strings.TrimSpace(r.FormValue("tax_id")),
		Valid:  r.FormValue("tax_id") != "",
	}

	// Submit the application
	err := h.repo.SubmitWholesaleApplication(ctx, repository.SubmitWholesaleApplicationParams{
		ID:                          user.ID,
		CompanyName:                 pgtype.Text{String: companyName, Valid: true},
		BusinessType:                pgtype.Text{String: businessType, Valid: true},
		TaxID:                       taxID,
		WholesaleApplicationNotes:   pgtype.Text{String: notes, Valid: notes != ""},
	})
	if err != nil {
		h.renderError(w, r, "Failed to submit application. Please try again.")
		return
	}

	// Redirect to success page
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/wholesale/status")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/wholesale/status", http.StatusSeeOther)
}

// Status handles GET /wholesale/status - shows application status
func (h *WholesaleApplicationHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/wholesale/status", http.StatusSeeOther)
		return
	}

	data := BaseTemplateData(r)
	data["User"] = user

	// Determine status message
	status := "none"
	if user.WholesaleApplicationStatus.Valid {
		status = user.WholesaleApplicationStatus.String
	}
	data["ApplicationStatus"] = status

	switch status {
	case "pending":
		data["StatusTitle"] = "Application Under Review"
		data["StatusMessage"] = "Thank you for applying for a wholesale account. We're reviewing your application and will be in touch within 1-2 business days."
	case "approved":
		data["StatusTitle"] = "Application Approved"
		data["StatusMessage"] = "Congratulations! Your wholesale account has been approved. You now have access to wholesale pricing and features."
	case "rejected":
		data["StatusTitle"] = "Application Not Approved"
		data["StatusMessage"] = "Unfortunately, we were unable to approve your wholesale application at this time. Please contact us for more information."
	default:
		// No application - redirect to apply page
		http.Redirect(w, r, "/wholesale/apply", http.StatusSeeOther)
		return
	}

	h.renderer.RenderHTTP(w, "storefront/wholesale_status", data)
}

// buildApplicationNotes creates a formatted notes string from form data
func buildApplicationNotes(r *http.Request) string {
	var parts []string

	// Estimated monthly volume
	if volume := strings.TrimSpace(r.FormValue("monthly_volume")); volume != "" {
		parts = append(parts, fmt.Sprintf("Estimated Monthly Volume: %s", volume))
	}

	// Current coffee supplier
	if supplier := strings.TrimSpace(r.FormValue("current_supplier")); supplier != "" {
		parts = append(parts, fmt.Sprintf("Current Supplier: %s", supplier))
	}

	// Business website
	if website := strings.TrimSpace(r.FormValue("website")); website != "" {
		parts = append(parts, fmt.Sprintf("Website: %s", website))
	}

	// How they heard about us
	if referral := strings.TrimSpace(r.FormValue("referral_source")); referral != "" {
		parts = append(parts, fmt.Sprintf("How they found us: %s", referral))
	}

	// Additional notes
	if additional := strings.TrimSpace(r.FormValue("additional_notes")); additional != "" {
		parts = append(parts, fmt.Sprintf("Additional Notes: %s", additional))
	}

	return strings.Join(parts, "\n")
}

// renderError sends an error response
func (h *WholesaleApplicationHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Retarget", "#error-message")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`<p class="text-red-600 text-sm">%s</p>`, message)))
		return
	}
	handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "%s", message))
}
