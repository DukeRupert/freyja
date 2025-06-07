-- internal/database/queries/products.sql

-- name: GetProduct :one
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE id = $1;

-- name: GetProductByName :one
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE name = $1;

-- name: GetProductByStripeProductID :one
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE stripe_product_id = $1;

-- name: ListProducts :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = true
ORDER BY name;

-- name: ListAllProducts :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListProductsByStatus :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: SearchProducts :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = true
  AND (name ILIKE $1 OR description ILIKE $1)
ORDER BY
  CASE WHEN name ILIKE $1 THEN 1 ELSE 2 END,
  name;

-- name: GetProductsInStock :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = true AND stock > 0
ORDER BY stock DESC;

-- name: GetLowStockProducts :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = true AND stock <= $1
ORDER BY stock ASC;

-- name: GetProductsWithoutStripeSync :many
SELECT id, name, description, price, stock, active, created_at, updated_at,
       stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
       stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id
FROM products
WHERE active = true AND stripe_product_id IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateProduct :one
INSERT INTO products (
  name, description, price, stock, active
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: UpdateProduct :one
UPDATE products
SET
  name = $2,
  description = $3,
  price = $4,
  stock = $5,
  active = $6,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: UpdateProductStock :one
UPDATE products
SET
  stock = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: UpdateProductPrice :one
UPDATE products
SET
  price = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: UpdateProductStripeProductID :one
UPDATE products
SET
  stripe_product_id = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: UpdateProductStripePrices :one
UPDATE products
SET
  stripe_price_onetime_id = COALESCE($2, stripe_price_onetime_id),
  stripe_price_14day_id = COALESCE($3, stripe_price_14day_id),
  stripe_price_21day_id = COALESCE($4, stripe_price_21day_id),
  stripe_price_30day_id = COALESCE($5, stripe_price_30day_id),
  stripe_price_60day_id = COALESCE($6, stripe_price_60day_id),
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: IncrementProductStock :one
UPDATE products
SET
  stock = stock + $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: DecrementProductStock :one
UPDATE products
SET
  stock = stock - $2,
  updated_at = NOW()
WHERE id = $1 AND stock >= $2
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: ActivateProduct :one
UPDATE products
SET
  active = true,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: DeactivateProduct :one
UPDATE products
SET
  active = false,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, active, created_at, updated_at,
          stripe_product_id, stripe_price_onetime_id, stripe_price_14day_id,
          stripe_price_21day_id, stripe_price_30day_id, stripe_price_60day_id;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

-- name: GetProductCount :one
SELECT COUNT(*) FROM products WHERE active = $1;

-- name: GetTotalProductValue :one
SELECT COALESCE(SUM(price * stock), 0)::integer as total_value
FROM products
WHERE active = true;
