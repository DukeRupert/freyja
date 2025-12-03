-- Invoice Queries
-- Manages wholesale billing invoices

-- =============================================================================
-- INVOICE CRUD
-- =============================================================================

-- name: CreateInvoice :one
-- Create a new invoice
INSERT INTO invoices (
    tenant_id,
    user_id,
    invoice_number,
    status,
    subtotal_cents,
    tax_cents,
    shipping_cents,
    discount_cents,
    total_cents,
    paid_cents,
    balance_cents,
    currency,
    payment_terms,
    payment_terms_id,
    due_date,
    billing_customer_id,
    billing_address_id,
    customer_notes,
    internal_notes,
    billing_period_start,
    billing_period_end,
    is_proforma
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
)
RETURNING *;

-- name: GetInvoiceByID :one
-- Get invoice by ID
SELECT * FROM invoices
WHERE id = $1
  AND tenant_id = $2
LIMIT 1;

-- name: GetInvoiceByNumber :one
-- Get invoice by invoice number
SELECT * FROM invoices
WHERE tenant_id = $1
  AND invoice_number = $2
LIMIT 1;

-- name: GetInvoiceByProviderID :one
-- Get invoice by billing provider ID (for Stripe webhook handling)
SELECT * FROM invoices
WHERE tenant_id = $1
  AND provider = $2
  AND provider_invoice_id = $3
LIMIT 1;

-- name: UpdateInvoiceStatus :exec
-- Update invoice status
UPDATE invoices
SET
    status = $3,
    sent_at = CASE WHEN $3 = 'sent' AND sent_at IS NULL THEN NOW() ELSE sent_at END,
    voided_at = CASE WHEN $3 = 'void' THEN NOW() ELSE voided_at END,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: UpdateInvoiceProviderID :exec
-- Link invoice to billing provider
UPDATE invoices
SET
    provider = $3,
    provider_invoice_id = $4,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: MarkInvoiceViewed :exec
-- Mark invoice as viewed (first view only)
UPDATE invoices
SET
    viewed_at = COALESCE(viewed_at, NOW()),
    status = CASE WHEN status = 'sent' THEN 'viewed' ELSE status END,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- =============================================================================
-- INVOICE ITEMS
-- =============================================================================

