-- name: ListActiveProducts :many
-- List all active products for a tenant with their primary image
SELECT
    p.id,
    p.tenant_id,
    p.name,
    p.slug,
    p.short_description,
    p.origin,
    p.roast_level,
    p.tasting_notes,
    p.sort_order,
    pi.url as primary_image_url,
    pi.alt_text as primary_image_alt
FROM products p
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE p.tenant_id = $1
  AND p.status = 'active'
  AND p.visibility = 'public'
ORDER BY p.sort_order ASC, p.created_at DESC;

-- name: ListActiveProductsFiltered :many
-- List active products with optional filters for roast level and origin
SELECT
    p.id,
    p.tenant_id,
    p.name,
    p.slug,
    p.short_description,
    p.origin,
    p.roast_level,
    p.tasting_notes,
    p.sort_order,
    pi.url as primary_image_url,
    pi.alt_text as primary_image_alt,
    (SELECT MIN(ple.price_cents)
     FROM product_skus ps
     JOIN price_list_entries ple ON ple.product_sku_id = ps.id
     JOIN price_lists pl ON pl.id = ple.price_list_id AND pl.list_type = 'default'
     WHERE ps.product_id = p.id AND ps.is_active = TRUE
    ) as base_price
FROM products p
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE p.tenant_id = $1
  AND p.status = 'active'
  AND p.visibility = 'public'
  AND (sqlc.narg('roast_level')::text IS NULL OR p.roast_level = sqlc.narg('roast_level')::text)
  AND (sqlc.narg('origin')::text IS NULL OR p.origin = sqlc.narg('origin')::text)
  AND (sqlc.narg('tasting_note')::text IS NULL OR sqlc.narg('tasting_note')::text = ANY(p.tasting_notes))
ORDER BY p.sort_order ASC, p.created_at DESC;

-- name: GetProductFilterOptions :one
-- Get distinct filter values for the product filters UI
SELECT
    (SELECT ARRAY_AGG(DISTINCT p2.roast_level ORDER BY p2.roast_level)
     FROM products p2
     WHERE p2.tenant_id = $1 AND p2.status = 'active' AND p2.visibility = 'public' AND p2.roast_level IS NOT NULL
    ) as roast_levels,
    (SELECT ARRAY_AGG(DISTINCT p3.origin ORDER BY p3.origin)
     FROM products p3
     WHERE p3.tenant_id = $1 AND p3.status = 'active' AND p3.visibility = 'public' AND p3.origin IS NOT NULL
    ) as origins,
    (SELECT ARRAY_AGG(DISTINCT note ORDER BY note)
     FROM products p4, UNNEST(p4.tasting_notes) AS note
     WHERE p4.tenant_id = $1 AND p4.status = 'active' AND p4.visibility = 'public'
    ) as tasting_notes;

-- name: GetProductBySlug :one
-- Get a single product by slug with all details
SELECT *
FROM products
WHERE tenant_id = $1
  AND slug = $2
  AND status = 'active'
LIMIT 1;

-- name: GetProductSKUs :many
-- Get all active SKUs for a product
SELECT
    id,
    tenant_id,
    product_id,
    sku,
    weight_value,
    weight_unit,
    grind,
    base_price_cents,
    inventory_quantity,
    inventory_policy,
    low_stock_threshold,
    is_active,
    weight_grams,
    requires_shipping,
    created_at,
    updated_at
FROM product_skus
WHERE product_id = $1
  AND is_active = TRUE
ORDER BY weight_value ASC, grind ASC;

-- name: GetProductImages :many
-- Get all images for a product
SELECT
    id,
    tenant_id,
    product_id,
    url,
    alt_text,
    width,
    height,
    file_size,
    sort_order,
    is_primary,
    created_at
FROM product_images
WHERE product_id = $1
ORDER BY
    is_primary DESC,
    sort_order ASC,
    created_at ASC;

-- name: GetPrimaryImage :one
-- Get the primary image for a product
SELECT
    id,
    tenant_id,
    product_id,
    url,
    alt_text,
    width,
    height,
    file_size,
    sort_order,
    is_primary,
    created_at
FROM product_images
WHERE product_id = $1
  AND is_primary = TRUE
LIMIT 1;

-- name: GetSKUByID :one
-- Get a single SKU by ID
SELECT
    id,
    tenant_id,
    product_id,
    sku,
    weight_value,
    weight_unit,
    grind,
    base_price_cents,
    inventory_quantity,
    inventory_policy,
    low_stock_threshold,
    is_active,
    weight_grams,
    requires_shipping,
    created_at,
    updated_at
FROM product_skus
WHERE id = $1
  AND is_active = TRUE
LIMIT 1;

-- name: GetSKUWithProduct :one
-- Get a SKU with its product details (for checkout display)
SELECT
    ps.id as sku_id,
    ps.tenant_id,
    ps.product_id,
    ps.sku,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    ps.base_price_cents,
    ps.is_active,
    p.name as product_name,
    p.slug as product_slug,
    p.short_description as product_short_description,
    p.origin as product_origin,
    p.roast_level as product_roast_level,
    pi.url as product_image_url
FROM product_skus ps
INNER JOIN products p ON p.id = ps.product_id
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE ps.id = $1
  AND ps.tenant_id = $2
  AND ps.is_active = TRUE
  AND p.status = 'active'
LIMIT 1;

