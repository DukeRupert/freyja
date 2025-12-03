package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceListHandler shows all invoices for admin
type InvoiceListHandler struct {
	invoiceService service.InvoiceService
	renderer       *handler.Renderer
}

// NewInvoiceListHandler creates a new invoice list handler
func NewInvoiceListHandler(invoiceService service.InvoiceService, renderer *handler.Renderer) *InvoiceListHandler {
	return &InvoiceListHandler{
		invoiceService: invoiceService,
		renderer:       renderer,
	}
}

func (h *InvoiceListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filter from query params
	statusFilter := r.URL.Query().Get("status")

	// Get invoices with pagination
	invoices, err := h.invoiceService.ListInvoices(r.Context(), 100, 0)
	if err != nil {
		http.Error(w, "Failed to load invoices", http.StatusInternalServerError)
		return
	}

	// Filter by status if specified
	var filteredInvoices []service.InvoiceSummary
	if statusFilter != "" {
		for _, inv := range invoices {
			if inv.Status == statusFilter {
				filteredInvoices = append(filteredInvoices, inv)
			}
		}
	} else {
		filteredInvoices = invoices
	}

	// Calculate stats
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

// InvoiceDetailHandler shows invoice details with actions
type InvoiceDetailHandler struct {
	invoiceService service.InvoiceService
	repo           repository.Querier
	renderer       *handler.Renderer
	tenantID       pgtype.UUID
}

// NewInvoiceDetailHandler creates a new invoice detail handler
func NewInvoiceDetailHandler(invoiceService service.InvoiceService, repo repository.Querier, renderer *handler.Renderer, tenantID string) *InvoiceDetailHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &InvoiceDetailHandler{
		invoiceService: invoiceService,
		repo:           repo,
		renderer:       renderer,
		tenantID:       tenantUUID,
	}
}

