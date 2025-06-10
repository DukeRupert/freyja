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
