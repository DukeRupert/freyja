package admin

import (
	"fmt"
	"net/http"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// CustomerHandler handles all customer-related admin routes
type CustomerHandler struct {
	repo           repository.Querier
	invoiceService service.InvoiceService
	renderer       *handler.Renderer
	tenantID       pgtype.UUID
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(repo repository.Querier, invoiceService service.InvoiceService, renderer *handler.Renderer, tenantID string) *CustomerHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &CustomerHandler{
		repo:           repo,
		invoiceService: invoiceService,
		renderer:       renderer,
		tenantID:       tenantUUID,
	}
}

// List handles GET /admin/customers
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	accountType := r.URL.Query().Get("type")

	var users []repository.User
	var err error

	if accountType != "" {
		users, err = h.repo.ListUsersByAccountType(r.Context(), repository.ListUsersByAccountTypeParams{
			TenantID:    h.tenantID,
			AccountType: accountType,
		})
	} else {
		users, err = h.repo.ListUsers(r.Context(), repository.ListUsersParams{
			TenantID: h.tenantID,
			Limit:    100,
			Offset:   0,
		})
	}

	if err != nil {
		http.Error(w, "Failed to load customers", http.StatusInternalServerError)
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
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID required", http.StatusBadRequest)
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	customer, err := h.repo.GetUserByID(r.Context(), customerUUID)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	if customer.TenantID != h.tenantID {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	var invoices []repository.Invoice
	if customer.AccountType == "wholesale" {
		invoices, err = h.invoiceService.ListInvoicesForUser(r.Context(), customerID, 20, 0)
		if err != nil {
			invoices = nil
		}
	}

	addresses, err := h.repo.ListAddressesForUser(r.Context(), repository.ListAddressesForUserParams{
		TenantID: h.tenantID,
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
		pt, err := h.repo.GetPaymentTermsByID(r.Context(), repository.GetPaymentTermsByIDParams{
			TenantID: h.tenantID,
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

// WholesaleApproval handles POST /admin/customers/{id}/wholesale/{action}
func (h *CustomerHandler) WholesaleApproval(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID required", http.StatusBadRequest)
		return
	}

	action := r.PathValue("action")
	if action != "approve" && action != "reject" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	customer, err := h.repo.GetUserByID(r.Context(), customerUUID)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	if customer.TenantID != h.tenantID {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	if !customer.WholesaleApplicationStatus.Valid || customer.WholesaleApplicationStatus.String != "pending" {
		http.Error(w, "No pending wholesale application", http.StatusBadRequest)
		return
	}

	var newStatus string
	if action == "approve" {
		newStatus = "approved"
	} else {
		newStatus = "rejected"
	}

	err = h.repo.UpdateWholesaleApplication(r.Context(), repository.UpdateWholesaleApplicationParams{
		ID:                         customerUUID,
		WholesaleApplicationStatus: pgtype.Text{String: newStatus, Valid: true},
		WholesaleApplicationNotes:  pgtype.Text{},
		WholesaleApprovedBy:        pgtype.UUID{},
		PaymentTerms:               pgtype.Text{String: "net_30", Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to update wholesale status", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/customers/"+customerID, http.StatusSeeOther)
}
