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

-- Admin queries

-- name: CreatePriceListEntry :one
-- Create a new price list entry for a SKU
INSERT INTO price_list_entries (
    tenant_id,
    price_list_id,
    product_sku_id,
    price_cents,
    compare_at_price_cents,
    is_available
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdatePriceListEntry :one
-- Update an existing price list entry
UPDATE price_list_entries
SET
    price_cents = $3,
    compare_at_price_cents = $4,
    is_available = $5,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: ListAllPriceLists :many
-- List all price lists for a tenant
SELECT *
FROM price_lists
WHERE tenant_id = $1
  AND is_active = TRUE
ORDER BY list_type DESC, name ASC;

-- name: CreatePriceList :one
-- Create a new price list
INSERT INTO price_lists (
    tenant_id,
    name,
    description,
    list_type,
    is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdatePriceList :one
-- Update a price list
UPDATE price_lists
SET
    name = $3,
    description = $4,
    is_active = $5,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: DeletePriceList :exec
-- Soft delete a price list (set inactive)
UPDATE price_lists
SET is_active = FALSE, updated_at = NOW()
WHERE tenant_id = $1 AND id = $2;

-- name: GetPriceListWithEntryCount :one
-- Get a price list with count of entries
SELECT
    pl.*,
    (SELECT COUNT(*) FROM price_list_entries ple WHERE ple.price_list_id = pl.id) as entry_count,
    (SELECT COUNT(*) FROM user_price_lists upl WHERE upl.price_list_id = pl.id) as customer_count
FROM price_lists pl
WHERE pl.tenant_id = $1 AND pl.id = $2;

-- name: ListPriceListEntries :many
-- List all entries for a price list with product/SKU details
SELECT
    ple.id,
    ple.price_list_id,
    ple.product_sku_id,
    ple.price_cents,
    ple.compare_at_price_cents,
    ple.is_available,
    ps.sku,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    ps.base_price_cents,
    p.name as product_name,
    p.slug as product_slug
FROM price_list_entries ple
INNER JOIN product_skus ps ON ps.id = ple.product_sku_id
INNER JOIN products p ON p.id = ps.product_id
WHERE ple.price_list_id = $1
ORDER BY p.name ASC, ps.weight_value ASC;

-- name: UpsertPriceListEntry :exec
-- Create or update a price list entry
INSERT INTO price_list_entries (
    tenant_id,
    price_list_id,
    product_sku_id,
    price_cents,
    compare_at_price_cents,
    is_available
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (price_list_id, product_sku_id) DO UPDATE
SET
    price_cents = EXCLUDED.price_cents,
    compare_at_price_cents = EXCLUDED.compare_at_price_cents,
    is_available = EXCLUDED.is_available,
    updated_at = NOW();

-- name: DeletePriceListEntry :exec
-- Delete a price list entry
DELETE FROM price_list_entries
WHERE tenant_id = $1 AND id = $2;
