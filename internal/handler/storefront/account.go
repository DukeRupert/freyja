package storefront

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// AccountHandler handles all account-related operations:
// - Dashboard
// - Order history
// - Address management
// - Payment methods
// - Profile settings
type AccountHandler struct {
	accountService      service.AccountService
	subscriptionService domain.SubscriptionService
	repo                repository.Querier
	renderer            *handler.Renderer
	tenantID            pgtype.UUID
	logger              *slog.Logger
}

// NewAccountHandler creates a new consolidated account handler
func NewAccountHandler(
	accountService service.AccountService,
	subscriptionService domain.SubscriptionService,
	repo repository.Querier,
	renderer *handler.Renderer,
	tenantID string,
) *AccountHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &AccountHandler{
		accountService:      accountService,
		subscriptionService: subscriptionService,
		repo:                repo,
		renderer:            renderer,
		tenantID:            tenantUUID,
		logger:              slog.Default().With("handler", "account"),
	}
}

// =============================================================================
// Dashboard
// =============================================================================

// Dashboard handles GET /account - shows account dashboard
func (h *AccountHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account", http.StatusSeeOther)
		return
	}

	// Get account summary (addresses, payment methods, orders)
	accountSummary, err := h.accountService.GetAccountSummary(ctx, h.tenantID, user.ID)
	if err != nil {
		accountSummary = service.AccountSummary{}
	}

	// Get subscription counts
	subscriptionCounts, err := h.subscriptionService.GetSubscriptionCountsForUser(ctx, h.tenantID, user.ID)
	if err != nil {
		subscriptionCounts = domain.SubscriptionCounts{}
	}

	data := BaseTemplateData(r)
	data["AccountSummary"] = accountSummary
	data["SubscriptionCounts"] = subscriptionCounts

	h.renderer.RenderHTTP(w, "storefront/account", data)
}

// =============================================================================
// Order History
// =============================================================================

// OrderSummary represents an order for display in the order history
type OrderSummary struct {
	ID                string
	OrderNumber       string
	OrderType         string
	Status            string
	FulfillmentStatus string
	TotalCents        int32
	Currency          string
	CreatedAt         time.Time
	PaymentStatus     string
	TrackingNumber    string
	Carrier           string
	ShipmentStatus    string
	ShippedAt         *time.Time
	IsSubscription    bool
	StatusColor       string
	StatusLabel       string
}

