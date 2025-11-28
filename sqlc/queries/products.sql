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
