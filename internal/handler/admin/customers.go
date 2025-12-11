package admin

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// CustomerHandler handles all customer-related admin routes
type CustomerHandler struct {
	repo           repository.Querier
	invoiceService domain.InvoiceService
	renderer       *handler.Renderer
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(repo repository.Querier, invoiceService domain.InvoiceService, renderer *handler.Renderer) *CustomerHandler {
	return &CustomerHandler{
		repo:           repo,
		invoiceService: invoiceService,
		renderer:       renderer,
	}
}

// List handles GET /admin/customers
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	accountType := r.URL.Query().Get("type")

	var users []repository.User
	var err error

	if accountType != "" {
		users, err = h.repo.ListUsersByAccountType(ctx, repository.ListUsersByAccountTypeParams{
			TenantID:    tenantID,
			AccountType: accountType,
		})
	} else {
		users, err = h.repo.ListUsers(ctx, repository.ListUsersParams{
			TenantID: tenantID,
			Limit:    100,
			Offset:   0,
		})
	}

	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	type DisplayUser struct {
		ID                 pgtype.UUID
		Email              string
		FirstName          pgtype.Text
		LastName           pgtype.Text
		FullName           string
		AccountType        string
		Status             string
		CreatedAt          pgtype.Timestamptz
		CreatedAtFormatted string
		LastLoginFormatted string
		CompanyName        pgtype.Text
		WholesaleStatus    pgtype.Text
	}

	displayUsers := make([]DisplayUser, len(users))
	for i, user := range users {
		createdAtFormatted := ""
		if user.CreatedAt.Valid {
			createdAtFormatted = user.CreatedAt.Time.Format("Jan 2, 2006")
		}

		fullName := ""
		if user.FirstName.Valid && user.LastName.Valid {
			fullName = user.FirstName.String + " " + user.LastName.String
		} else if user.FirstName.Valid {
			fullName = user.FirstName.String
		} else if user.LastName.Valid {
			fullName = user.LastName.String
		}

		displayUsers[i] = DisplayUser{
			ID:                 user.ID,
			Email:              user.Email,
			FirstName:          user.FirstName,
			LastName:           user.LastName,
			FullName:           fullName,
			AccountType:        user.AccountType,
			Status:             user.Status,
			CreatedAt:          user.CreatedAt,
			CreatedAtFormatted: createdAtFormatted,
			LastLoginFormatted: "Never",
			CompanyName:        user.CompanyName,
			WholesaleStatus:    user.WholesaleApplicationStatus,
		}
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Customers":   displayUsers,
		"FilterType":  accountType,
	}

	h.renderer.RenderHTTP(w, "admin/customers", data)
}

// Detail handles GET /admin/customers/{id}
func (h *CustomerHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	customerID := r.PathValue("id")
	if customerID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Customer ID required"))
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid customer ID"))
		return
	}

	customer, err := h.repo.GetUserByID(ctx, customerUUID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if customer.TenantID != tenantID {
		handler.NotFoundResponse(w, r)
		return
	}

	var invoices []repository.Invoice
	if customer.AccountType == "wholesale" {
		invoices, err = h.invoiceService.ListInvoicesForUser(ctx, customerID, 20, 0)
		if err != nil {
			invoices = nil
		}
	}

	addresses, err := h.repo.ListAddressesForUser(ctx, repository.ListAddressesForUserParams{
		TenantID: tenantID,
		UserID:   customerUUID,
	})
	if err != nil {
		addresses = nil
	}

	fullName := ""
	if customer.FirstName.Valid && customer.LastName.Valid {
		fullName = customer.FirstName.String + " " + customer.LastName.String
	} else if customer.FirstName.Valid {
		fullName = customer.FirstName.String
	} else if customer.LastName.Valid {
		fullName = customer.LastName.String
	}

	var paymentTerms *repository.PaymentTerm
	if customer.AccountType == "wholesale" && customer.PaymentTermsID.Valid {
		pt, err := h.repo.GetPaymentTermsByID(ctx, repository.GetPaymentTermsByIDParams{
			TenantID: tenantID,
			ID:       customer.PaymentTermsID,
		})
		if err == nil {
			paymentTerms = &pt
		}
	}

	data := map[string]interface{}{
		"CurrentPath":  r.URL.Path,
		"Customer":     customer,
		"FullName":     fullName,
		"Invoices":     invoices,
		"Addresses":    addresses,
		"PaymentTerms": paymentTerms,
	}

	h.renderer.RenderHTTP(w, "admin/customer_detail", data)
}

