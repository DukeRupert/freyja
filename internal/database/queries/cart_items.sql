-- internal/database/queries/cart_items.sql
-- Updated for product variants system

-- Basic cart item queries

-- name: GetCartItem :one
SELECT id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE id = $1;

-- name: GetCartItems :many
SELECT 
    ci.id, ci.cart_id, ci.product_variant_id, ci.quantity, ci.price, 
    ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
    pv.name as variant_name,
    pv.stock as variant_stock,
    pv.active as variant_active,
    pv.options_display,
    p.id as product_id,
    p.name as product_name, 
    p.description as product_description,
    p.active as product_active
FROM cart_items ci
JOIN product_variants pv ON ci.product_variant_id = pv.id AND pv.archived_at IS NULL
JOIN products p ON pv.product_id = p.id
WHERE ci.cart_id = $1
ORDER BY ci.created_at ASC;

-- name: GetCartItemsWithOptions :many
SELECT 
    ci.id, ci.cart_id, ci.product_variant_id, ci.quantity, ci.price, 
    ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
    pv.name as variant_name,
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
    )::text as variant_options
FROM cart_items ci
JOIN product_variants pv ON ci.product_variant_id = pv.id AND pv.archived_at IS NULL
JOIN products p ON pv.product_id = p.id
LEFT JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
LEFT JOIN product_options po ON pvo.product_option_id = po.id
LEFT JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE ci.cart_id = $1
GROUP BY ci.id, ci.cart_id, ci.product_variant_id, ci.quantity, ci.price, 
         ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
         pv.name, pv.stock, pv.active, pv.options_display,
         p.id, p.name, p.description, p.active
ORDER BY ci.created_at ASC;

-- name: GetCartItemsByVariant :many
SELECT id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 AND product_variant_id = $2;

-- name: GetCartItemByVariantID :one
SELECT id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 AND product_variant_id = $2
LIMIT 1;

-- name: GetCartItemByVariantAndType :one
SELECT id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 
  AND product_variant_id = $2 
  AND purchase_type = $3 
  AND (subscription_interval = $4 OR ($4 IS NULL AND subscription_interval IS NULL));

-- Cart item management

