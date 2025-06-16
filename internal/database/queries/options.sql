-- internal/database/queries/options.sql
-- Product options and option values queries

-- Product Options CRUD

-- name: GetProductOption :one
SELECT id, product_id, option_key, created_at
FROM product_options
WHERE id = $1;

-- name: GetProductOptionsByProduct :many
SELECT id, product_id, option_key, created_at
FROM product_options
WHERE product_id = $1
ORDER BY option_key ASC;

-- name: GetProductOptionByKey :one
SELECT id, product_id, option_key, created_at
FROM product_options
WHERE product_id = $1 AND option_key = $2;

-- name: CreateProductOption :one
INSERT INTO product_options (
    product_id, option_key
) VALUES (
    $1, $2
)
RETURNING id, product_id, option_key, created_at;

-- name: UpdateProductOption :one
UPDATE product_options
SET
    option_key = $2
WHERE id = $1
RETURNING id, product_id, option_key, created_at;

-- name: DeleteProductOption :exec
DELETE FROM product_options
WHERE id = $1;

-- Product Option Values CRUD

-- name: GetProductOptionValue :one
SELECT id, product_option_id, value, created_at
FROM product_option_values
WHERE id = $1;

-- name: GetProductOptionValuesByOption :many
SELECT id, product_option_id, value, created_at
FROM product_option_values
WHERE product_option_id = $1
ORDER BY value ASC;

-- name: GetProductOptionValuesByProduct :many
SELECT 
    pov.id, pov.product_option_id, pov.value, pov.created_at,
    po.option_key
FROM product_option_values pov
JOIN product_options po ON pov.product_option_id = po.id
WHERE po.product_id = $1
ORDER BY po.option_key ASC, pov.value ASC;

-- name: GetProductOptionValueByValue :one
SELECT id, product_option_id, value, created_at
FROM product_option_values
WHERE product_option_id = $1 AND value = $2;

-- name: CreateProductOptionValue :one
INSERT INTO product_option_values (
    product_option_id, value
) VALUES (
    $1, $2
)
RETURNING id, product_option_id, value, created_at;

-- name: UpdateProductOptionValue :one
UPDATE product_option_values
SET
    value = $2
WHERE id = $1
RETURNING id, product_option_id, value, created_at;

-- name: DeleteProductOptionValue :exec
DELETE FROM product_option_values
WHERE id = $1;

-- Product Variant Options (Junction Table) CRUD

-- name: GetVariantOption :one
SELECT id, product_variant_id, product_option_id, product_option_value_id
FROM product_variant_options
WHERE id = $1;

-- name: GetVariantOptionsByVariant :many
SELECT 
    pvo.id, pvo.product_variant_id, pvo.product_option_id, pvo.product_option_value_id,
    po.option_key,
    pov.value
FROM product_variant_options pvo
JOIN product_options po ON pvo.product_option_id = po.id
JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE pvo.product_variant_id = $1
ORDER BY po.option_key ASC;

-- name: GetVariantOptionsByProduct :many
SELECT 
    pvo.id, pvo.product_variant_id, pvo.product_option_id, pvo.product_option_value_id,
    po.option_key,
    pov.value,
    pv.name as variant_name,
    pv.price as variant_price
FROM product_variant_options pvo
JOIN product_options po ON pvo.product_option_id = po.id
JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
JOIN product_variants pv ON pvo.product_variant_id = pv.id
WHERE po.product_id = $1 AND pv.archived_at IS NULL
ORDER BY pv.name ASC, po.option_key ASC;

-- name: CreateVariantOption :one
INSERT INTO product_variant_options (
    product_variant_id, product_option_id, product_option_value_id
) VALUES (
    $1, $2, $3
)
RETURNING id, product_variant_id, product_option_id, product_option_value_id;

-- name: DeleteVariantOption :exec
DELETE FROM product_variant_options
WHERE id = $1;

-- name: DeleteVariantOptionsByVariant :exec
DELETE FROM product_variant_options
WHERE product_variant_id = $1;

-- name: DeleteVariantOptionsByOption :exec
DELETE FROM product_variant_options
WHERE product_option_id = $1;

-- name: DeleteVariantOptionsByOptionValue :exec
DELETE FROM product_variant_options
WHERE product_option_value_id = $1;

-- Complex option queries for variant management

-- name: GetVariantByOptionCombination :one
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, 
    pv.is_subscription, pv.archived_at, pv.created_at, pv.updated_at,
    pv.stripe_product_id, pv.stripe_price_onetime_id, pv.stripe_price_14day_id,
    pv.stripe_price_21day_id, pv.stripe_price_30day_id, pv.stripe_price_60day_id,
    pv.options_display
