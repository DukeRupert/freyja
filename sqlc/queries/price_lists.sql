-- name: GetDefaultPriceList :one
-- Get the default price list for a tenant (used for guests and unassigned users)
SELECT
    id,
    tenant_id,
    name,
    description,
    list_type,
    is_active,
    created_at,
    updated_at
FROM price_lists
WHERE tenant_id = $1
  AND list_type = 'default'
  AND is_active = TRUE
LIMIT 1;

-- name: GetPriceListByID :one
-- Get a price list by ID
SELECT
    id,
    tenant_id,
    name,
    description,
    list_type,
    is_active,
    created_at,
    updated_at
FROM price_lists
WHERE id = $1
  AND is_active = TRUE
LIMIT 1;

-- name: GetPriceForSKU :one
-- Get the price for a specific SKU on a price list
SELECT
    id,
    tenant_id,
    price_list_id,
    product_sku_id,
    price_cents,
    compare_at_price_cents,
    is_available,
    created_at,
    updated_at
FROM price_list_entries
WHERE price_list_id = $1
  AND product_sku_id = $2
  AND is_available = TRUE
LIMIT 1;

-- name: GetPricesForSKUs :many
-- Batch fetch prices for multiple SKUs on a price list
SELECT
    product_sku_id,
    price_cents,
    compare_at_price_cents
FROM price_list_entries
WHERE price_list_id = $1
  AND product_sku_id = ANY($2::uuid[])
  AND is_available = TRUE;

-- name: GetPricesForProduct :many
-- Get all SKU prices for a product on a specific price list
SELECT
    ple.id,
    ple.product_sku_id,
    ple.price_cents,
    ple.compare_at_price_cents,
    ple.is_available,
    ps.sku,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    ps.inventory_quantity,
    ps.is_active
FROM price_list_entries ple
INNER JOIN product_skus ps ON ps.id = ple.product_sku_id
WHERE ple.price_list_id = $1
  AND ps.product_id = $2
  AND ple.is_available = TRUE
  AND ps.is_active = TRUE
ORDER BY ps.weight_value ASC, ps.grind ASC;
