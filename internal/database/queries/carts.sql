-- internal/database/queries/carts.sql

-- name: GetCart :one
SELECT id, customer_id, session_id, created_at, updated_at
FROM carts
WHERE id = $1;

-- name: GetCartByCustomerID :one
SELECT id, customer_id, session_id, created_at, updated_at
FROM carts
WHERE customer_id = $1;

-- name: GetCartBySessionID :one
SELECT id, customer_id, session_id, created_at, updated_at
FROM carts
WHERE session_id = $1;

-- name: CreateCart :one
INSERT INTO carts (
  customer_id, session_id
) VALUES (
  $1, $2
)
RETURNING id, customer_id, session_id, created_at, updated_at;

-- name: UpdateCartTimestamp :one
UPDATE carts
SET updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, session_id, created_at, updated_at;

-- name: DeleteCart :exec
DELETE FROM carts
WHERE id = $1;

-- name: ClearCartItems :exec
DELETE FROM cart_items
WHERE cart_id = $1;

-- Cart Items queries

-- name: GetCartItem :one
SELECT id, cart_id, product_id, quantity, price, created_at
FROM cart_items
WHERE id = $1;

-- name: GetCartItems :many
SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.price, ci.created_at,
       p.name as product_name, p.description as product_description, p.stock as product_stock
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
WHERE ci.cart_id = $1
ORDER BY ci.created_at ASC;

-- name: GetCartItemByProductID :one
SELECT id, cart_id, product_id, quantity, price, created_at
FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: CreateCartItem :one
INSERT INTO cart_items (
  cart_id, product_id, quantity, price
) VALUES (
  $1, $2, $3, $4
)
RETURNING id, cart_id, product_id, quantity, price, created_at;

-- name: UpdateCartItem :one
UPDATE cart_items
SET
  quantity = $2,
  price = $3
WHERE id = $1
RETURNING id, cart_id, product_id, quantity, price, created_at;

-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET quantity = $2
WHERE id = $1
RETURNING id, cart_id, product_id, quantity, price, created_at;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE id = $1;

-- name: DeleteCartItemByProductID :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: GetCartTotal :one
SELECT COALESCE(SUM(quantity * price), 0)::integer as total
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItemCount :one
SELECT COALESCE(SUM(quantity), 0)::integer as item_count
FROM cart_items
WHERE cart_id = $1;