FROM product_variants pv
WHERE pv.product_id = $1 
  AND pv.archived_at IS NULL
  AND NOT EXISTS (
    SELECT 1 
    FROM product_variant_options pvo1
    WHERE pvo1.product_variant_id = pv.id
    AND pvo1.product_option_value_id != ANY($2::int[])
  )
  AND (
    SELECT COUNT(*) 
    FROM product_variant_options pvo2
    WHERE pvo2.product_variant_id = pv.id
  ) = array_length($2::int[], 1);

-- name: GetAvailableOptionValues :many
SELECT DISTINCT
    po.id as option_id,
    po.option_key,
    pov.id as value_id,
    pov.value
FROM product_options po
JOIN product_option_values pov ON po.id = pov.product_option_id
JOIN product_variant_options pvo ON pov.id = pvo.product_option_value_id
JOIN product_variants pv ON pvo.product_variant_id = pv.id
WHERE po.product_id = $1 
  AND pv.active = true 
  AND pv.archived_at IS NULL
ORDER BY po.option_key ASC, pov.value ASC;

-- name: GetOptionCombinationsInStock :many
SELECT 
    pv.id as variant_id,
    pv.stock,
    json_agg(
        json_build_object(
            'option_id', po.id,
            'option_key', po.option_key,
            'value_id', pov.id,
            'value', pov.value
        ) ORDER BY po.option_key
    ) as option_combination
FROM product_variants pv
JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
JOIN product_options po ON pvo.product_option_id = po.id
JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE po.product_id = $1
  AND pv.active = true
  AND pv.archived_at IS NULL
  AND pv.stock > 0
GROUP BY pv.id, pv.stock
ORDER BY pv.stock DESC;

-- name: GetVariantsWithOptionValues :many
SELECT 
    pv.id as variant_id,
    pv.name as variant_name,
    pv.price,
    pv.stock,
    pv.active,
    json_agg(
        json_build_object(
            'option_id', po.id,
            'option_key', po.option_key,
            'value_id', pov.id,
            'value', pov.value
        ) ORDER BY po.option_key
    ) as options
FROM product_variants pv
LEFT JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
LEFT JOIN product_options po ON pvo.product_option_id = po.id
LEFT JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE pv.product_id = $1 AND pv.archived_at IS NULL
GROUP BY pv.id, pv.name, pv.price, pv.stock, pv.active
ORDER BY pv.price ASC, pv.name ASC;

-- Validation and integrity queries

-- name: CheckOptionValueUsage :one
SELECT COUNT(*) as usage_count
FROM product_variant_options pvo
JOIN product_variants pv ON pvo.product_variant_id = pv.id
WHERE pvo.product_option_value_id = $1 AND pv.archived_at IS NULL;

-- name: CheckOptionUsage :one
SELECT COUNT(*) as usage_count
FROM product_variant_options pvo
JOIN product_variants pv ON pvo.product_variant_id = pv.id
WHERE pvo.product_option_id = $1 AND pv.archived_at IS NULL;

-- name: GetOrphanedOptionValues :many
SELECT pov.id, pov.product_option_id, pov.value, pov.created_at
FROM product_option_values pov
LEFT JOIN product_variant_options pvo ON pov.id = pvo.product_option_value_id
LEFT JOIN product_variants pv ON pvo.product_variant_id = pv.id AND pv.archived_at IS NULL
WHERE pv.id IS NULL;

-- name: GetOrphanedOptions :many
SELECT po.id, po.product_id, po.option_key, po.created_at
FROM product_options po
LEFT JOIN product_option_values pov ON po.id = pov.product_option_id
WHERE pov.id IS NULL;

-- Option analytics

-- name: GetOptionPopularity :many
SELECT 
    po.id,
    po.option_key,
    pov.id as value_id,
    pov.value,
    COUNT(DISTINCT pv.id) as variant_count,
    COUNT(DISTINCT oi.id) as order_count,
    COALESCE(SUM(oi.quantity), 0) as total_sold
FROM product_options po
JOIN product_option_values pov ON po.id = pov.product_option_id
LEFT JOIN product_variant_options pvo ON pov.id = pvo.product_option_value_id
LEFT JOIN product_variants pv ON pvo.product_variant_id = pv.id AND pv.archived_at IS NULL
LEFT JOIN order_items oi ON pv.id = oi.product_variant_id
LEFT JOIN orders o ON oi.order_id = o.id
WHERE po.product_id = $1
  AND ($2::timestamp IS NULL OR o.created_at >= $2)
  AND ($3::timestamp IS NULL OR o.created_at <= $3)
GROUP BY po.id, po.option_key, pov.id, pov.value
ORDER BY po.option_key ASC, total_sold DESC;