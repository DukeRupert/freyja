package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/jobs"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceService is re-exported from domain for backwards compatibility.
type InvoiceService = domain.InvoiceService

// Type aliases for backwards compatibility - all types now live in domain package.
type (
	CreateInvoiceParams       = domain.CreateInvoiceParams
	RecordPaymentParams       = domain.RecordPaymentParams
	ConsolidatedInvoiceParams = domain.ConsolidatedInvoiceParams
	InvoiceDetail             = domain.InvoiceDetail
	InvoiceSummary            = domain.InvoiceSummary
)

type invoiceService struct {
	repo                repository.Querier
	paymentTermsService PaymentTermsService
	billingProvider     billing.Provider
	tenantID            pgtype.UUID
	tenantIDStr         string
}

// NewInvoiceService creates a new InvoiceService instance.
func NewInvoiceService(
	repo repository.Querier,
	paymentTermsService PaymentTermsService,
	billingProvider billing.Provider,
	tenantID string,
) (InvoiceService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &invoiceService{
		repo:                repo,
		paymentTermsService: paymentTermsService,
		billingProvider:     billingProvider,
		tenantID:            tenantUUID,
		tenantIDStr:         tenantID,
	}, nil
}

// CreateInvoice creates an invoice for one or more orders.
func (s *invoiceService) CreateInvoice(ctx context.Context, params CreateInvoiceParams) (*InvoiceDetail, error) {
	if len(params.OrderIDs) == 0 {
		return nil, ErrNoOrdersToInvoice
	}

	// Parse user ID
	var userID pgtype.UUID
	if err := userID.Scan(params.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// Verify user is wholesale
	if user.AccountType != "wholesale" {
		return nil, ErrNotWholesaleUser
	}

	// Determine payment terms
	var paymentTerms *repository.PaymentTerm
	if params.PaymentTermsID != "" {
		pt, err := s.paymentTermsService.GetPaymentTerms(ctx, params.PaymentTermsID)
		if err != nil {
			return nil, err
		}
		paymentTerms = pt
	} else if user.PaymentTermsID.Valid {
		pt, err := s.paymentTermsService.GetPaymentTerms(ctx, user.PaymentTermsID.String())
		if err == nil {
			paymentTerms = pt
		}
	}

	// Fall back to default payment terms
	if paymentTerms == nil {
		pt, err := s.paymentTermsService.GetDefaultPaymentTerms(ctx)
		if err != nil {
			return nil, ErrNoPaymentTermsAvailable
		}
		paymentTerms = pt
	}

	// Calculate totals from orders
	var subtotalCents, taxCents, shippingCents int32
	var orderUUIDs []pgtype.UUID
	for _, orderIDStr := range params.OrderIDs {
		var orderID pgtype.UUID
		if err := orderID.Scan(orderIDStr); err != nil {
			return nil, fmt.Errorf("invalid order ID: %w", err)
		}

		order, err := s.repo.GetOrder(ctx, repository.GetOrderParams{
			TenantID: s.tenantID,
			ID:       orderID,
		})
		if err != nil {
			return nil, ErrOrderNotFound
		}

		// Verify order belongs to user
		if order.UserID != userID {
			return nil, ErrOrderNotFound
		}

		// Verify order is wholesale
		if order.OrderType != "wholesale" {
			return nil, ErrOrderNotWholesale
		}

		subtotalCents += order.SubtotalCents
		taxCents += order.TaxCents
		shippingCents += order.ShippingCents
		orderUUIDs = append(orderUUIDs, orderID)
	}

	totalCents := subtotalCents + taxCents + shippingCents

	// Get billing address
	var billingAddressID pgtype.UUID
	if params.BillingAddressID != "" {
		if err := billingAddressID.Scan(params.BillingAddressID); err != nil {
			return nil, fmt.Errorf("invalid billing address ID: %w", err)
		}
	} else {
		// Use address from first order
		firstOrder, _ := s.repo.GetOrder(ctx, repository.GetOrderParams{
			TenantID: s.tenantID,
			ID:       orderUUIDs[0],
		})
		billingAddressID = firstOrder.BillingAddressID
	}

	// Generate invoice number
	invoiceNumberRow, err := s.repo.GenerateInvoiceNumber(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}
	invoiceNumber, ok := invoiceNumberRow.(string)
	if !ok {
		return nil, ErrInvoiceNumberGeneration
	}

	// Calculate due date
	invoiceDate := time.Now()
	dueDate := s.paymentTermsService.CalculateDueDateFromTerms(paymentTerms, invoiceDate)

	// Build optional fields
	customerNotes := pgtype.Text{}
	if params.CustomerNotes != "" {
		customerNotes.String = params.CustomerNotes
		customerNotes.Valid = true
	}

	internalNotes := pgtype.Text{}
	if params.InternalNotes != "" {
		internalNotes.String = params.InternalNotes
		internalNotes.Valid = true
	}

	billingPeriodStart := pgtype.Date{}
	if params.BillingPeriodStart != nil {
		billingPeriodStart.Time = *params.BillingPeriodStart
		billingPeriodStart.Valid = true
	}

	billingPeriodEnd := pgtype.Date{}
	if params.BillingPeriodEnd != nil {
		billingPeriodEnd.Time = *params.BillingPeriodEnd
		billingPeriodEnd.Valid = true
	}

	// Create invoice
	inv, err := s.repo.CreateInvoice(ctx, repository.CreateInvoiceParams{
		TenantID:           s.tenantID,
		UserID:             userID,
		InvoiceNumber:      invoiceNumber,
		Status:             "draft",
		SubtotalCents:      subtotalCents,
		TaxCents:           taxCents,
		ShippingCents:      shippingCents,
		DiscountCents:      0,
		TotalCents:         totalCents,
		PaidCents:          0,
		BalanceCents:       totalCents,
		Currency:           "USD",
		PaymentTerms:       paymentTerms.Code,
		PaymentTermsID:     paymentTerms.ID,
		DueDate:            pgtype.Date{Time: dueDate, Valid: true},
		BillingCustomerID:  pgtype.UUID{}, // Will be set when syncing with Stripe
		BillingAddressID:   billingAddressID,
		CustomerNotes:      customerNotes,
		InternalNotes:      internalNotes,
		BillingPeriodStart: billingPeriodStart,
		BillingPeriodEnd:   billingPeriodEnd,
		IsProforma:         params.IsProforma,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Link orders to invoice and create invoice items
	for _, orderID := range orderUUIDs {
		order, _ := s.repo.GetOrder(ctx, repository.GetOrderParams{
			TenantID: s.tenantID,
			ID:       orderID,
		})

		// Create invoice order link
		_, err := s.repo.CreateInvoiceOrder(ctx, repository.CreateInvoiceOrderParams{
			TenantID:        s.tenantID,
			InvoiceID:       inv.ID,
			OrderID:         orderID,
			OrderNumber:     order.OrderNumber,
			OrderTotalCents: order.TotalCents,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to link order to invoice: %w", err)
		}

		// Get order items and create invoice items
		orderItems, err := s.repo.GetOrderItems(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order items: %w", err)
		}

		for _, item := range orderItems {
			description := item.ProductName
			if item.VariantDescription.Valid {
				description += " - " + item.VariantDescription.String
			}

			// Convert quantity to pgtype.Numeric
			var quantity pgtype.Numeric
			_ = quantity.Scan(fmt.Sprintf("%d", item.Quantity))

			_, err := s.repo.CreateInvoiceItem(ctx, repository.CreateInvoiceItemParams{
				TenantID:        s.tenantID,
				InvoiceID:       inv.ID,
				ItemType:        "product",
				ProductSkuID:    item.ProductSkuID,
				OrderID:         orderID,
				Description:     description,
				Quantity:        quantity,
				UnitPriceCents:  item.UnitPriceCents,
				TotalPriceCents: item.TotalPriceCents,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create invoice item: %w", err)
			}
		}

		// Add shipping as line item if present
		if order.ShippingCents > 0 {
			var shippingQty pgtype.Numeric
			_ = shippingQty.Scan("1")

			_, err := s.repo.CreateInvoiceItem(ctx, repository.CreateInvoiceItemParams{
				TenantID:        s.tenantID,
				InvoiceID:       inv.ID,
				ItemType:        "shipping",
				ProductSkuID:    pgtype.UUID{},
				OrderID:         orderID,
				Description:     "Shipping - Order " + order.OrderNumber,
				Quantity:        shippingQty,
				UnitPriceCents:  order.ShippingCents,
				TotalPriceCents: order.ShippingCents,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create shipping invoice item: %w", err)
			}
		}
	}

	// If SendImmediately, sync with Stripe and send
	if params.SendImmediately {
		// Best-effort send - invoice was already created, could retry via background job
		_ = s.SendInvoice(ctx, inv.ID.String())
	}

	return s.GetInvoice(ctx, inv.ID.String())
}

// GetInvoice retrieves an invoice by ID with full details.
func (s *invoiceService) GetInvoice(ctx context.Context, invoiceID string) (*InvoiceDetail, error) {
	var invID pgtype.UUID
	if err := invID.Scan(invoiceID); err != nil {
		return nil, fmt.Errorf("invalid invoice ID: %w", err)
	}

	inv, err := s.repo.GetInvoiceByID(ctx, repository.GetInvoiceByIDParams{
		ID:       invID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	// Get invoice items
	items, err := s.repo.GetInvoiceItems(ctx, invID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice items: %w", err)
	}

	// Get linked orders
	orders, err := s.repo.GetInvoiceOrders(ctx, invID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice orders: %w", err)
	}

	// Get payments
	payments, err := s.repo.GetInvoicePayments(ctx, invID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice payments: %w", err)
	}

	// Get customer
	user, err := s.repo.GetUserByID(ctx, inv.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Get payment terms
	var paymentTerms *repository.PaymentTerm
	if inv.PaymentTermsID.Valid {
		pt, err := s.paymentTermsService.GetPaymentTerms(ctx, inv.PaymentTermsID.String())
		if err == nil {
			paymentTerms = pt
		}
	}

	return &InvoiceDetail{
		Invoice:      inv,
		Items:        items,
		Orders:       orders,
		Payments:     payments,
		Customer:     &user,
		PaymentTerms: paymentTerms,
	}, nil
}

// GetInvoiceByNumber retrieves an invoice by invoice number.
func (s *invoiceService) GetInvoiceByNumber(ctx context.Context, invoiceNumber string) (*InvoiceDetail, error) {
	inv, err := s.repo.GetInvoiceByNumber(ctx, repository.GetInvoiceByNumberParams{
		TenantID:      s.tenantID,
		InvoiceNumber: invoiceNumber,
	})
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	return s.GetInvoice(ctx, inv.ID.String())
}

// ListInvoices lists invoices for admin with pagination.
func (s *invoiceService) ListInvoices(ctx context.Context, limit, offset int32) ([]InvoiceSummary, error) {
	rows, err := s.repo.ListInvoices(ctx, repository.ListInvoicesParams{
		TenantID: s.tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	summaries := make([]InvoiceSummary, len(rows))
	for i, row := range rows {
		summaries[i] = InvoiceSummary{
			ID:            row.ID,
			InvoiceNumber: row.InvoiceNumber,
			Status:        row.Status,
			TotalCents:    row.TotalCents,
			BalanceCents:  row.BalanceCents,
			Currency:      row.Currency,
			DueDate:       row.DueDate,
			CreatedAt:     row.CreatedAt,
			CustomerEmail: row.CustomerEmail,
			CustomerName:  row.CustomerName,
			CompanyName:   row.CompanyName.String,
			IsProforma:    row.IsProforma,
		}
	}

	return summaries, nil
}

// ListInvoicesForUser lists invoices for a specific customer.
func (s *invoiceService) ListInvoicesForUser(ctx context.Context, userID string, limit, offset int32) ([]repository.Invoice, error) {
	var uID pgtype.UUID
	if err := uID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	invoices, err := s.repo.ListInvoicesForUser(ctx, repository.ListInvoicesForUserParams{
		TenantID: s.tenantID,
		UserID:   uID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	return invoices, nil
}

// UpdateInvoiceStatus updates invoice status.
func (s *invoiceService) UpdateInvoiceStatus(ctx context.Context, invoiceID string, status string) error {
	var invID pgtype.UUID
	if err := invID.Scan(invoiceID); err != nil {
		return fmt.Errorf("invalid invoice ID: %w", err)
	}

	return s.repo.UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
		TenantID: s.tenantID,
		ID:       invID,
		Status:   status,
	})
}

// RecordPayment records a payment against an invoice.
func (s *invoiceService) RecordPayment(ctx context.Context, params RecordPaymentParams) error {
	var invID pgtype.UUID
	if err := invID.Scan(params.InvoiceID); err != nil {
		return fmt.Errorf("invalid invoice ID: %w", err)
	}

	// Get invoice to check balance
	inv, err := s.repo.GetInvoiceByID(ctx, repository.GetInvoiceByIDParams{
		ID:       invID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return ErrInvoiceNotFound
	}

	// Validate payment doesn't exceed balance
	if params.AmountCents > inv.BalanceCents {
		return ErrPaymentExceedsBalance
	}

	// Build optional payment_id
	var paymentID pgtype.UUID
	if params.PaymentID != "" {
		if err := paymentID.Scan(params.PaymentID); err != nil {
			return fmt.Errorf("invalid payment ID: %w", err)
		}
	}

	// Build optional notes
	notes := pgtype.Text{}
	if params.Notes != "" {
		notes.String = params.Notes
		notes.Valid = true
	}

	paymentRef := pgtype.Text{}
	if params.PaymentReference != "" {
		paymentRef.String = params.PaymentReference
		paymentRef.Valid = true
	}

	// Record payment (triggers update invoice balance)
	_, err = s.repo.CreateInvoicePayment(ctx, repository.CreateInvoicePaymentParams{
		TenantID:         s.tenantID,
		InvoiceID:        invID,
		PaymentID:        paymentID,
		AmountCents:      params.AmountCents,
		PaymentMethod:    pgtype.Text{String: params.PaymentMethod, Valid: params.PaymentMethod != ""},
		PaymentReference: paymentRef,
		Notes:            notes,
		PaymentDate:      pgtype.Date{Time: params.PaymentDate, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to record payment: %w", err)
	}

	return nil
}

// SendInvoice finalizes and sends an invoice via email.
func (s *invoiceService) SendInvoice(ctx context.Context, invoiceID string) error {
	var invID pgtype.UUID
	if err := invID.Scan(invoiceID); err != nil {
		return fmt.Errorf("invalid invoice ID: %w", err)
	}

	inv, err := s.repo.GetInvoiceByID(ctx, repository.GetInvoiceByIDParams{
		ID:       invID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return ErrInvoiceNotFound
	}

	// Can only send draft invoices
	if inv.Status != "draft" {
		return ErrInvoiceNotDraft
	}

	// Get customer's Stripe customer ID
	billingCustomer, err := s.repo.GetBillingCustomerByUserID(ctx, repository.GetBillingCustomerByUserIDParams{
		TenantID: s.tenantID,
		UserID:   inv.UserID,
		Provider: "stripe",
	})

	var stripeCustomerID string
	if err != nil {
		// Auto-create Stripe customer
		user, err := s.repo.GetUserByID(ctx, inv.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		name := ""
		if user.FirstName.Valid {
			name = user.FirstName.String
		}
		if user.LastName.Valid {
			if name != "" {
				name += " "
			}
			name += user.LastName.String
		}

		customer, err := s.billingProvider.CreateCustomer(ctx, billing.CreateCustomerParams{
			Email: user.Email,
			Name:  name,
			Metadata: map[string]string{
				"tenant_id": s.tenantIDStr,
				"user_id":   inv.UserID.String(),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Stripe customer: %w", err)
		}
		stripeCustomerID = customer.ID

		// Save billing customer
		_, err = s.repo.CreateBillingCustomer(ctx, repository.CreateBillingCustomerParams{
			TenantID:           s.tenantID,
			UserID:             inv.UserID,
			Provider:           "stripe",
			ProviderCustomerID: stripeCustomerID,
			Metadata:           []byte("{}"),
		})
		// Best-effort customer creation - continue with sync regardless
		_ = err
	} else {
		stripeCustomerID = billingCustomer.ProviderCustomerID
	}

	// Create Stripe invoice
	stripeInv, err := s.billingProvider.CreateInvoice(ctx, billing.CreateInvoiceParams{
		CustomerID:       stripeCustomerID,
		TenantID:         s.tenantIDStr,
		Currency:         inv.Currency,
		Description:      fmt.Sprintf("Invoice %s", inv.InvoiceNumber),
		DueDate:          inv.DueDate.Time,
		CollectionMethod: "send_invoice",
		AutoAdvance:      false,
		Metadata: map[string]string{
			"tenant_id":  s.tenantIDStr,
			"invoice_id": inv.ID.String(),
		},
		IdempotencyKey: fmt.Sprintf("inv_%s", inv.ID.String()),
	})
	if err != nil {
		return fmt.Errorf("failed to create Stripe invoice: %w", err)
	}

	// Add line items to Stripe invoice
	items, _ := s.repo.GetInvoiceItems(ctx, invID)
	for _, item := range items {
		// Convert pgtype.Numeric to int32
		var qty int32 = 1
		if f, err := item.Quantity.Float64Value(); err == nil && f.Valid {
			qty = int32(f.Float64)
		}

		err := s.billingProvider.AddInvoiceItem(ctx, billing.AddInvoiceItemParams{
			CustomerID:  stripeCustomerID,
			InvoiceID:   stripeInv.ID,
			TenantID:    s.tenantIDStr,
			Description: item.Description,
			Quantity:    qty,
			UnitAmount:  item.UnitPriceCents,
			Currency:    inv.Currency,
		})
		if err != nil {
			return fmt.Errorf("failed to add invoice item to Stripe: %w", err)
		}
	}

	// Finalize Stripe invoice
	_, err = s.billingProvider.FinalizeInvoice(ctx, billing.FinalizeInvoiceParams{
		InvoiceID: stripeInv.ID,
		TenantID:  s.tenantIDStr,
	})
	if err != nil {
		return fmt.Errorf("failed to finalize Stripe invoice: %w", err)
	}

	// Send Stripe invoice
	err = s.billingProvider.SendInvoice(ctx, billing.SendInvoiceParams{
		InvoiceID: stripeInv.ID,
		TenantID:  s.tenantIDStr,
	})
	if err != nil {
		return fmt.Errorf("failed to send Stripe invoice: %w", err)
	}

	// Update local invoice with Stripe ID and status
	err = s.repo.UpdateInvoiceProviderID(ctx, repository.UpdateInvoiceProviderIDParams{
		TenantID:          s.tenantID,
		ID:                invID,
		Provider:          pgtype.Text{String: "stripe", Valid: true},
		ProviderInvoiceID: pgtype.Text{String: stripeInv.ID, Valid: true},
	})
	// Best-effort provider ID update - continue regardless
	_ = err

	err = s.repo.UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
		TenantID: s.tenantID,
		ID:       invID,
		Status:   "sent",
	})
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %w", err)
	}

	// Enqueue invoice sent email
	s.enqueueInvoiceSentEmail(ctx, inv, items)

	return nil
}

// enqueueInvoiceSentEmail enqueues an email notification for a sent invoice
func (s *invoiceService) enqueueInvoiceSentEmail(ctx context.Context, inv repository.Invoice, items []repository.InvoiceItem) {
	// Get user info for email
	user, err := s.repo.GetUserByID(ctx, inv.UserID)
	if err != nil {
		// Log but don't fail - email is not critical
		return
	}

	// Build customer name
	customerName := user.Email
	if user.FirstName.Valid {
		customerName = user.FirstName.String
		if user.LastName.Valid {
			customerName += " " + user.LastName.String
		}
	}

	// Convert invoice items to email format
	emailItems := make([]jobs.InvoiceItemData, len(items))
	for i, item := range items {
		var qty int = 1
		if f, err := item.Quantity.Float64Value(); err == nil && f.Valid {
			qty = int(f.Float64)
		}
		emailItems[i] = jobs.InvoiceItemData{
			Description: item.Description,
			Quantity:    qty,
			UnitCents:   int64(item.UnitPriceCents),
			TotalCents:  int64(item.TotalPriceCents),
		}
	}

	// Payment URL - Stripe sends its own email with payment link,
	// but we include our invoice detail page as a reference
	paymentURL := fmt.Sprintf("/invoices/%s", inv.ID.String())

	// Determine payment terms string
	paymentTerms := "Due upon receipt"
	if inv.DueDate.Valid && inv.CreatedAt.Valid {
		days := int(inv.DueDate.Time.Sub(inv.CreatedAt.Time).Hours() / 24)
		if days > 0 {
			paymentTerms = fmt.Sprintf("Net %d", days)
		}
	}

	// Convert tenant ID
	tenantUUID, err := uuid.Parse(s.tenantIDStr)
	if err != nil {
		return
	}

	payload := jobs.InvoiceSentPayload{
		InvoiceID:     uuid.UUID(inv.ID.Bytes),
		Email:         user.Email,
		CustomerName:  customerName,
		InvoiceNumber: inv.InvoiceNumber,
		InvoiceDate:   inv.CreatedAt.Time,
		DueDate:       inv.DueDate.Time,
		PaymentTerms:  paymentTerms,
		Items:         emailItems,
		SubtotalCents: int64(inv.SubtotalCents),
		ShippingCents: int64(inv.ShippingCents),
		TaxCents:      int64(inv.TaxCents),
		DiscountCents: int64(inv.DiscountCents),
		TotalCents:    int64(inv.TotalCents),
		PaymentURL:    paymentURL,
	}

	// Enqueue email job - ignore errors as email is not critical path
	_ = jobs.EnqueueInvoiceSentEmail(ctx, s.repo, tenantUUID, payload)
}

// SyncInvoiceFromStripe handles Stripe webhook events for invoice updates.
func (s *invoiceService) SyncInvoiceFromStripe(ctx context.Context, stripeInvoiceID string) error {
	// Get Stripe invoice
	stripeInv, err := s.billingProvider.GetInvoice(ctx, billing.GetInvoiceParams{
		InvoiceID: stripeInvoiceID,
		TenantID:  s.tenantIDStr,
	})
	if err != nil {
		return fmt.Errorf("failed to get Stripe invoice: %w", err)
	}

	// Find local invoice by Stripe ID
	inv, err := s.repo.GetInvoiceByProviderID(ctx, repository.GetInvoiceByProviderIDParams{
		TenantID:          s.tenantID,
		Provider:          pgtype.Text{String: "stripe", Valid: true},
		ProviderInvoiceID: pgtype.Text{String: stripeInvoiceID, Valid: true},
	})
	if err != nil {
		return ErrInvoiceNotFound
	}

	// Update status based on Stripe status
	var newStatus string
	switch stripeInv.Status {
	case "paid":
		newStatus = "paid"
	case "open":
		newStatus = "sent"
	case "void":
		newStatus = "void"
	case "uncollectible":
		newStatus = "cancelled"
	default:
		newStatus = inv.Status // Keep current
	}

	if newStatus != inv.Status {
		err = s.repo.UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
			TenantID: s.tenantID,
			ID:       inv.ID,
			Status:   newStatus,
		})
		if err != nil {
			return fmt.Errorf("failed to update invoice status: %w", err)
		}
	}

	// If paid, record the payment
	if stripeInv.Status == "paid" && stripeInv.AmountPaidCents > 0 {
		paidAt := time.Now()
		if stripeInv.PaidAt != nil {
			paidAt = *stripeInv.PaidAt
		}

		err = s.RecordPayment(ctx, RecordPaymentParams{
			InvoiceID:        inv.ID.String(),
			AmountCents:      int32(stripeInv.AmountPaidCents),
			PaymentMethod:    "stripe",
			PaymentReference: stripeInv.PaymentIntentID,
			PaymentDate:      paidAt,
			Notes:            "Payment via Stripe",
		})
		// Payment may already be recorded - don't fail
		_ = err
	}

	return nil
}

// GenerateConsolidatedInvoice creates an invoice for all uninvoiced orders
// within a customer's billing period.
func (s *invoiceService) GenerateConsolidatedInvoice(ctx context.Context, params ConsolidatedInvoiceParams) (*InvoiceDetail, error) {
	var userID pgtype.UUID
	if err := userID.Scan(params.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get uninvoiced orders in period
	orders, err := s.repo.GetUninvoicedOrdersInPeriod(ctx, repository.GetUninvoicedOrdersInPeriodParams{
		TenantID:    s.tenantID,
		UserID:      userID,
		CreatedAt:   pgtype.Timestamptz{Time: params.BillingPeriodStart, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: params.BillingPeriodEnd, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get uninvoiced orders: %w", err)
	}

	if len(orders) == 0 {
		return nil, nil // No orders to invoice - not an error
	}

	// Collect order IDs
	orderIDs := make([]string, len(orders))
	for i, order := range orders {
		orderIDs[i] = order.ID.String()
	}

	// Create consolidated invoice
	return s.CreateInvoice(ctx, CreateInvoiceParams{
		UserID:             params.UserID,
		OrderIDs:           orderIDs,
		BillingPeriodStart: &params.BillingPeriodStart,
		BillingPeriodEnd:   &params.BillingPeriodEnd,
		SendImmediately:    true,
	})
}

// GetOverdueInvoices returns all overdue invoices for the tenant.
func (s *invoiceService) GetOverdueInvoices(ctx context.Context) ([]repository.ListOverdueInvoicesRow, error) {
	return s.repo.ListOverdueInvoices(ctx, s.tenantID)
}

// MarkInvoicesOverdue updates status for invoices past due date and sends notifications.
func (s *invoiceService) MarkInvoicesOverdue(ctx context.Context) (int, error) {
	invoices, err := s.repo.ListOverdueInvoices(ctx, s.tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to list overdue invoices: %w", err)
	}

	tenantUUID, _ := uuid.Parse(s.tenantIDStr)
	count := 0
	for _, inv := range invoices {
		if inv.Status == "sent" || inv.Status == "viewed" {
			err := s.repo.UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
				TenantID: s.tenantID,
				ID:       inv.ID,
				Status:   "overdue",
			})
			if err == nil {
				count++
				// Enqueue overdue notification email
				s.enqueueInvoiceOverdueEmail(ctx, inv, tenantUUID)
			}
		}
	}

	return count, nil
}

// enqueueInvoiceOverdueEmail enqueues an overdue notification email for an invoice
func (s *invoiceService) enqueueInvoiceOverdueEmail(ctx context.Context, inv repository.ListOverdueInvoicesRow, tenantUUID uuid.UUID) {
	// Get user info for email
	user, err := s.repo.GetUserByID(ctx, inv.UserID)
	if err != nil {
		// Log but don't fail - email is not critical
		return
	}

	// Build customer name
	customerName := user.Email
	if user.FirstName.Valid {
		customerName = user.FirstName.String
		if user.LastName.Valid {
			customerName += " " + user.LastName.String
		}
	}

	// Calculate days overdue
	daysOverdue := 0
	if inv.DueDate.Valid {
		daysOverdue = int(time.Since(inv.DueDate.Time).Hours() / 24)
		if daysOverdue < 1 {
			daysOverdue = 1
		}
	}

	// Payment URL
	paymentURL := fmt.Sprintf("/invoices/%s", inv.ID.String())

	payload := jobs.InvoiceOverduePayload{
		InvoiceID:     uuid.UUID(inv.ID.Bytes),
		Email:         user.Email,
		CustomerName:  customerName,
		InvoiceNumber: inv.InvoiceNumber,
		DueDate:       inv.DueDate.Time,
		BalanceCents:  int64(inv.BalanceCents),
		DaysOverdue:   daysOverdue,
		PaymentURL:    paymentURL,
	}

	// Enqueue email job - ignore errors as email is not critical path
	_ = jobs.EnqueueInvoiceOverdueEmail(ctx, s.repo, tenantUUID, payload)
}