-- name: CreateInvoiceItem :one
-- Create an invoice line item
INSERT INTO invoice_items (
    tenant_id,
    invoice_id,
    item_type,
    product_sku_id,
    order_id,
    description,
    quantity,
    unit_price_cents,
    total_price_cents
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetInvoiceItems :many
-- Get all items for an invoice
SELECT * FROM invoice_items
WHERE invoice_id = $1
ORDER BY created_at ASC;

-- =============================================================================
-- INVOICE PAYMENTS
-- =============================================================================

-- name: CreateInvoicePayment :one
-- Record a payment against an invoice
-- Note: The update_invoice_balance trigger automatically updates invoice totals
INSERT INTO invoice_payments (
    tenant_id,
    invoice_id,
    payment_id,
    amount_cents,
    payment_method,
    payment_reference,
    notes,
    payment_date
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetInvoicePayments :many
-- Get all payments for an invoice
SELECT * FROM invoice_payments
WHERE invoice_id = $1
ORDER BY payment_date DESC;

-- =============================================================================
-- INVOICE-ORDER LINKING (Consolidated Invoicing)
-- =============================================================================

-- name: CreateInvoiceOrder :one
-- Link an order to an invoice
INSERT INTO invoice_orders (
    tenant_id,
    invoice_id,
    order_id,
    order_number,
    order_total_cents
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInvoiceOrders :many
-- Get all orders linked to an invoice
SELECT
    io.id,
    io.order_id,
    io.order_number,
    io.order_total_cents,
    io.created_at,
    o.status as order_status,
    o.fulfillment_status,
    o.created_at as order_created_at
FROM invoice_orders io
JOIN orders o ON o.id = io.order_id
WHERE io.invoice_id = $1
ORDER BY o.created_at ASC;

-- name: GetInvoiceForOrder :one
-- Get the invoice linked to a specific order (if any)
SELECT i.*
FROM invoices i
JOIN invoice_orders io ON io.invoice_id = i.id
WHERE io.order_id = $1
LIMIT 1;

-- =============================================================================
-- LISTING QUERIES
-- =============================================================================

-- name: ListInvoicesForUser :many
-- List invoices for a customer
SELECT * FROM invoices
WHERE tenant_id = $1
  AND user_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListInvoices :many
-- List all invoices for admin with customer details
SELECT
    i.id,
    i.tenant_id,
    i.invoice_number,
    i.status,
    i.total_cents,
    i.paid_cents,
    i.balance_cents,
    i.currency,
    i.payment_terms,
    i.due_date,
    i.created_at,
    i.sent_at,
    i.paid_at,
    i.is_proforma,
    u.email as customer_email,
    u.company_name,
    CONCAT(u.first_name, ' ', u.last_name) as customer_name
FROM invoices i
JOIN users u ON u.id = i.user_id
WHERE i.tenant_id = $1
ORDER BY i.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListInvoicesByStatus :many
-- List invoices filtered by status
SELECT
    i.id,
    i.tenant_id,
    i.invoice_number,
    i.status,
    i.total_cents,
    i.paid_cents,
    i.balance_cents,
    i.currency,
    i.due_date,
    i.created_at,
    u.email as customer_email,
    u.company_name
FROM invoices i
JOIN users u ON u.id = i.user_id
WHERE i.tenant_id = $1
  AND i.status = $2
ORDER BY i.due_date ASC;

-- name: ListOverdueInvoices :many
-- List invoices that are past due
SELECT
    i.id,
    i.tenant_id,
    i.invoice_number,
    i.status,
    i.total_cents,
    i.balance_cents,
    i.currency,
    i.due_date,
    i.created_at,
    u.id as user_id,
    u.email as customer_email,
    u.company_name,
    u.email_invoices
FROM invoices i
JOIN users u ON u.id = i.user_id
WHERE i.tenant_id = $1
  AND i.status NOT IN ('paid', 'cancelled', 'void')
  AND i.due_date < CURRENT_DATE
ORDER BY i.due_date ASC;

-- name: CountInvoices :one
-- Count invoices for pagination
SELECT COUNT(*)
FROM invoices
WHERE tenant_id = $1;

-- =============================================================================
-- CONSOLIDATED BILLING QUERIES
-- =============================================================================

-- name: GetUninvoicedOrdersForUser :many
-- Get orders that haven't been invoiced yet for a user
-- Used for generating consolidated invoices
SELECT o.*
FROM orders o
LEFT JOIN invoice_orders io ON io.order_id = o.id
WHERE o.tenant_id = $1
  AND o.user_id = $2
  AND o.order_type = 'wholesale'
  AND o.status IN ('paid', 'processing', 'shipped', 'delivered')
  AND io.id IS NULL
ORDER BY o.created_at ASC;

-- name: GetUninvoicedOrdersInPeriod :many
-- Get uninvoiced orders within a billing period
SELECT o.*
FROM orders o
LEFT JOIN invoice_orders io ON io.order_id = o.id
WHERE o.tenant_id = $1
  AND o.user_id = $2
  AND o.order_type = 'wholesale'
  AND o.status IN ('paid', 'processing', 'shipped', 'delivered')
  AND o.created_at >= $3
  AND o.created_at < $4
  AND io.id IS NULL
ORDER BY o.created_at ASC;

-- =============================================================================
-- STATISTICS
-- =============================================================================

-- name: GetInvoiceStats :one
-- Get invoice statistics for dashboard
SELECT
    COUNT(*) as total_invoices,
    COUNT(*) FILTER (WHERE status = 'sent') as sent_invoices,
    COUNT(*) FILTER (WHERE status = 'overdue') as overdue_invoices,
    COUNT(*) FILTER (WHERE status = 'paid') as paid_invoices,
    COALESCE(SUM(total_cents), 0) as total_invoiced_cents,
    COALESCE(SUM(balance_cents), 0) as total_outstanding_cents
FROM invoices
WHERE tenant_id = $1
  AND status NOT IN ('cancelled', 'void');

-- name: GetInvoiceWithDetails :one
-- Get complete invoice with customer and payment terms details
SELECT
    i.*,
    u.email as customer_email,
    u.first_name as customer_first_name,
    u.last_name as customer_last_name,
    u.company_name,
    u.phone as customer_phone,
    pt.name as payment_terms_name,
    pt.days as payment_terms_days,
    ba.full_name as billing_name,
    ba.company as billing_company,
    ba.address_line1 as billing_address_line1,
    ba.address_line2 as billing_address_line2,
    ba.city as billing_city,
    ba.state as billing_state,
    ba.postal_code as billing_postal_code,
    ba.country as billing_country
FROM invoices i
JOIN users u ON u.id = i.user_id
LEFT JOIN payment_terms pt ON pt.id = i.payment_terms_id
LEFT JOIN addresses ba ON ba.id = i.billing_address_id
WHERE i.tenant_id = $1
  AND i.id = $2
LIMIT 1;

-- name: GenerateInvoiceNumber :one
-- Generate next invoice number for a tenant
-- Format: INV-YYYYMM-XXXX (e.g., INV-202412-0001)
SELECT 'INV-' || TO_CHAR(NOW(), 'YYYYMM') || '-' ||
       LPAD((COALESCE(MAX(
           CASE WHEN invoice_number LIKE 'INV-' || TO_CHAR(NOW(), 'YYYYMM') || '-%'
                THEN CAST(SUBSTRING(invoice_number FROM 13) AS INTEGER)
                ELSE 0
           END
       ), 0) + 1)::TEXT, 4, '0') as next_invoice_number
FROM invoices
WHERE tenant_id = $1;