-- name: GetProductsForCustomer :many
-- Get all products available to a specific customer
SELECT p.*
FROM products p
WHERE p.tenant_id = $1
  AND p.status = 'active'
  AND (
    -- Standard products visible to this customer's price list
    (p.is_white_label = FALSE AND p.visibility != 'hidden')
    OR
    -- White-label products specifically for this customer
    (p.is_white_label = TRUE AND p.white_label_customer_id = $2)
  )
ORDER BY p.name;

-- name: GetWhiteLabelProductsForCustomer :many
-- Get all white-label products for a specific customer
SELECT p.*
FROM products p
WHERE p.tenant_id = $1
  AND p.is_white_label = TRUE
  AND p.white_label_customer_id = $2
  AND p.status = 'active'
ORDER BY p.name;

-- name: GetBaseProductForWhiteLabel :one
-- Get the base product for a white-label product
SELECT base.*
FROM products p
INNER JOIN products base ON base.id = p.base_product_id
WHERE p.id = $1
  AND p.is_white_label = TRUE;

-- Admin queries

-- name: ListAllProducts :many
-- List all products for admin (includes inactive and all visibility levels)
SELECT
    p.id,
    p.tenant_id,
    p.name,
    p.slug,
    p.short_description,
    p.status,
    p.visibility,
    p.origin,
    p.roast_level,
    p.sort_order,
    p.created_at,
    p.updated_at,
    pi.url as primary_image_url,
    pi.alt_text as primary_image_alt
FROM products p
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE p.tenant_id = $1
ORDER BY p.created_at DESC;

-- name: GetProductByID :one
-- Get a single product by ID (admin - no status filter)
SELECT *
FROM products
WHERE tenant_id = $1
  AND id = $2
LIMIT 1;

-- name: CreateProduct :one
-- Create a new product
INSERT INTO products (
    tenant_id,
    name,
    slug,
    short_description,
    description,
    status,
    visibility,
    origin,
    region,
    producer,
    process,
    roast_level,
    tasting_notes,
    elevation_min,
    elevation_max,
    is_white_label,
    base_product_id,
    white_label_customer_id,
    sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
RETURNING *;

-- name: UpdateProduct :one
-- Update an existing product
UPDATE products
SET
    name = $3,
    slug = $4,
    short_description = $5,
    description = $6,
    status = $7,
    visibility = $8,
    origin = $9,
    region = $10,
    producer = $11,
    process = $12,
    roast_level = $13,
    tasting_notes = $14,
    elevation_min = $15,
    elevation_max = $16,
    sort_order = $17,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: DeleteProduct :exec
-- Soft delete a product (set status to 'archived')
UPDATE products
SET
    status = 'archived',
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: CreateProductSKU :one
-- Create a new product SKU
INSERT INTO product_skus (
    tenant_id,
    product_id,
    sku,
    weight_value,
    weight_unit,
    grind,
    base_price_cents,
    inventory_quantity,
    inventory_policy,
    low_stock_threshold,
    is_active,
    weight_grams,
    requires_shipping
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: UpdateProductSKU :one
-- Update an existing product SKU
UPDATE product_skus
SET
    sku = $3,
    weight_value = $4,
    weight_unit = $5,
    grind = $6,
    base_price_cents = $7,
    inventory_quantity = $8,
    inventory_policy = $9,
    low_stock_threshold = $10,
    is_active = $11,
    weight_grams = $12,
    requires_shipping = $13,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: DeleteProductSKU :exec
-- Soft delete a product SKU (set is_active to false)
UPDATE product_skus
SET
    is_active = FALSE,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2;

-- name: CreateProductImage :one
-- Create a new product image
INSERT INTO product_images (
    tenant_id,
    product_id,
    url,
    alt_text,
    width,
    height,
    file_size,
    sort_order,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpdateProductImage :one
-- Update an existing product image
UPDATE product_images
SET
    url = $3,
    alt_text = $4,
    width = $5,
    height = $6,
    file_size = $7,
    sort_order = $8,
    is_primary = $9
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: DeleteProductImage :exec
-- Delete a product image
DELETE FROM product_images
WHERE tenant_id = $1
  AND id = $2;

-- name: SetPrimaryImage :exec
-- Set a product image as primary (and unset others)
UPDATE product_images pi
SET is_primary = (pi.id = $2)
WHERE pi.tenant_id = $1
  AND pi.product_id = (SELECT product_id FROM product_images WHERE id = $2);

-- name: GetPriceListForUser :one
-- Get the price list assigned to a user, or NULL if none
SELECT upl.price_list_id
FROM user_price_lists upl
WHERE upl.user_id = $1
LIMIT 1;

-- name: ListProductsWithSKUsForWholesale :many
-- Get all active products with their SKUs and prices for wholesale ordering matrix view
-- This query denormalizes the data for efficient display in a table format
SELECT
    p.id as product_id,
    p.name as product_name,
    p.slug as product_slug,
    p.origin as product_origin,
    pi.url as product_image_url,
    ps.id as sku_id,
    ps.sku as sku_code,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    ps.inventory_quantity,
    ps.inventory_policy,
    ps.low_stock_threshold,
    ple.price_cents
FROM products p
INNER JOIN product_skus ps ON ps.product_id = p.id AND ps.is_active = TRUE
INNER JOIN price_list_entries ple ON ple.product_sku_id = ps.id AND ple.price_list_id = $2
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE p.tenant_id = $1
  AND p.status = 'active'
  AND (p.visibility = 'public' OR p.visibility = 'wholesale_only')
ORDER BY p.sort_order ASC, p.name ASC, ps.weight_value ASC, ps.grind ASC;
