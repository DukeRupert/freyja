package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// TaxRateHandler handles all tax rate related admin routes
type TaxRateHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewTaxRateHandler creates a new tax rate handler
func NewTaxRateHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *TaxRateHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &TaxRateHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// usStateCodes returns all valid US state codes
func usStateCodes() []string {
	return []string{
		"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "FL", "GA",
		"HI", "ID", "IL", "IN", "IA", "KS", "KY", "LA", "ME", "MD",
		"MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH", "NJ",
		"NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC",
		"SD", "TN", "TX", "UT", "VT", "VA", "WA", "WV", "WI", "WY",
		"DC",
	}
}

// ListPage handles GET /admin/settings/tax-rates
func (h *TaxRateHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	taxRates, err := h.repo.ListTaxRates(ctx, h.tenantID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Build a map of configured states for easy lookup
	configuredStates := make(map[string]repository.TaxRate)
	for _, rate := range taxRates {
		configuredStates[rate.State] = rate
	}

	// Prepare display data with all states
	type DisplayTaxRate struct {
		ID          pgtype.UUID
		State       string
		Rate        string
		TaxShipping bool
		Name        string
		IsActive    bool
		IsConfigured bool
	}

	var displayRates []DisplayTaxRate
	for _, state := range usStateCodes() {
		if rate, exists := configuredStates[state]; exists {
			// Convert decimal rate to percentage string
			rateFloat, _ := rate.Rate.Float64Value()
			ratePercent := rateFloat.Float64 * 100

			displayRates = append(displayRates, DisplayTaxRate{
				ID:           rate.ID,
				State:        rate.State,
				Rate:         fmt.Sprintf("%.2f", ratePercent),
				TaxShipping:  rate.TaxShipping,
				Name:         rate.Name.String,
				IsActive:     rate.IsActive,
				IsConfigured: true,
			})
		} else {
			displayRates = append(displayRates, DisplayTaxRate{
				State:        state,
				IsConfigured: false,
			})
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"TaxRates":    displayRates,
		"StateCodes":  usStateCodes(),
	}

	h.renderer.RenderHTTP(w, "admin/tax_rates", data)
}

// Create handles POST /admin/settings/tax-rates
func (h *TaxRateHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	state := strings.TrimSpace(strings.ToUpper(r.FormValue("state")))
	rateStr := strings.TrimSpace(r.FormValue("rate"))
	taxShipping := r.FormValue("tax_shipping") == "on"
	name := strings.TrimSpace(r.FormValue("name"))
	isActive := r.FormValue("is_active") == "on"

	if state == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "State is required"))
		return
	}

	if rateStr == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Rate is required"))
		return
	}

	// Parse rate as percentage and convert to decimal (divide by 100)
	ratePercent, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || ratePercent < 0 || ratePercent > 100 {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid rate (must be between 0 and 100)"))
		return
	}

	// Convert percentage to decimal rate
	rateDecimal := decimal.NewFromFloat(ratePercent / 100)
	var rateNumeric pgtype.Numeric
	if err := rateNumeric.Scan(rateDecimal.String()); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	_, err = h.repo.CreateTaxRate(ctx, repository.CreateTaxRateParams{
		TenantID:    h.tenantID,
		State:       state,
		Rate:        rateNumeric,
		TaxShipping: taxShipping,
		Name:        pgtype.Text{String: name, Valid: name != ""},
		IsActive:    isActive,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/settings/tax-rates", http.StatusSeeOther)
}

// Update handles PUT /admin/settings/tax-rates/{id}
func (h *TaxRateHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	taxRateID := r.PathValue("id")
	if taxRateID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Tax Rate ID required"))
		return
	}

	var taxRateUUID pgtype.UUID
	if err := taxRateUUID.Scan(taxRateID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid tax rate ID"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	rateStr := strings.TrimSpace(r.FormValue("rate"))
	taxShipping := r.FormValue("tax_shipping") == "on"
	name := strings.TrimSpace(r.FormValue("name"))
	isActive := r.FormValue("is_active") == "on"

	if rateStr == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Rate is required"))
		return
	}

	// Parse rate as percentage and convert to decimal
	ratePercent, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || ratePercent < 0 || ratePercent > 100 {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid rate (must be between 0 and 100)"))
		return
	}

	rateDecimal := decimal.NewFromFloat(ratePercent / 100)
	var rateNumeric pgtype.Numeric
	if err := rateNumeric.Scan(rateDecimal.String()); err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	_, err = h.repo.UpdateTaxRate(ctx, repository.UpdateTaxRateParams{
		TenantID:    h.tenantID,
		ID:          taxRateUUID,
		Rate:        rateNumeric,
		TaxShipping: taxShipping,
		Name:        pgtype.Text{String: name, Valid: name != ""},
		IsActive:    isActive,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Tax rate updated"))
		return
	}

	http.Redirect(w, r, "/admin/settings/tax-rates", http.StatusSeeOther)
}

// Delete handles DELETE /admin/settings/tax-rates/{id}
func (h *TaxRateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	taxRateID := r.PathValue("id")
	if taxRateID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Tax Rate ID required"))
		return
	}

	var taxRateUUID pgtype.UUID
	if err := taxRateUUID.Scan(taxRateID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid tax rate ID"))
		return
	}

	err := h.repo.DeleteTaxRate(ctx, repository.DeleteTaxRateParams{
		TenantID: h.tenantID,
		ID:       taxRateUUID,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/settings/tax-rates", http.StatusSeeOther)
}
