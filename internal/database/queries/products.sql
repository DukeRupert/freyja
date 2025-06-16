-- internal/database/queries/products.sql
-- Updated for product variants system

-- name: GetProduct :one
SELECT p.id, p.name, p.description, p.active, p.created_at, p.updated_at
FROM products p
WHERE p.id = $1;

-- name: GetProductWithSummary :one
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_id = $1 AND pss.product_active = true;

-- name: GetProductByName :one
SELECT p.id, p.name, p.description, p.active, p.created_at, p.updated_at
FROM products p
WHERE p.name = $1;

-- name: ListProducts :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = true
ORDER BY pss.name;

-- name: ListAllProducts :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
ORDER BY pss.name
LIMIT $1 OFFSET $2;

-- name: ListProductsByStatus :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = $1
ORDER BY pss.name
LIMIT $2 OFFSET $3;

-- name: SearchProducts :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = true
  AND (pss.name ILIKE $1 OR pss.description ILIKE $1)
ORDER BY
  CASE WHEN pss.name ILIKE $1 THEN 1 ELSE 2 END,
  pss.name;

-- name: SearchProductsWithOptions :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = true
  AND (pss.name ILIKE $1 OR pss.description ILIKE $1 OR pss.available_options::text ILIKE $1)
ORDER BY
  CASE WHEN pss.name ILIKE $1 THEN 1 
       WHEN pss.description ILIKE $1 THEN 2 
       ELSE 3 END,
  pss.name;

-- name: GetProductsInStock :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = true AND pss.has_stock = true
ORDER BY pss.total_stock DESC;

-- name: GetLowStockProducts :many
SELECT 
    pss.product_id,
    pss.name,
    pss.description,
    pss.product_active,
    pss.total_stock,
    pss.variants_in_stock,
    pss.total_variants,
    pss.min_price,
    pss.max_price,
    pss.has_stock,
    pss.stock_status,
    pss.available_options,
    pss.last_stock_update
FROM product_stock_summary pss
WHERE pss.product_active = true AND pss.total_stock <= $1
ORDER BY pss.total_stock ASC;

-- name: GetProductsWithoutVariants :many
SELECT p.id, p.name, p.description, p.active, p.created_at, p.updated_at
FROM products p
LEFT JOIN product_variants pv ON p.id = pv.product_id AND pv.archived_at IS NULL
WHERE p.active = true AND pv.id IS NULL
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- Product management queries (admin operations)

-- name: CreateProduct :one
INSERT INTO products (
  name, description, active
) VALUES (
  $1, $2, $3
)
RETURNING id, name, description, active, created_at, updated_at;

-- name: UpdateProduct :one
UPDATE products
SET
  name = COALESCE(NULLIF($2, ''), name),
  description = COALESCE($3, description),
  active = COALESCE($4, active),
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, active, created_at, updated_at;

-- name: ActivateProduct :one
UPDATE products
SET
  active = true,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, active, created_at, updated_at;

-- name: DeactivateProduct :one
UPDATE products
SET
  active = false,
  updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, active, created_at, updated_at;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

-- Product analytics and reporting queries

-- name: GetProductPerformanceStats :many
SELECT 
    p.id,
    p.name,
    pss.total_variants,
    pss.variants_in_stock,
    pss.total_stock,
    pss.min_price,
    pss.max_price,
    COUNT(DISTINCT oi.id) as total_orders,
    COALESCE(SUM(oi.quantity), 0) as total_sold,
    COALESCE(SUM(oi.price * oi.quantity), 0) as total_revenue
FROM products p
LEFT JOIN product_stock_summary pss ON p.id = pss.product_id
LEFT JOIN product_variants pv ON p.id = pv.product_id AND pv.archived_at IS NULL
LEFT JOIN order_items oi ON pv.id = oi.product_variant_id
LEFT JOIN orders o ON oi.order_id = o.id
WHERE p.active = true
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY p.id, p.name, pss.total_variants, pss.variants_in_stock, 
         pss.total_stock, pss.min_price, pss.max_price
ORDER BY total_revenue DESC;

-- name: GetTopSellingProducts :many
SELECT 
    p.id,
    p.name,
    SUM(oi.quantity) as total_sold,
    SUM(oi.price * oi.quantity) as total_revenue,
    COUNT(DISTINCT oi.order_id) as order_count
FROM products p
JOIN product_variants pv ON p.id = pv.product_id AND pv.archived_at IS NULL
JOIN order_items oi ON pv.id = oi.product_variant_id
JOIN orders o ON oi.order_id = o.id
WHERE p.active = true
  AND ($1::timestamp IS NULL OR o.created_at >= $1)
  AND ($2::timestamp IS NULL OR o.created_at <= $2)
GROUP BY p.id, p.name
ORDER BY total_sold DESC
LIMIT $3 OFFSET $4;

-- Utility queries

-- name: RefreshProductStockSummary :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY product_stock_summary;