func (h *InvoiceDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}

	// Get invoice details
	invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	// Get invoice items
	var invoiceUUID pgtype.UUID
	if err := invoiceUUID.Scan(invoiceID); err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	items, err := h.repo.GetInvoiceItems(r.Context(), invoiceUUID)
	if err != nil {
		http.Error(w, "Failed to load invoice items", http.StatusInternalServerError)
		return
	}

	// Get payments
	payments, err := h.repo.GetInvoicePayments(r.Context(), invoiceUUID)
	if err != nil {
		http.Error(w, "Failed to load payments", http.StatusInternalServerError)
		return
	}

	// Get linked orders
	orders, err := h.repo.GetInvoiceOrders(r.Context(), invoiceUUID)
	if err != nil {
		// Not critical - may be empty
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

// SendInvoiceHandler handles sending an invoice to the customer
type SendInvoiceHandler struct {
	invoiceService service.InvoiceService
}

// NewSendInvoiceHandler creates a new send invoice handler
func NewSendInvoiceHandler(invoiceService service.InvoiceService) *SendInvoiceHandler {
	return &SendInvoiceHandler{
		invoiceService: invoiceService,
	}
}

func (h *SendInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}

	// Send the invoice
	err := h.invoiceService.SendInvoice(r.Context(), invoiceID)
	if err != nil {
		http.Error(w, "Failed to send invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to invoice detail
	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// VoidInvoiceHandler handles voiding an invoice
type VoidInvoiceHandler struct {
	invoiceService service.InvoiceService
}

// NewVoidInvoiceHandler creates a new void invoice handler
func NewVoidInvoiceHandler(invoiceService service.InvoiceService) *VoidInvoiceHandler {
	return &VoidInvoiceHandler{
		invoiceService: invoiceService,
	}
}

func (h *VoidInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}

	// Void the invoice
	err := h.invoiceService.UpdateInvoiceStatus(r.Context(), invoiceID, "void")
	if err != nil {
		http.Error(w, "Failed to void invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to invoice detail
	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// RecordPaymentHandler handles recording a manual payment on an invoice
type RecordPaymentHandler struct {
	invoiceService service.InvoiceService
	renderer       *handler.Renderer
}

// NewRecordPaymentHandler creates a new record payment handler
func NewRecordPaymentHandler(invoiceService service.InvoiceService, renderer *handler.Renderer) *RecordPaymentHandler {
	return &RecordPaymentHandler{
		invoiceService: invoiceService,
		renderer:       renderer,
	}
}

func (h *RecordPaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.PathValue("id")
	if invoiceID == "" {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		// Show payment form
		invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
		if err != nil {
			http.Error(w, "Invoice not found", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"CurrentPath": r.URL.Path,
			"Invoice":     invoice,
		}

		h.renderer.RenderHTTP(w, "admin/record_payment", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Parse amount (in dollars, convert to cents)
	amountStr := r.FormValue("amount")
	amountDollars, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amountDollars <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	amountCents := int32(amountDollars * 100)

	paymentMethod := r.FormValue("payment_method")
	if paymentMethod == "" {
		paymentMethod = "other"
	}

	reference := r.FormValue("reference")
	notes := r.FormValue("notes")

	// Record the payment
	err = h.invoiceService.RecordPayment(r.Context(), service.RecordPaymentParams{
		InvoiceID:        invoiceID,
		AmountCents:      amountCents,
		PaymentMethod:    paymentMethod,
		PaymentReference: reference,
		Notes:            notes,
	})
	if err != nil {
		http.Error(w, "Failed to record payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to invoice detail
	http.Redirect(w, r, "/admin/invoices/"+invoiceID, http.StatusSeeOther)
}

// CreateInvoiceHandler handles creating a new invoice
type CreateInvoiceHandler struct {
	invoiceService service.InvoiceService
	repo           repository.Querier
	renderer       *handler.Renderer
	tenantID       pgtype.UUID
}

// NewCreateInvoiceHandler creates a new invoice creation handler
func NewCreateInvoiceHandler(invoiceService service.InvoiceService, repo repository.Querier, renderer *handler.Renderer, tenantID string) *CreateInvoiceHandler {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		panic(fmt.Sprintf("invalid tenant ID: %v", err))
	}

	return &CreateInvoiceHandler{
		invoiceService: invoiceService,
		repo:           repo,
		renderer:       renderer,
		tenantID:       tenantUUID,
	}
}

func (h *CreateInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Show create invoice form with customer selection
		customers, err := h.repo.ListWholesaleCustomers(r.Context(), repository.ListWholesaleCustomersParams{
			TenantID: h.tenantID,
			Limit:    100,
			Offset:   0,
		})
		if err != nil {
			http.Error(w, "Failed to load customers", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"CurrentPath": r.URL.Path,
			"Customers":   customers,
		}

		h.renderer.RenderHTTP(w, "admin/invoice_form", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	customerID := r.FormValue("customer_id")
	if customerID == "" {
		http.Error(w, "Customer required", http.StatusBadRequest)
		return
	}

	// Get uninvoiced orders for this customer
	var customerUUID pgtype.UUID
	if err := customerUUID.Scan(customerID); err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	// List orders that need invoicing (wholesale orders without invoice)
	orders, err := h.repo.GetUninvoicedOrdersForUser(r.Context(), repository.GetUninvoicedOrdersForUserParams{
		TenantID: h.tenantID,
		UserID:   customerUUID,
	})
	if err != nil {
		http.Error(w, "Failed to load orders", http.StatusInternalServerError)
		return
	}

	// Get selected order IDs
	orderIDs := r.Form["order_ids"]
	if len(orderIDs) == 0 {
		// If no orders selected, show order selection page
		data := map[string]interface{}{
			"CurrentPath": r.URL.Path,
			"CustomerID":  customerID,
			"Orders":      orders,
		}
		h.renderer.RenderHTTP(w, "admin/invoice_select_orders", data)
		return
	}

	// Create invoice from selected orders
	invoice, err := h.invoiceService.CreateInvoice(r.Context(), service.CreateInvoiceParams{
		UserID:   customerID,
		OrderIDs: orderIDs,
	})
	if err != nil {
		http.Error(w, "Failed to create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the new invoice
	http.Redirect(w, r, "/admin/invoices/"+invoice.Invoice.ID.String(), http.StatusSeeOther)
}
