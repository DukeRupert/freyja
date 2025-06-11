-- internal/database/queries/orders.sql - Updated queries

-- name: GetOrder :one
SELECT *
FROM orders
WHERE id = $1;

-- name: GetOrdersByCustomerID :many
SELECT *
FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetOrderByStripePaymentIntentID :one
SELECT *
FROM orders
WHERE stripe_payment_intent_id = $1;

-- name: GetOrderByStripeChargeID :one
SELECT *
FROM orders
WHERE stripe_charge_id = $1;

-- name: CreateOrder :one
INSERT INTO orders (
  customer_id, status, total, stripe_session_id, stripe_payment_intent_id, stripe_charge_id
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, stripe_charge_id, created_at, updated_at;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, stripe_charge_id, created_at, updated_at;

-- name: UpdateStripeChargeID :one
UPDATE orders
SET stripe_charge_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, stripe_charge_id, created_at, updated_at;

-- name: GetAllOrders :many
SELECT id, customer_id, status, total, stripe_session_id, stripe_payment_intent_id, stripe_charge_id, created_at, updated_at
FROM orders
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrdersByStatus :many
SELECT *
FROM orders
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetOrderCountByStatus :many
SELECT status, COUNT(*)::integer as count
FROM orders
GROUP BY status
ORDER BY count DESC;

-- name: GetOrderStats :one
SELECT 
  COUNT(*) as total_orders,
  COALESCE(SUM(total), 0) as total_revenue,
  COALESCE(AVG(total), 0) as average_order_value,
  COUNT(CASE WHEN status = 'confirmed' THEN 1 END) as confirmed_orders,
  COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_orders,
  COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders
FROM orders;