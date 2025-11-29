-- name: GetOrderByPaymentIntentID :one
-- Idempotency check: Returns existing order if payment intent was already processed
-- This prevents duplicate order creation from webhook retries
SELECT o.* FROM orders o
INNER JOIN payments p ON p.id = o.payment_id AND p.tenant_id = o.tenant_id
WHERE o.tenant_id = $1
  AND p.provider_payment_id = $2
LIMIT 1;

-- name: CreateOrder :one
-- Creates a new order record with all required fields
-- Returns the complete order with generated ID and timestamps
INSERT INTO orders (
    tenant_id,
    cart_id,
    user_id,
    order_number,
    order_type,
    status,
    subtotal_cents,
    shipping_cents,
    tax_cents,
    total_cents,
    currency,
    shipping_address_id,
    billing_address_id,
    customer_notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: CreateOrderItem :one
-- Creates an order line item linked to a specific order
-- Captures product state at time of purchase
INSERT INTO order_items (
    tenant_id,
    order_id,
    product_sku_id,
    product_name,
    sku,
    variant_description,
    quantity,
    unit_price_cents,
    total_price_cents
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: CreateAddress :one
-- Creates a new address record for shipping or billing
-- Returns complete address with generated ID
INSERT INTO addresses (
    tenant_id,
    address_type,
    full_name,
    company,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    country,
    phone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: CreateBillingCustomer :one
-- Creates a billing customer record (links to Stripe customer)
-- This is for tracking payment method details
INSERT INTO billing_customers (
    tenant_id,
    user_id,
    provider,
    provider_customer_id
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (user_id, provider) DO UPDATE
SET updated_at = NOW()
RETURNING *;

-- name: CreatePayment :one
-- Records a payment transaction linked to an order
-- Includes Stripe payment intent ID for reconciliation
INSERT INTO payments (
    tenant_id,
    billing_customer_id,
    provider,
    provider_payment_id,
    amount_cents,
    currency,
    status,
    payment_method_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: DecrementSKUStock :exec
-- Decrements inventory for a SKU after order placement
-- Uses optimistic locking to prevent overselling
UPDATE product_skus
SET inventory_quantity = inventory_quantity - $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
  AND inventory_quantity >= $3;  -- Ensures sufficient stock

-- name: UpdateCartStatus :exec
-- Marks cart as converted to order
-- Prevents duplicate order creation from same cart
UPDATE carts
SET status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: GetOrder :one
-- Retrieves a single order by ID with tenant scoping
SELECT * FROM orders
WHERE tenant_id = $1
  AND id = $2
LIMIT 1;

-- name: GetOrderByNumber :one
-- Retrieves a single order by order number with tenant scoping
-- Order numbers are unique per tenant
SELECT * FROM orders
WHERE tenant_id = $1
  AND order_number = $2
LIMIT 1;

-- name: GetOrderItems :many
-- Retrieves all line items for a specific order
SELECT * FROM order_items
WHERE order_id = $1
ORDER BY created_at ASC;

-- name: GetAddressByID :one
-- Retrieves a single address by ID
SELECT * FROM addresses
WHERE id = $1
LIMIT 1;

-- name: GetPaymentByID :one
-- Retrieves a single payment by ID
SELECT * FROM payments
WHERE id = $1
LIMIT 1;

-- name: UpdateOrderPaymentID :exec
-- Links a payment to an order after both are created
UPDATE orders
SET payment_id = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- Admin queries

-- name: ListOrders :many
-- List all orders for admin with pagination
SELECT
    o.id,
    o.tenant_id,
    o.order_number,
    o.order_type,
    o.status,
    o.total_cents,
    o.currency,
    o.created_at,
    o.updated_at,
    u.email as customer_email,
    CONCAT(u.first_name, ' ', u.last_name) as customer_name,
    sa.address_line1 as shipping_address_line1,
    sa.city as shipping_city,
    sa.state as shipping_state
FROM orders o
LEFT JOIN users u ON u.id = o.user_id
LEFT JOIN addresses sa ON sa.id = o.shipping_address_id
WHERE o.tenant_id = $1
ORDER BY o.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrders :one
-- Count total orders for pagination
SELECT COUNT(*)
FROM orders
WHERE tenant_id = $1;

-- name: ListOrdersByStatus :many
-- List orders filtered by status
SELECT
    o.id,
    o.tenant_id,
    o.order_number,
    o.order_type,
    o.status,
    o.total_cents,
    o.currency,
    o.created_at,
    u.email as customer_email,
    CONCAT(u.first_name, ' ', u.last_name) as customer_name
FROM orders o
LEFT JOIN users u ON u.id = o.user_id
WHERE o.tenant_id = $1
  AND o.status = $2
ORDER BY o.created_at DESC;

-- name: UpdateOrderStatus :exec
-- Update order status
UPDATE orders
SET
    status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: UpdateOrderFulfillmentStatus :exec
-- Update order fulfillment status
UPDATE orders
SET
    fulfillment_status = $3,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: GetOrderStats :one
-- Get order statistics for dashboard
SELECT
    COUNT(*) as total_orders,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_orders,
    COUNT(*) FILTER (WHERE status = 'processing') as processing_orders,
    COUNT(*) FILTER (WHERE status = 'shipped') as shipped_orders,
    COALESCE(SUM(total_cents), 0) as total_revenue_cents
FROM orders
WHERE tenant_id = $1
  AND created_at >= $2;

-- name: GetOrderWithDetails :one
-- Get complete order details including addresses and payment info
SELECT
    o.id,
    o.tenant_id,
    o.order_number,
    o.order_type,
    o.status,
    o.fulfillment_status,
    o.subtotal_cents,
    o.shipping_cents,
    o.tax_cents,
    o.total_cents,
    o.currency,
    o.customer_notes,
    o.created_at,
    o.updated_at,
    u.email as customer_email,
    u.first_name as customer_first_name,
    u.last_name as customer_last_name,
    sa.full_name as shipping_name,
    sa.company as shipping_company,
    sa.address_line1 as shipping_address_line1,
    sa.address_line2 as shipping_address_line2,
    sa.city as shipping_city,
    sa.state as shipping_state,
    sa.postal_code as shipping_postal_code,
    sa.country as shipping_country,
    sa.phone as shipping_phone,
    ba.full_name as billing_name,
    ba.address_line1 as billing_address_line1,
    ba.address_line2 as billing_address_line2,
    ba.city as billing_city,
    ba.state as billing_state,
    ba.postal_code as billing_postal_code,
    ba.country as billing_country,
    p.status as payment_status,
    p.provider_payment_id
FROM orders o
LEFT JOIN users u ON u.id = o.user_id
LEFT JOIN addresses sa ON sa.id = o.shipping_address_id
LEFT JOIN addresses ba ON ba.id = o.billing_address_id
LEFT JOIN payments p ON p.id = o.payment_id
WHERE o.tenant_id = $1
  AND o.id = $2
LIMIT 1;

-- name: CreateShipment :one
-- Create a shipment record for an order
INSERT INTO shipments (
    tenant_id,
    order_id,
    carrier,
    tracking_number,
    shipping_method_id,
    status
) VALUES (
    $1, $2, $3, $4, $5, 'pending'
)
RETURNING *;

-- name: UpdateShipmentStatus :exec
-- Update shipment status
UPDATE shipments
SET
    status = $3,
    shipped_at = CASE WHEN $3 = 'shipped' THEN NOW() ELSE shipped_at END,
    delivered_at = CASE WHEN $3 = 'delivered' THEN NOW() ELSE delivered_at END,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: GetShipmentsByOrderID :many
-- Get all shipments for an order
SELECT * FROM shipments
WHERE order_id = $1
ORDER BY created_at DESC;