// OrderList handles GET /account/orders - shows order history
func (h *AccountHandler) OrderList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/orders", http.StatusSeeOther)
		return
	}

	// Parse pagination params
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Parse status filter
	statusFilter := r.URL.Query().Get("status")

	limit := int32(20)
	offset := int32((page - 1) * int(limit))

	// Fetch orders for user
	orders, err := h.repo.ListOrdersForUser(ctx, repository.ListOrdersForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Get total count for pagination
	totalCount, err := h.repo.CountOrdersForUser(ctx, repository.CountOrdersForUserParams{
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		totalCount = 0
	}

	// Transform orders for display
	displayOrders := make([]OrderSummary, 0, len(orders))
	for _, o := range orders {
		// Apply status filter if provided
		if statusFilter != "" && o.Status != statusFilter {
			continue
		}

		summary := OrderSummary{
			OrderNumber:       o.OrderNumber,
			OrderType:         o.OrderType,
			Status:            o.Status,
			FulfillmentStatus: o.FulfillmentStatus,
			TotalCents:        o.TotalCents,
			Currency:          o.Currency,
			CreatedAt:         o.CreatedAt.Time,
			IsSubscription:    o.SubscriptionID.Valid,
			ShipmentStatus:    o.ShipmentStatus,
		}

		// Format UUID as string
		summary.ID = fmt.Sprintf("%x-%x-%x-%x-%x",
			o.ID.Bytes[0:4], o.ID.Bytes[4:6], o.ID.Bytes[6:8],
			o.ID.Bytes[8:10], o.ID.Bytes[10:16])

		// Set payment status
		if o.PaymentStatus.Valid {
			summary.PaymentStatus = o.PaymentStatus.String
		}

		// Set shipment info
		if o.TrackingNumber.Valid {
			summary.TrackingNumber = o.TrackingNumber.String
		}
		if o.Carrier.Valid {
			summary.Carrier = o.Carrier.String
		}
		if o.ShippedAt.Valid {
			t := o.ShippedAt.Time
			summary.ShippedAt = &t
		}

		// Set status display properties
		summary.StatusLabel, summary.StatusColor = getOrderStatusDisplay(o.Status, o.FulfillmentStatus)

		displayOrders = append(displayOrders, summary)
	}

	// Calculate pagination
	totalPages := int(totalCount) / int(limit)
	if int(totalCount)%int(limit) > 0 {
		totalPages++
	}

	data := BaseTemplateData(r)
	data["Orders"] = displayOrders
	data["CurrentPage"] = page
	data["TotalPages"] = totalPages
	data["TotalCount"] = totalCount
	data["StatusFilter"] = statusFilter
	data["HasPrevPage"] = page > 1
	data["HasNextPage"] = page < totalPages
	data["PrevPage"] = page - 1
	data["NextPage"] = page + 1

	h.renderer.RenderHTTP(w, "storefront/orders", data)
}

// getOrderStatusDisplay returns the display label and color for an order status
func getOrderStatusDisplay(status, fulfillmentStatus string) (label, color string) {
	switch status {
	case "pending":
		return "Processing", "amber"
	case "paid":
		if fulfillmentStatus == "fulfilled" {
			return "Shipped", "blue"
		}
		return "Confirmed", "teal"
	case "processing":
		return "Processing", "amber"
	case "shipped":
		return "Shipped", "blue"
	case "delivered":
		return "Delivered", "green"
	case "cancelled":
		return "Cancelled", "red"
	case "refunded":
		return "Refunded", "neutral"
	default:
		return status, "neutral"
	}
}

// =============================================================================
// Address Management
// =============================================================================

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

// AddressList handles GET /account/addresses
func (h *AccountHandler) AddressList(w http.ResponseWriter, r *http.Request) {
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
		handler.InternalErrorResponse(w, r, err)
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

// AddressCreate handles POST /account/addresses
func (h *AccountHandler) AddressCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderAddressError(w, r, "Invalid form data")
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
		h.renderAddressError(w, r, "Please fill in all required fields")
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
		h.renderAddressError(w, r, "Failed to create address")
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
		h.renderAddressError(w, r, "Failed to link address")
		return
	}

	// Check if htmx request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// AddressUpdate handles POST /account/addresses/{id}
func (h *AccountHandler) AddressUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Address ID required"))
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid address ID"))
		return
	}

	// Verify user owns this address
	_, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderAddressError(w, r, "Invalid form data")
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
		h.renderAddressError(w, r, "Please fill in all required fields")
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
		h.renderAddressError(w, r, "Failed to update address")
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

// AddressDelete handles POST /account/addresses/{id}/delete
func (h *AccountHandler) AddressDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Address ID required"))
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid address ID"))
		return
	}

	// Delete the customer-address link (address itself remains for order history)
	err := h.repo.DeleteCustomerAddress(ctx, repository.DeleteCustomerAddressParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		AddressID: addressUUID,
	})
	if err != nil {
		h.renderAddressError(w, r, "Failed to delete address")
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// AddressSetDefault handles POST /account/addresses/{id}/default
func (h *AccountHandler) AddressSetDefault(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Address ID required"))
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid address ID"))
		return
	}

	// Verify user owns this address
	_, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	// Set as default shipping address
	err = h.repo.SetDefaultShippingAddress(ctx, repository.SetDefaultShippingAddressParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		AddressID: addressUUID,
	})
	if err != nil {
		h.renderAddressError(w, r, "Failed to set default address")
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/addresses")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/addresses", http.StatusSeeOther)
}