// Edit handles GET /admin/customers/{id}/edit
func (h *CustomerHandler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	customerID := r.PathValue("id")
	if customerID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Customer ID required"))
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid customer ID"))
		return
	}

	customer, err := h.repo.GetUserByID(ctx, customerUUID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if customer.TenantID != tenantID {
		handler.NotFoundResponse(w, r)
		return
	}

	fullName := ""
	if customer.FirstName.Valid && customer.LastName.Valid {
		fullName = customer.FirstName.String + " " + customer.LastName.String
	} else if customer.FirstName.Valid {
		fullName = customer.FirstName.String
	} else if customer.LastName.Valid {
		fullName = customer.LastName.String
	}

	// Get payment terms for dropdown
	paymentTerms, _ := h.repo.ListPaymentTerms(ctx, tenantID)

	data := map[string]interface{}{
		"CurrentPath":   r.URL.Path,
		"Customer":      customer,
		"CustomerID":    customerID,
		"FullName":      fullName,
		"PaymentTerms":  paymentTerms,
		"StatusOptions": []string{"active", "suspended", "closed"},
	}

	h.renderer.RenderHTTP(w, "admin/customer_edit", data)
}

// Update handles POST /admin/customers/{id}
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	customerID := r.PathValue("id")
	if customerID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Customer ID required"))
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid customer ID"))
		return
	}

	customer, err := h.repo.GetUserByID(ctx, customerUUID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if customer.TenantID != tenantID {
		handler.NotFoundResponse(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	// Build update params
	params := repository.AdminUpdateCustomerParams{
		ID:       customerUUID,
		TenantID: tenantID,
	}

	// Handle optional text fields
	if firstName := r.FormValue("first_name"); firstName != "" {
		params.FirstName = pgtype.Text{String: firstName, Valid: true}
	}
	if lastName := r.FormValue("last_name"); lastName != "" {
		params.LastName = pgtype.Text{String: lastName, Valid: true}
	}
	if phone := r.FormValue("phone"); phone != "" {
		params.Phone = pgtype.Text{String: phone, Valid: true}
	}
	if companyName := r.FormValue("company_name"); companyName != "" {
		params.CompanyName = pgtype.Text{String: companyName, Valid: true}
	}
	if businessType := r.FormValue("business_type"); businessType != "" {
		params.BusinessType = pgtype.Text{String: businessType, Valid: true}
	}
	if taxID := r.FormValue("tax_id"); taxID != "" {
		params.TaxID = pgtype.Text{String: taxID, Valid: true}
	}
	if status := r.FormValue("status"); status != "" {
		params.Status = pgtype.Text{String: status, Valid: true}
	}
	if internalNote := r.FormValue("internal_note"); internalNote != "" {
		params.InternalNote = pgtype.Text{String: internalNote, Valid: true}
	}

	err = h.repo.AdminUpdateCustomer(ctx, params)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	// Redirect back to detail page
	http.Redirect(w, r, "/admin/customers/"+customerID, http.StatusSeeOther)
}

// WholesaleApproval handles POST /admin/customers/{id}/wholesale/{action}
func (h *CustomerHandler) WholesaleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	customerID := r.PathValue("id")
	if customerID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Customer ID required"))
		return
	}

	action := r.PathValue("action")
	if action != "approve" && action != "reject" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid action"))
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid customer ID"))
		return
	}

	customer, err := h.repo.GetUserByID(ctx, customerUUID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	if customer.TenantID != tenantID {
		handler.NotFoundResponse(w, r)
		return
	}

	if !customer.WholesaleApplicationStatus.Valid || customer.WholesaleApplicationStatus.String != "pending" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "No pending wholesale application"))
		return
	}

	var newStatus string
	if action == "approve" {
		newStatus = "approved"
	} else {
		newStatus = "rejected"
	}

	err = h.repo.UpdateWholesaleApplication(ctx, repository.UpdateWholesaleApplicationParams{
		ID:                         customerUUID,
		WholesaleApplicationStatus: pgtype.Text{String: newStatus, Valid: true},
		WholesaleApplicationNotes:  pgtype.Text{},
		WholesaleApprovedBy:        pgtype.UUID{},
		PaymentTerms:               pgtype.Text{String: "net_30", Valid: true},
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/customers/"+customerID, http.StatusSeeOther)
}
