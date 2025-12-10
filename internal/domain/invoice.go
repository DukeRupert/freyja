package domain

import (
	"context"
	"time"

	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// Invoice-related domain errors.
var (
	ErrInvoiceAlreadyFinalized = &Error{Code: ECONFLICT, Message: "Invoice already finalized"}
	ErrInvoiceNotDraft         = &Error{Code: EINVALID, Message: "Invoice must be in draft status"}
	ErrInvoiceAlreadyPaid      = &Error{Code: ECONFLICT, Message: "Invoice already paid in full"}
	ErrPaymentExceedsBalance   = &Error{Code: EINVALID, Message: "Payment amount exceeds invoice balance"}
	ErrNoOrdersToInvoice       = &Error{Code: ENOTFOUND, Message: "No uninvoiced orders found for period"}
	ErrOrderNotWholesale       = &Error{Code: EINVALID, Message: "Order is not a wholesale order"}
	ErrOrderAlreadyInvoiced    = &Error{Code: ECONFLICT, Message: "Order already invoiced"}
	ErrInvoiceNumberGeneration = &Error{Code: EINTERNAL, Message: "Failed to generate invoice number"}
	ErrNoPaymentTermsAvailable = &Error{Code: ENOTFOUND, Message: "No payment terms available"}
	ErrNotWholesaleUser        = &Error{Code: EFORBIDDEN, Message: "User is not a wholesale customer"}
	ErrMinimumSpendNotMet      = &Error{Code: EINVALID, Message: "Order does not meet minimum spend requirement"}
)

// Payment terms errors.
var (
	ErrPaymentTermsNotFound      = &Error{Code: ENOTFOUND, Message: "Payment terms not found"}
	ErrPaymentTermsInUse         = &Error{Code: ECONFLICT, Message: "Payment terms in use by customers or invoices"}
	ErrDuplicatePaymentTermsCode = &Error{Code: ECONFLICT, Message: "Payment terms code already exists"}
)

// Fulfillment errors.
var (
	ErrShipmentNotFound       = &Error{Code: ENOTFOUND, Message: "Shipment not found"}
	ErrExceedsOrderedQuantity = &Error{Code: EINVALID, Message: "Shipment quantity exceeds ordered quantity"}
	ErrItemAlreadyFulfilled   = &Error{Code: ECONFLICT, Message: "Order item already fully fulfilled"}
	ErrNoItemsToShip          = &Error{Code: EINVALID, Message: "No items to ship"}
)

// InvoiceService manages wholesale invoices, payment tracking, and Stripe integration.
type InvoiceService interface {
	// CreateInvoice creates an invoice for one or more orders.
	// For consolidated invoices, pass multiple order IDs.
	CreateInvoice(ctx context.Context, params CreateInvoiceParams) (*InvoiceDetail, error)

	// GetInvoice retrieves an invoice by ID with full details.
	GetInvoice(ctx context.Context, invoiceID string) (*InvoiceDetail, error)

	// GetInvoiceByNumber retrieves an invoice by invoice number.
	GetInvoiceByNumber(ctx context.Context, invoiceNumber string) (*InvoiceDetail, error)

	// ListInvoices lists invoices for admin with pagination.
	ListInvoices(ctx context.Context, limit, offset int32) ([]InvoiceSummary, error)

	// ListInvoicesForUser lists invoices for a specific customer.
	ListInvoicesForUser(ctx context.Context, userID string, limit, offset int32) ([]repository.Invoice, error)

	// UpdateInvoiceStatus updates invoice status.
	UpdateInvoiceStatus(ctx context.Context, invoiceID string, status string) error

	// RecordPayment records a payment against an invoice.
	RecordPayment(ctx context.Context, params RecordPaymentParams) error

	// SendInvoice finalizes and sends an invoice via email.
	SendInvoice(ctx context.Context, invoiceID string) error

	// SyncInvoiceFromStripe handles Stripe webhook events for invoice updates.
	SyncInvoiceFromStripe(ctx context.Context, stripeInvoiceID string) error

	// GenerateConsolidatedInvoice creates an invoice for all uninvoiced orders
	// within a customer's billing period.
	GenerateConsolidatedInvoice(ctx context.Context, params ConsolidatedInvoiceParams) (*InvoiceDetail, error)

	// GetOverdueInvoices returns all overdue invoices for the tenant.
	GetOverdueInvoices(ctx context.Context) ([]repository.ListOverdueInvoicesRow, error)

	// MarkInvoicesOverdue updates status for invoices past due date.
	// Called by nightly background job.
	MarkInvoicesOverdue(ctx context.Context) (int, error)
}

// CreateInvoiceParams contains parameters for creating an invoice.
type CreateInvoiceParams struct {
	UserID             string
	OrderIDs           []string
	PaymentTermsID     string // Optional - uses customer's default if not provided
	BillingAddressID   string // Optional - uses customer's default if not provided
	CustomerNotes      string
	InternalNotes      string
	BillingPeriodStart *time.Time // For consolidated invoices
	BillingPeriodEnd   *time.Time // For consolidated invoices
	IsProforma         bool
	SendImmediately    bool
}

// RecordPaymentParams contains parameters for recording a payment.
type RecordPaymentParams struct {
	InvoiceID        string
	PaymentID        string // Optional - links to existing payment record
	AmountCents      int32
	PaymentMethod    string // "stripe", "check", "wire_transfer", "cash"
	PaymentReference string
	PaymentDate      time.Time
	Notes            string
}

// ConsolidatedInvoiceParams contains parameters for generating a consolidated invoice.
type ConsolidatedInvoiceParams struct {
	UserID             string
	BillingPeriodStart time.Time
	BillingPeriodEnd   time.Time
}

// InvoiceDetail aggregates invoice with items, orders, and customer details.
type InvoiceDetail struct {
	Invoice        repository.Invoice
	Items          []repository.InvoiceItem
	Orders         []repository.GetInvoiceOrdersRow
	Payments       []repository.InvoicePayment
	Customer       *repository.User
	PaymentTerms   *repository.PaymentTerm
	BillingAddress *repository.Address
}

// InvoiceSummary is a lightweight invoice representation for lists.
type InvoiceSummary struct {
	ID            pgtype.UUID
	InvoiceNumber string
	Status        string
	TotalCents    int32
	BalanceCents  int32
	Currency      string
	DueDate       pgtype.Date
	CreatedAt     pgtype.Timestamptz
	CustomerEmail string
	CustomerName  string
	CompanyName   string
	IsProforma    bool
}