-- name: CreateCartItem :one
INSERT INTO cart_items (
  cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: UpdateCartItem :one
UPDATE cart_items
SET
  quantity = $2,
  price = $3,
  stripe_price_id = $4
WHERE id = $1
RETURNING id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET quantity = $2
WHERE id = $1
RETURNING id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: IncrementCartItemQuantity :one
UPDATE cart_items
SET quantity = quantity + $2
WHERE id = $1
RETURNING id, cart_id, product_variant_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE id = $1;

-- name: DeleteCartItemByVariantID :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_variant_id = $2;

-- name: DeleteCartItemByVariantAndType :exec
DELETE FROM cart_items
WHERE cart_id = $1 
  AND product_variant_id = $2 
  AND purchase_type = $3 
  AND (subscription_interval = $4 OR ($4 IS NULL AND subscription_interval IS NULL));

-- Cart summary and analytics

-- name: GetCartTotal :one
SELECT COALESCE(SUM(quantity * price), 0)::integer as total
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItemCount :one
SELECT COALESCE(SUM(quantity), 0)::integer as item_count
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartTotalByPurchaseType :one
SELECT COALESCE(SUM(quantity * price), 0)::integer as total
FROM cart_items
WHERE cart_id = $1 AND purchase_type = $2;

-- name: GetCartItemsByPurchaseType :many
SELECT 
    ci.id, ci.cart_id, ci.product_variant_id, ci.quantity, ci.price, 
    ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
    pv.name as variant_name,
    pv.stock as variant_stock,
    pv.active as variant_active,
    pv.options_display,
    p.id as product_id,
    p.name as product_name, 
    p.description as product_description,
    p.active as product_active
FROM cart_items ci
JOIN product_variants pv ON ci.product_variant_id = pv.id AND pv.archived_at IS NULL
JOIN products p ON pv.product_id = p.id
WHERE ci.cart_id = $1 AND ci.purchase_type = $2
ORDER BY ci.created_at ASC;

-- name: GetCartSubscriptionSummary :many
SELECT 
    ci.subscription_interval,
    COUNT(*) as item_count,
    SUM(ci.quantity) as total_quantity,
    SUM(ci.quantity * ci.price) as total_amount
FROM cart_items ci
WHERE ci.cart_id = $1 AND ci.purchase_type = 'subscription'
GROUP BY ci.subscription_interval
ORDER BY ci.subscription_interval;

-- Cart validation queries

-- name: ValidateCartItems :many
SELECT 
    ci.id as cart_item_id,
    ci.product_variant_id,
    ci.quantity as requested_quantity,
    pv.stock as available_stock,
    pv.active as variant_active,
    p.active as product_active,
    CASE 
        WHEN NOT p.active THEN 'product_inactive'
        WHEN NOT pv.active THEN 'variant_inactive'
        WHEN pv.archived_at IS NOT NULL THEN 'variant_archived'
        WHEN ci.quantity > pv.stock THEN 'insufficient_stock'
        ELSE 'valid'
    END as validation_status
FROM cart_items ci
JOIN product_variants pv ON ci.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
WHERE ci.cart_id = $1;

-- name: GetInvalidCartItems :many
SELECT 
    ci.id, ci.cart_id, ci.product_variant_id, ci.quantity, ci.price, 
    ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
    pv.name as variant_name,
    pv.stock as variant_stock,
    pv.active as variant_active,
    p.name as product_name,
    CASE 
        WHEN NOT p.active THEN 'product_inactive'
        WHEN NOT pv.active THEN 'variant_inactive'
        WHEN pv.archived_at IS NOT NULL THEN 'variant_archived'
        WHEN ci.quantity > pv.stock THEN 'insufficient_stock'
        ELSE 'valid'
    END as issue_type
FROM cart_items ci
JOIN product_variants pv ON ci.product_variant_id = pv.id
JOIN products p ON pv.product_id = p.id
WHERE ci.cart_id = $1
  AND (NOT p.active 
       OR NOT pv.active 
       OR pv.archived_at IS NOT NULL 
       OR ci.quantity > pv.stock);

-- name: CheckVariantAvailability :one
SELECT 
    pv.id,
    pv.stock,
    pv.active,
    p.active as product_active,
    CASE 
        WHEN NOT p.active THEN false
        WHEN NOT pv.active THEN false
        WHEN pv.archived_at IS NOT NULL THEN false
        WHEN pv.stock < $2 THEN false
        ELSE true
    END as is_available
FROM product_variants pv
JOIN products p ON pv.product_id = p.id
WHERE pv.id = $1;

-- Cart cleanup and maintenance

-- name: RemoveUnavailableCartItems :exec
DELETE FROM cart_items 
WHERE cart_id = $1 
  AND product_variant_id IN (
    SELECT pv.id 
    FROM product_variants pv 
    JOIN products p ON pv.product_id = p.id 
    WHERE NOT p.active 
       OR NOT pv.active 
       OR pv.archived_at IS NOT NULL
  );

-- name: UpdateCartItemPrices :exec
UPDATE cart_items 
SET price = pv.price
FROM product_variants pv
WHERE cart_items.product_variant_id = pv.id 
  AND cart_items.cart_id = $1
  AND pv.archived_at IS NULL;

-- Analytics queries

-- name: GetCartAbandonmentData :many
SELECT 
    c.id as cart_id,
    c.created_at as cart_created,
    c.updated_at as cart_updated,
    COUNT(ci.id) as item_count,
    SUM(ci.quantity * ci.price) as cart_value,
    EXTRACT(EPOCH FROM (NOW() - c.updated_at))/3600 as hours_since_update
FROM carts c
LEFT JOIN cart_items ci ON c.id = ci.cart_id
WHERE c.updated_at < NOW() - INTERVAL '1 hour'
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.cart_id = c.id)
GROUP BY c.id, c.created_at, c.updated_at
HAVING COUNT(ci.id) > 0
ORDER BY cart_value DESC
LIMIT $1 OFFSET $2;