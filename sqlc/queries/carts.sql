-- name: CreateCart :one
-- Create a new cart for a session
INSERT INTO carts (
    tenant_id,
    session_id,
    status,
    last_activity_at
) VALUES (
    $1, $2, 'active', NOW()
) RETURNING *;

-- name: GetCartBySessionID :one
-- Get active cart for a session
SELECT
    id,
    tenant_id,
    user_id,
    session_id,
    status,
    notes,
    metadata,
    last_activity_at,
    converted_to_order_id,
    expires_at,
    created_at,
    updated_at
FROM carts
WHERE session_id = $1
  AND status = 'active'
LIMIT 1;

-- name: GetCartByID :one
-- Get cart by ID
SELECT
    id,
    tenant_id,
    user_id,
    session_id,
    status,
    notes,
    metadata,
    last_activity_at,
    converted_to_order_id,
    expires_at,
    created_at,
    updated_at
FROM carts
WHERE id = $1
LIMIT 1;

-- name: AddCartItem :one
-- Add an item to cart (or update quantity if exists)
INSERT INTO cart_items (
    tenant_id,
    cart_id,
    product_sku_id,
    quantity,
    unit_price_cents
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (cart_id, product_sku_id)
DO UPDATE SET
    quantity = cart_items.quantity + EXCLUDED.quantity,
    updated_at = NOW()
RETURNING *;

-- name: UpdateCartItemQuantity :exec
-- Update quantity of a cart item
UPDATE cart_items
SET
    quantity = $3,
    updated_at = NOW()
WHERE cart_id = $1
  AND product_sku_id = $2;

-- name: RemoveCartItem :exec
-- Remove an item from cart
DELETE FROM cart_items
WHERE cart_id = $1
  AND product_sku_id = $2;

-- name: GetCartItems :many
-- Get all items in a cart with product details
SELECT
    ci.id,
    ci.cart_id,
    ci.product_sku_id,
    ci.quantity,
    ci.unit_price_cents,
    ci.created_at,
    ci.updated_at,
    p.id as product_id,
    p.name as product_name,
    p.slug as product_slug,
    ps.sku,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    ps.inventory_quantity,
    pi.url as image_url,
    pi.alt_text as image_alt
FROM cart_items ci
INNER JOIN product_skus ps ON ps.id = ci.product_sku_id
INNER JOIN products p ON p.id = ps.product_id
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE ci.cart_id = $1
ORDER BY ci.created_at ASC;

-- name: ClearCart :exec
-- Remove all items from a cart
DELETE FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItemCount :one
-- Get total number of items in cart
SELECT COALESCE(SUM(quantity), 0)::integer as item_count
FROM cart_items
WHERE cart_id = $1;
