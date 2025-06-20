-- internal/database/queries/order_items.sql
-- Updated for product variants system

-- Basic order item queries

-- name: GetOrderItem :one
SELECT id, order_id, product_variant_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE id = $1;

-- name: GetOrderItems :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    pv.stock as variant_stock, 
    pv.active as variant_active,
    pv.options_display,
    p.id as product_id,
    p.name as product_name,
    p.description as product_description,
    p.active as product_active
FROM order_items oi
LEFT JOIN product_variants pv ON oi.product_variant_id = pv.id
LEFT JOIN products p ON pv.product_id = p.id
WHERE oi.order_id = $1
ORDER BY oi.created_at ASC;

-- name: GetOrderItemsWithOptions :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    pv.stock as variant_stock, 
    pv.active as variant_active,
    pv.options_display,
    p.id as product_id,
    p.name as product_name,
    p.description as product_description,
    p.active as product_active,
    COALESCE(
        json_agg(
            json_build_object(
                'option_key', po.option_key,
                'value', pov.value
            ) ORDER BY po.option_key
        ) FILTER (WHERE po.id IS NOT NULL), 
        '[]'::json
    ) as variant_options
FROM order_items oi
LEFT JOIN product_variants pv ON oi.product_variant_id = pv.id
LEFT JOIN products p ON pv.product_id = p.id
LEFT JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
LEFT JOIN product_options po ON pvo.product_option_id = po.id
LEFT JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE oi.order_id = $1
GROUP BY oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
         oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
         pv.stock, pv.active, pv.options_display,
         p.id, p.name, p.active
ORDER BY oi.created_at ASC;

-- name: CreateOrderItem :one
INSERT INTO order_items (
  order_id, product_variant_id, name, variant_name, quantity, price, purchase_type, subscription_interval, stripe_price_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, order_id, product_variant_id, name, variant_name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- Order item filtering and grouping

-- name: GetOrderItemsByPurchaseType :many
SELECT id, order_id, product_variant_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE order_id = $1 AND purchase_type = $2
ORDER BY created_at ASC;

-- name: GetSubscriptionOrderItems :many
SELECT id, order_id, product_variant_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE order_id = $1 AND purchase_type = 'subscription'
ORDER BY created_at ASC;

-- name: GetOneTimeOrderItems :many
SELECT id, order_id, product_variant_id, name, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM order_items
WHERE order_id = $1 AND purchase_type = 'one_time'
ORDER BY created_at ASC;

-- Order item analytics

-- name: GetOrderItemsByVariant :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    o.customer_id, o.status as order_status, o.created_at as order_date
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
WHERE oi.product_variant_id = $1
ORDER BY oi.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetOrderItemsByProduct :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    o.customer_id, o.status as order_status, o.created_at as order_date,
    pv.name as variant_name, pv.options_display
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
JOIN product_variants pv ON oi.product_variant_id = pv.id
WHERE pv.product_id = $1
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

-- name: GetSubscriptionSummaryByOrder :many
SELECT 
    subscription_interval,
    COUNT(*) as item_count,
    SUM(quantity) as total_quantity,
    SUM(quantity * price) as total_amount
FROM order_items
WHERE order_id = $1 AND purchase_type = 'subscription'
GROUP BY subscription_interval
ORDER BY subscription_interval;

-- Sales analytics queries

-- name: GetVariantSalesStats :many
SELECT 
    pv.id as variant_id,
    pv.name as variant_name,
    pv.options_display,
    p.name as product_name,
    COUNT(oi.id) as order_count,
    SUM(oi.quantity) as total_sold,
    SUM(oi.quantity * oi.price) as total_revenue,
    AVG(oi.price) as avg_price
FROM order_items oi
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
JOIN orders o ON oi.order_id = o.id
WHERE o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY pv.id, pv.name, pv.options_display, p.name
ORDER BY total_sold DESC
LIMIT $3 OFFSET $4;

-- name: GetProductSalesStats :many
SELECT 
    p.id as product_id,
    p.name as product_name,
    COUNT(DISTINCT pv.id) as variant_count,
    COUNT(oi.id) as order_count,
    SUM(oi.quantity) as total_sold,
    SUM(oi.quantity * oi.price) as total_revenue,
    AVG(oi.price) as avg_price
FROM order_items oi
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
JOIN orders o ON oi.order_id = o.id
WHERE o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY p.id, p.name
ORDER BY total_sold DESC
LIMIT $3 OFFSET $4;

-- name: GetTopSellingVariants :many
SELECT 
    pv.id as variant_id,
    pv.name as variant_name,
    pv.options_display,
    p.name as product_name,
    SUM(oi.quantity) as total_sold,
    SUM(oi.quantity * oi.price) as total_revenue
FROM order_items oi
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
JOIN orders o ON oi.order_id = o.id
WHERE o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY pv.id, pv.name, pv.options_display, p.name
ORDER BY total_sold DESC
LIMIT $3 OFFSET $4;

-- Customer purchase behavior

-- name: GetCustomerVariantPurchaseHistory :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.created_at,
    o.status as order_status,
    pv.name as variant_name,
    pv.options_display,
    p.name as product_name
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
WHERE o.customer_id = $1
  AND ($2::timestamp IS NULL OR o.created_at >= $2)
  AND ($3::timestamp IS NULL OR o.created_at <= $3)
ORDER BY oi.created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetCustomerFavoriteVariants :many
SELECT 
    pv.id as variant_id,
    pv.name as variant_name,
    pv.options_display,
    p.name as product_name,
    COUNT(oi.id) as purchase_count,
    SUM(oi.quantity) as total_quantity,
    MAX(o.created_at) as last_purchased
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
WHERE o.customer_id = $1
  AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
GROUP BY pv.id, pv.name, pv.options_display, p.name
ORDER BY purchase_count DESC, total_quantity DESC
LIMIT $2 OFFSET $3;

-- Subscription management

-- name: GetActiveSubscriptionItems :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    o.customer_id, o.status as order_status,
    pv.name as variant_name,
    pv.options_display,
    p.name as product_name
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
JOIN product_variants pv ON oi.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
WHERE oi.purchase_type = 'subscription'
  AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
  AND ($1::int4 IS NULL OR o.customer_id = $1)
ORDER BY oi.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubscriptionsByInterval :many
SELECT 
    subscription_interval,
    COUNT(*) as subscription_count,
    SUM(quantity) as total_quantity,
    SUM(quantity * price) as total_monthly_revenue
FROM order_items oi
JOIN orders o ON oi.order_id = o.id
WHERE oi.purchase_type = 'subscription'
  AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY subscription_interval
ORDER BY total_monthly_revenue DESC;

-- Data integrity and validation

-- name: GetOrderItemsWithMissingVariants :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at
FROM order_items oi
LEFT JOIN product_variants pv ON oi.product_variant_id = pv.id
WHERE pv.id IS NULL
ORDER BY oi.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrderItemsWithArchivedVariants :many
SELECT 
    oi.id, oi.order_id, oi.product_variant_id, oi.name, oi.quantity, oi.price, 
    oi.purchase_type, oi.subscription_interval, oi.stripe_price_id, oi.created_at,
    pv.archived_at
FROM order_items oi
JOIN product_variants pv ON oi.product_variant_id = pv.id
WHERE pv.archived_at IS NOT NULL
ORDER BY oi.created_at DESC
LIMIT $1 OFFSET $2;