// AddressGetJSON handles GET /account/addresses/{id}/json (for modal editing)
func (h *AccountHandler) AddressGetJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	addressID := r.PathValue("id")
	if addressID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Address ID required"))
		return
	}

	var addressUUID pgtype.UUID
	if err := addressUUID.Scan(addressID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid address ID"))
		return
	}

	address, err := h.repo.GetAddressByIDForUser(ctx, repository.GetAddressByIDForUserParams{
		ID:       addressUUID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
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
func (h *AccountHandler) toDisplayAddress(addr repository.ListAddressesForUserRow) AddressDisplay {
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

// renderAddressError sends an error response for address operations
func (h *AccountHandler) renderAddressError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Retarget", "#error-message")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`<p class="text-red-600 text-sm">%s</p>`, message)))
		return
	}
	handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "%s", message))
}

// =============================================================================
// Payment Methods
// =============================================================================

// PaymentMethodList handles GET /account/payment-methods
func (h *AccountHandler) PaymentMethodList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/payment-methods", http.StatusSeeOther)
		return
	}

	// Get payment methods
	paymentMethods, err := h.accountService.ListPaymentMethods(ctx, h.tenantID, user.ID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := BaseTemplateData(r)
	data["PaymentMethods"] = paymentMethods

	h.renderer.RenderHTTP(w, "storefront/payment_methods", data)
}

// PaymentMethodSetDefault handles POST /account/payment-methods/{id}/default
func (h *AccountHandler) PaymentMethodSetDefault(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
		return
	}

	// Get payment method ID from path
	paymentMethodIDStr := r.PathValue("id")
	if paymentMethodIDStr == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Payment method ID required"))
		return
	}

	var paymentMethodID pgtype.UUID
	if err := paymentMethodID.Scan(paymentMethodIDStr); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid payment method ID"))
		return
	}

	// Verify ownership and get billing_customer_id
	pm, err := h.repo.GetPaymentMethodByID(ctx, repository.GetPaymentMethodByIDParams{
		ID:       paymentMethodID,
		TenantID: h.tenantID,
		UserID:   user.ID,
	})
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	// Set as default (this updates all payment methods for the billing customer)
	err = h.repo.SetDefaultPaymentMethod(ctx, repository.SetDefaultPaymentMethodParams{
		BillingCustomerID: pm.BillingCustomerID,
		ID:                paymentMethodID,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Handle htmx request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/payment-methods")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/account/payment-methods", http.StatusSeeOther)
}

// PaymentMethodPortal handles GET /account/payment-methods/portal
func (h *AccountHandler) PaymentMethodPortal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/login?return_to=/account/payment-methods/portal", http.StatusSeeOther)
		return
	}

	// Create portal session with return URL to payment methods page
	portalURL, err := h.subscriptionService.CreateCustomerPortalSession(ctx, domain.PortalSessionParams{
		TenantID:  h.tenantID,
		UserID:    user.ID,
		ReturnURL: "/account/payment-methods",
	})

	if err != nil {
		// If user doesn't have a Stripe customer, redirect back with message
		http.Redirect(w, r, "/account/payment-methods?error=no_stripe_customer", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}

// =============================================================================
// Profile Settings
// =============================================================================

// ProfileShow handles GET /account/settings
func (h *AccountHandler) ProfileShow(w http.ResponseWriter, r *http.Request) {
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

// ProfileUpdate handles POST /account/settings/profile
func (h *AccountHandler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
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

// PasswordChange handles POST /account/settings/password
func (h *AccountHandler) PasswordChange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		handler.UnauthorizedResponse(w, r)
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
func (h *AccountHandler) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	encodedMsg := url.QueryEscape(message)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/account/settings?error="+encodedMsg)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/account/settings?error="+encodedMsg, http.StatusSeeOther)
}
