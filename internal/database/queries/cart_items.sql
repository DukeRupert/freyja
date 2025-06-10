-- internal/database/queries/cart_items.sql

-- name: GetCartItem :one
SELECT id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE id = $1;

-- name: GetCartItems :many
SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.price, ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
       p.name as product_name, p.description as product_description, p.stock as product_stock
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
WHERE ci.cart_id = $1
ORDER BY ci.created_at ASC;

-- name: GetCartItemsByProduct :many
SELECT id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: GetCartItemByProductID :one
SELECT id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 AND product_id = $2
LIMIT 1;

-- name: GetCartItemByProductAndType :one
SELECT id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at
FROM cart_items
WHERE cart_id = $1 AND product_id = $2 AND purchase_type = $3 AND (subscription_interval = $4 OR ($4 IS NULL AND subscription_interval IS NULL));

-- name: CreateCartItem :one
INSERT INTO cart_items (
  cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: UpdateCartItem :one
UPDATE cart_items
SET
  quantity = $2,
  price = $3,
  stripe_price_id = $4
WHERE id = $1
RETURNING id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET quantity = $2
WHERE id = $1
RETURNING id, cart_id, product_id, quantity, price, purchase_type, subscription_interval, stripe_price_id, created_at;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE id = $1;

-- name: DeleteCartItemByProductID :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: DeleteCartItemByProductAndType :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2 AND purchase_type = $3 AND (subscription_interval = $4 OR ($4 IS NULL AND subscription_interval IS NULL));

-- name: GetCartTotal :one
SELECT COALESCE(SUM(quantity * price), 0)::integer as total
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItemCount :one
SELECT COALESCE(SUM(quantity), 0)::integer as item_count
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItemsByPurchaseType :many
SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.price, ci.purchase_type, ci.subscription_interval, ci.stripe_price_id, ci.created_at,
       p.name as product_name, p.description as product_description, p.stock as product_stock
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
WHERE ci.cart_id = $1 AND ci.purchase_type = $2
ORDER BY ci.created_at ASC;