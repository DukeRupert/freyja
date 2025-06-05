-- internal/database/queries/orders.sql

-- name: GetOrder :one
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: GetOrderByStripeSessionID :one
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
WHERE stripe_session_id = $1;

-- name: GetOrderByStripePaymentIntentID :one
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
WHERE stripe_payment_intent_id = $1;

-- name: CreateOrder :one
INSERT INTO orders (
    customer_id, status, total, stripe_session_id, stripe_payment_intent_id
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at;

-- name: UpdateOrderStatus :one
UPDATE orders
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at;

-- name: GetOrderItems :many
SELECT id, order_id, product_id, name, quantity, price, created_at
FROM order_items
WHERE order_id = $1
ORDER BY created_at ASC;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id, product_id, name, quantity, price
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, order_id, product_id, name, quantity, price, created_at;

-- name: GetOrdersByCustomerID :many
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCustomerOrderCount :one
SELECT COUNT(*) 
FROM orders
WHERE customer_id = $1;

-- name: GetOrderWithItems :many
SELECT sqlc.embed(o), sqlc.embed(oi)
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
WHERE o.id = $1;

-- name: GetRecentOrders :many
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrdersByStatus :many
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, created_at, updated_at
FROM orders
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetTotalOrderCount :one
SELECT COUNT(*)::integer as total_orders
FROM orders;

-- name: GetTotalRevenue :one
SELECT COALESCE(SUM(total), 0)::integer as total_revenue
FROM orders
WHERE status IN ('confirmed', 'processing', 'shipped', 'delivered');

-- name: GetOrderCountByStatus :many
SELECT status, COUNT(*)::integer as count
FROM orders
GROUP BY status
ORDER BY count DESC;