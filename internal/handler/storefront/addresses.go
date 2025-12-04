package storefront

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// AddressHandler handles address management for customers
type AddressHandler struct {
	repo     repository.Querier
	renderer *handler.Renderer
	tenantID pgtype.UUID
}

// NewAddressHandler creates a new address handler
func NewAddressHandler(repo repository.Querier, renderer *handler.Renderer, tenantID string) *AddressHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &AddressHandler{
		repo:     repo,
		renderer: renderer,
		tenantID: tenantUUID,
	}
}

// AddressDisplay represents an address for template rendering
type AddressDisplay struct {
	ID                string
	FullName          string
	Company           string
	AddressLine1      string
	AddressLine2      string
	City              string
	State             string
	PostalCode        string
	Country           string
	Phone             string
	Email             string
	Label             string
	IsDefaultShipping bool
	IsDefaultBilling  bool
	FormattedAddress  string
}

// List handles GET /account/addresses
func (h *AddressHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/addresses", http.StatusSeeOther)
		return
	}

	addresses, err := h.repo.ListAddressesForUser(ctx, repository.ListAddressesForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "Failed to load addresses", http.StatusInternalServerError)
		return
	}

	displayAddresses := make([]AddressDisplay, len(addresses))
	for i, addr := range addresses {
		displayAddresses[i] = h.toDisplayAddress(addr)
	}

	data := BaseTemplateData(r)
	data["Addresses"] = displayAddresses

	h.renderer.RenderHTTP(w, "storefront/addresses", data)
}

