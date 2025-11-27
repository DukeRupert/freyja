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

-- name: GetProductBySlug :one
-- Get a single product by slug with all details
SELECT
    id,
    tenant_id,
    name,
    slug,
    description,
    short_description,
    origin,
    region,
    producer,
    process,
    roast_level,
    elevation_min,
    elevation_max,
    variety,
    harvest_year,
    tasting_notes,
    status,
    visibility,
    meta_title,
    meta_description,
    sort_order,
    created_at,
    updated_at
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
