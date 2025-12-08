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
    customer_notes,
    subscription_id,
    customer_po_number,
    requested_delivery_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
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
-- Retrieves all line items for a specific order with product images
SELECT
    oi.id,
    oi.tenant_id,
    oi.order_id,
    oi.product_sku_id,
    oi.product_name,
    oi.sku,
    oi.variant_description,
    oi.quantity,
    oi.unit_price_cents,
    oi.total_price_cents,
    oi.fulfillment_status,
    oi.metadata,
    oi.created_at,
    oi.updated_at,
    oi.quantity_dispatched,
    pi.url as image_url
FROM order_items oi
LEFT JOIN product_skus ps ON ps.id = oi.product_sku_id
LEFT JOIN products p ON p.id = ps.product_id
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE oi.order_id = $1
ORDER BY oi.created_at ASC;

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

-- name: CountOrdersForUser :one
-- Count orders for a user (for account dashboard)
SELECT COUNT(*) as order_count
FROM orders
WHERE tenant_id = $1
  AND user_id = $2;

-- name: ListOrdersForUser :many
-- List orders for a user (storefront order history)
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
    o.subscription_id,
    o.created_at,
    o.updated_at,
    p.status as payment_status,
    s.tracking_number,
    s.carrier,
    s.status as shipment_status,
    s.shipped_at
FROM orders o
LEFT JOIN payments p ON p.id = o.payment_id
LEFT JOIN LATERAL (
    SELECT tracking_number, carrier, status, shipped_at
    FROM shipments
    WHERE order_id = o.id
    ORDER BY created_at DESC
    LIMIT 1
) s ON true
WHERE o.tenant_id = $1
  AND o.user_id = $2
ORDER BY o.created_at DESC
LIMIT $3 OFFSET $4;

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

-- Checkout queries

-- name: GetTenantWarehouseAddress :one
-- Get the primary warehouse address for a tenant (for shipping origin calculations)
-- Used by CheckoutService.GetShippingRates to determine shipping origin
SELECT
    id,
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
FROM addresses
WHERE tenant_id = $1
  AND address_type = 'warehouse'
LIMIT 1;

-- name: ListOrdersBySubscription :many
-- Get all orders for a specific subscription
-- Used by subscription detail page to show order history
SELECT
    o.id,
    o.tenant_id,
    o.order_number,
    o.order_type,
    o.status,
    o.total_cents,
    o.currency,
    o.fulfillment_status,
    o.created_at,
    p.status as payment_status
FROM orders o
LEFT JOIN payments p ON p.id = o.payment_id
WHERE o.tenant_id = $1
  AND o.subscription_id = $2
ORDER BY o.created_at DESC;

-- =============================================================================
-- WHOLESALE ORDER QUERIES
-- =============================================================================

-- name: ListWholesaleOrders :many
-- List wholesale orders with customer details
SELECT
    o.id,
    o.tenant_id,
    o.order_number,
    o.order_type,
    o.status,
    o.fulfillment_status,
    o.total_cents,
    o.currency,
    o.customer_po_number,
    o.requested_delivery_date,
    o.created_at,
    u.email as customer_email,
    u.company_name,
    CONCAT(u.first_name, ' ', u.last_name) as customer_name
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.tenant_id = $1
  AND o.order_type = 'wholesale'
ORDER BY o.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetOrderWithWholesaleDetails :one
-- Get order with wholesale-specific fields
SELECT
    o.*,
    u.email as customer_email,
    u.first_name as customer_first_name,
    u.last_name as customer_last_name,
    u.company_name,
    u.payment_terms_id,
    pt.name as payment_terms_name,
    pt.days as payment_terms_days
FROM orders o
JOIN users u ON u.id = o.user_id
LEFT JOIN payment_terms pt ON pt.id = u.payment_terms_id
WHERE o.tenant_id = $1
  AND o.id = $2
LIMIT 1;

-- =============================================================================
-- PARTIAL FULFILLMENT QUERIES
-- =============================================================================

-- name: UpdateOrderItemDispatchedQuantity :exec
-- Update the dispatched quantity for an order item
UPDATE order_items
SET
    quantity_dispatched = quantity_dispatched + $3,
    fulfillment_status = CASE
        WHEN quantity_dispatched + $3 >= quantity THEN 'fulfilled'
        ELSE fulfillment_status
    END,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: GetOrderItemsWithFulfillment :many
-- Get order items with fulfillment status for partial shipment display
SELECT
    oi.id,
    oi.order_id,
    oi.product_sku_id,
    oi.product_name,
    oi.sku,
    oi.variant_description,
    oi.quantity,
    oi.quantity_dispatched,
    oi.unit_price_cents,
    oi.total_price_cents,
    oi.fulfillment_status,
    (oi.quantity - oi.quantity_dispatched) as quantity_remaining
FROM order_items oi
WHERE oi.order_id = $1
ORDER BY oi.created_at ASC;

-- name: GetUnfulfilledOrderItems :many
-- Get order items that still need to be shipped
SELECT
    oi.id,
    oi.order_id,
    oi.product_sku_id,
    oi.product_name,
    oi.sku,
    oi.variant_description,
    oi.quantity,
    oi.quantity_dispatched,
    (oi.quantity - oi.quantity_dispatched) as quantity_remaining
FROM order_items oi
WHERE oi.order_id = $1
  AND oi.quantity_dispatched < oi.quantity
ORDER BY oi.created_at ASC;

-- name: RecalculateOrderFulfillmentStatus :exec
-- Update order fulfillment status based on item statuses
UPDATE orders
SET
    fulfillment_status = (
        SELECT CASE
            WHEN COUNT(*) FILTER (WHERE oi.quantity_dispatched < oi.quantity) = 0 THEN 'fulfilled'
            WHEN COUNT(*) FILTER (WHERE oi.quantity_dispatched > 0) > 0 THEN 'partial'
            ELSE 'unfulfilled'
        END
        FROM order_items oi
        WHERE oi.order_id = orders.id
    ),
    updated_at = NOW()
WHERE orders.tenant_id = $1
  AND orders.id = $2;

-- =============================================================================
-- SHIPMENT ITEM QUERIES
-- =============================================================================

-- name: CreateShipmentItem :one
-- Create a shipment line item for partial fulfillment
INSERT INTO shipment_items (
    tenant_id,
    shipment_id,
    order_item_id,
    quantity
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetShipmentItems :many
-- Get items in a shipment
SELECT
    si.id,
    si.shipment_id,
    si.order_item_id,
    si.quantity,
    oi.product_name,
    oi.sku,
    oi.variant_description
FROM shipment_items si
JOIN order_items oi ON oi.id = si.order_item_id
WHERE si.shipment_id = $1
ORDER BY oi.created_at ASC;

-- name: GetShipmentHistory :many
-- Get shipment history for an order item
SELECT
    s.id as shipment_id,
    s.shipment_number,
    s.carrier,
    s.tracking_number,
    s.status,
    s.shipped_at,
    si.quantity
FROM shipment_items si
JOIN shipments s ON s.id = si.shipment_id
WHERE si.order_item_id = $1
ORDER BY s.created_at DESC;
