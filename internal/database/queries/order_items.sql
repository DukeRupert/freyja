-- internal/database/queries/order_items.sql - Updated queries

-- name: GetOrderItem :one
SELECT id, order_id, product_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE id = $1;

-- name: GetOrderItems :many
SELECT oi.id, oi.order_id, oi.product_id, oi.name, oi.quantity, oi.price, 
       oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
       p.stock as product_stock, p.active as product_active
FROM order_items oi
LEFT JOIN products p ON oi.product_id = p.id
WHERE oi.order_id = $1
ORDER BY oi.created_at ASC;

-- name: CreateOrderItem :one
INSERT INTO order_items (
  order_id, product_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, order_id, product_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: GetOrderItemsByPurchaseType :many
SELECT id, order_id, product_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE order_id = $1 AND purchase_type = $2
ORDER BY created_at ASC;

-- name: GetSubscriptionOrderItems :many
SELECT id, order_id, product_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE order_id = $1 AND purchase_type = 'subscription'
ORDER BY created_at ASC;

-- name: GetOrderItemsByProduct :many
SELECT oi.id, oi.order_id, oi.product_id, oi.name, oi.quantity, oi.price, 
       oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
       o.customer_id, o.status as order_status
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
WHERE oi.product_id = $1
ORDER BY oi.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetOrderItemStats :one
SELECT 
  COUNT(*) as total_items,
  COALESCE(SUM(quantity), 0) as total_quantity,
  COALESCE(SUM(quantity * price), 0) as total_value,
  COUNT(CASE WHEN purchase_type = 'one_time' THEN 1 END) as one_time_items,
  COUNT(CASE WHEN purchase_type = 'subscription' THEN 1 END) as subscription_items
FROM order_items
WHERE order_id = $1;