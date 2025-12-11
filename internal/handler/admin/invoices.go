package admin

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/handler"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceHandler handles all invoice-related admin routes
type InvoiceHandler struct {
	invoiceService domain.InvoiceService
	repo           repository.Querier
	renderer       *handler.Renderer
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(invoiceService domain.InvoiceService, repo repository.Querier, renderer *handler.Renderer) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
		repo:           repo,
		renderer:       renderer,
	}
}

// List handles GET /admin/invoices
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	invoices, err := h.invoiceService.ListInvoices(r.Context(), 100, 0)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	var filteredInvoices []domain.InvoiceSummary
	if statusFilter != "" {
		for _, inv := range invoices {
			if inv.Status == statusFilter {
				filteredInvoices = append(filteredInvoices, inv)
			}
		}
	} else {
		filteredInvoices = invoices
	}

	var stats struct {
		TotalCount   int
		DraftCount   int
		SentCount    int
		PaidCount    int
		OverdueCount int
		TotalOwed    int32
	}

	for _, inv := range invoices {
		stats.TotalCount++
		switch inv.Status {
		case "draft":
			stats.DraftCount++
		case "sent", "viewed":
			stats.SentCount++
			stats.TotalOwed += inv.BalanceCents
		case "paid":
			stats.PaidCount++
		case "overdue":
			stats.OverdueCount++
			stats.TotalOwed += inv.BalanceCents
		}
	}

	data := map[string]interface{}{
		"CurrentPath":  r.URL.Path,
		"Invoices":     filteredInvoices,
		"Stats":        stats,
		"StatusFilter": statusFilter,
	}

	h.renderer.RenderHTTP(w, "admin/invoices", data)
}

// Detail handles GET /admin/invoices/{id}
func (h *InvoiceHandler) Detail(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invoice ID required"))
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	var invoiceUUID pgtype.UUID
	if err := invoiceUUID.Scan(invoiceID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid invoice ID"))
		return
	}

	items, err := h.repo.GetInvoiceItems(r.Context(), invoiceUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	payments, err := h.repo.GetInvoicePayments(r.Context(), invoiceUUID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	orders, err := h.repo.GetInvoiceOrders(r.Context(), invoiceUUID)
	if err != nil {
		orders = nil
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Invoice":     invoice,
		"Items":       items,
		"Payments":    payments,
		"Orders":      orders,
	}

	h.renderer.RenderHTTP(w, "admin/invoice_detail", data)
}

// Send handles POST /admin/invoices/{id}/send
func (h *InvoiceHandler) Send(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invoice ID required"))
		return
	}

	err := h.invoiceService.SendInvoice(r.Context(), invoiceID)
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// Void handles POST /admin/invoices/{id}/void
func (h *InvoiceHandler) Void(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invoice ID required"))
		return
	}

	err := h.invoiceService.UpdateInvoiceStatus(r.Context(), invoiceID, "void")
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// ShowPaymentForm handles GET /admin/invoices/{id}/payment
func (h *InvoiceHandler) ShowPaymentForm(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invoice ID required"))
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		handler.NotFoundResponse(w, r)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Invoice":     invoice,
	}

	h.renderer.RenderHTTP(w, "admin/record_payment", data)
}

// HandlePayment handles POST /admin/invoices/{id}/payment
func (h *InvoiceHandler) HandlePayment(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invoice ID required"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	amountStr := r.FormValue("amount")
	amountDollars, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amountDollars <= 0 {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid amount"))
		return
	}
	amountCents := int32(amountDollars * 100)

	paymentMethod := r.FormValue("payment_method")
	if paymentMethod == "" {
		paymentMethod = "other"
	}

	reference := r.FormValue("reference")
	notes := r.FormValue("notes")

	err = h.invoiceService.RecordPayment(r.Context(), service.RecordPaymentParams{
		InvoiceID:        invoiceID,
		AmountCents:      amountCents,
		PaymentMethod:    paymentMethod,
		PaymentReference: reference,
		Notes:            notes,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// ShowCreateForm handles GET /admin/invoices/new
func (h *InvoiceHandler) ShowCreateForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	customers, err := h.repo.ListWholesaleCustomers(ctx, repository.ListWholesaleCustomersParams{
		TenantID: tenantID,
		Limit:    100,
		Offset:   0,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Customers":   customers,
	}

	h.renderer.RenderHTTP(w, "admin/invoice_form", data)
}

// HandleCreate handles POST /admin/invoices/new
func (h *InvoiceHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := getTenantID(ctx)
	if !tenantID.Valid {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "No tenant context"))
		return
	}

	if err := r.ParseForm(); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid form data"))
		return
	}

	customerID := r.FormValue("customer_id")
	if customerID == "" {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Customer required"))
		return
	}

	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid customer ID"))
		return
	}

	orders, err := h.repo.GetUninvoicedOrdersForUser(ctx, repository.GetUninvoicedOrdersForUserParams{
		TenantID: tenantID,
		UserID:   customerUUID,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	orderIDs := r.Form["order_ids"]
	if len(orderIDs) == 0 {
		data := map[string]interface{}{
			"CurrentPath": r.URL.Path,
			"CustomerID":  customerID,
			"Orders":      orders,
		}
		h.renderer.RenderHTTP(w, "admin/invoice_select_orders", data)
		return
	}

	invoice, err := h.invoiceService.CreateInvoice(ctx, service.CreateInvoiceParams{
		UserID:   customerID,
		OrderIDs: orderIDs,
	})
	if err != nil {
		handler.InternalErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, "/admin/invoices/"+invoice.Invoice.ID.String(), http.StatusSeeOther)
}