// Create handles POST /account/addresses
func (h *AddressHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data")
		return
	}

	// Validate required fields
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	country := strings.TrimSpace(r.FormValue("country"))

	if fullName == "" || addressLine1 == "" || city == "" || state == "" || postalCode == "" {
		h.renderError(w, r, "Please fill in all required fields")
		return
	}

	if country == "" {
		country = "US"
	}

	// Create the address
	address, err := h.repo.CreateAddress(ctx, repository.CreateAddressParams{
		TenantID:     h.tenantID,
		FullName:     pgtype.Text{String: fullName, Valid: true},
		Company:      pgtype.Text{String: strings.TrimSpace(r.FormValue("company")), Valid: r.FormValue("company") != ""},
		AddressLine1: addressLine1,
		AddressLine2: pgtype.Text{String: strings.TrimSpace(r.FormValue("address_line2")), Valid: r.FormValue("address_line2") != ""},
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		Phone:        pgtype.Text{String: strings.TrimSpace(r.FormValue("phone")), Valid: r.FormValue("phone") != ""},
		Email:        pgtype.Text{String: strings.TrimSpace(r.FormValue("email")), Valid: r.FormValue("email") != ""},
		AddressType:  "customer",
	})
	if err != nil {
		h.renderError(w, r, "Failed to create address")
		return
	}

	// Check if this is the first address (make it default)
	count, _ := h.repo.CountAddressesForUser(ctx, repository.CountAddressesForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	isFirstAddress := count.AddressCount == 0

	// Set default flags
	isDefaultShipping := r.FormValue("is_default_shipping") == "on" || isFirstAddress
	isDefaultBilling := r.FormValue("is_default_billing") == "on" || isFirstAddress

	// If setting as default, clear other defaults first
	if isDefaultShipping {
		_ = h.repo.SetDefaultShippingAddress(ctx, repository.SetDefaultShippingAddressParams{
			TenantID:  h.tenantID,
			UserID:    user.ID,
			AddressID: address.ID,
		})
	}

	// Link address to user
	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = "Address"
	}

	_, err = h.repo.CreateCustomerAddress(ctx, repository.CreateCustomerAddressParams{
		TenantID:          h.tenantID,
		UserID:            user.ID,
		AddressID:         address.ID,
		IsDefaultShipping: isDefaultShipping,
		IsDefaultBilling:  isDefaultBilling,
		Label:             pgtype.Text{String: label, Valid: true},
	})
	if err != nil {
		h.renderError(w, r, "Failed to link address")
		return
	}

	// Check if htmx request
	if r.Header.Get("HX-Request") == "true" {
		// Return success response for htmx
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// Update handles POST /account/addresses/{id}
func (h *AddressHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		http.Error(w, "Invalid address ID", http.StatusBadRequest)
		return
	}

	// Verify user owns this address
	_, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "Address not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data")
		return
	}

	// Validate required fields
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	country := strings.TrimSpace(r.FormValue("country"))

	if fullName == "" || addressLine1 == "" || city == "" || state == "" || postalCode == "" {
		h.renderError(w, r, "Please fill in all required fields")
		return
	}

	if country == "" {
		country = "US"
	}

	// Update the address
	_, err = h.repo.UpdateAddress(ctx, repository.UpdateAddressParams{
		TenantID:     h.tenantID,
		ID:           addressUUID,
		FullName:     pgtype.Text{String: fullName, Valid: true},
		Company:      pgtype.Text{String: strings.TrimSpace(r.FormValue("company")), Valid: r.FormValue("company") != ""},
		AddressLine1: addressLine1,
		AddressLine2: pgtype.Text{String: strings.TrimSpace(r.FormValue("address_line2")), Valid: r.FormValue("address_line2") != ""},
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		Phone:        pgtype.Text{String: strings.TrimSpace(r.FormValue("phone")), Valid: r.FormValue("phone") != ""},
		Email:        pgtype.Text{String: strings.TrimSpace(r.FormValue("email")), Valid: r.FormValue("email") != ""},
	})
	if err != nil {
		h.renderError(w, r, "Failed to update address")
		return
	}

	// Handle default shipping update
	if r.FormValue("is_default_shipping") == "on" {
		_ = h.repo.SetDefaultShippingAddress(ctx, repository.SetDefaultShippingAddressParams{
			TenantID:  h.tenantID,
			UserID:    user.ID,
			AddressID: addressUUID,
		})
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// Delete handles POST /account/addresses/{id}/delete
func (h *AddressHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		http.Error(w, "Invalid address ID", http.StatusBadRequest)
		return
	}

	// Delete the customer-address link (address itself remains for order history)
	err := h.repo.DeleteCustomerAddress(ctx, repository.DeleteCustomerAddressParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		AddressID: addressUUID,
	})
	if err != nil {
		h.renderError(w, r, "Failed to delete address")
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// SetDefault handles POST /account/addresses/{id}/default
func (h *AddressHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		http.Error(w, "Invalid address ID", http.StatusBadRequest)
		return
	}

	// Verify user owns this address
	_, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "Address not found", http.StatusNotFound)
		return
	}

	// Set as default shipping address
	err = h.repo.SetDefaultShippingAddress(ctx, repository.SetDefaultShippingAddressParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		AddressID: addressUUID,
	})
	if err != nil {
		h.renderError(w, r, "Failed to set default address")
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// GetAddressJSON handles GET /account/addresses/{id}/json (for modal editing)
func (h *AddressHandler) GetAddressJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		http.Error(w, "Invalid address ID", http.StatusBadRequest)
		return
	}

	address, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		http.Error(w, "Address not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":            addressID,
		"full_name":     address.FullName,
		"company":       address.Company.String,
		"address_line1": address.AddressLine1,
		"address_line2": address.AddressLine2.String,
		"city":          address.City,
		"state":         address.State,
		"postal_code":   address.PostalCode,
		"country":       address.Country,
		"phone":         address.Phone.String,
		"email":         address.Email.String,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// toDisplayAddress converts a database address row to display format
func (h *AddressHandler) toDisplayAddress(addr repository.ListAddressesForUserRow) AddressDisplay {
	display := AddressDisplay{
		AddressLine1:      addr.AddressLine1,
		City:              addr.City,
		State:             addr.State,
		PostalCode:        addr.PostalCode,
		Country:           addr.Country,
		IsDefaultShipping: addr.IsDefaultShipping,
		IsDefaultBilling:  addr.IsDefaultBilling,
	}

	// Handle nullable FullName
	if addr.FullName.Valid {
		display.FullName = addr.FullName.String
	}

	// Format UUID as string
	display.ID = fmt.Sprintf("%x-%x-%x-%x-%x",
		addr.ID.Bytes[0:4], addr.ID.Bytes[4:6], addr.ID.Bytes[6:8],
		addr.ID.Bytes[8:10], addr.ID.Bytes[10:16])

	if addr.Company.Valid {
		display.Company = addr.Company.String
	}
	if addr.AddressLine2.Valid {
		display.AddressLine2 = addr.AddressLine2.String
	}
	if addr.Phone.Valid {
		display.Phone = addr.Phone.String
	}
	if addr.Email.Valid {
		display.Email = addr.Email.String
	}
	if addr.Label.Valid {
		display.Label = addr.Label.String
	} else {
		display.Label = "Address"
	}

	// Build formatted address
	parts := []string{display.AddressLine1}
	if display.AddressLine2 != "" {
		parts = append(parts, display.AddressLine2)
	}
	parts = append(parts, fmt.Sprintf("%s, %s %s", display.City, display.State, display.PostalCode))
	display.FormattedAddress = strings.Join(parts, ", ")

	return display
}

// renderError sends an error response
func (h *AddressHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Retarget", "#error-message")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`<p class="text-red-600 text-sm">%s</p>`, message)))
		return
	}
	http.Error(w, message, http.StatusBadRequest)
}
