-- internal/database/queries/variants.sql
-- Product variant queries for the variant system

-- START --
-- Basic variant CRUD operations
-- name: GetVariantsByProduct :many
SELECT * 
FROM product_variants
WHERE product_id = $1 AND archived_at IS NULL
ORDER BY price ASC, name ASC;

-- name: GetActiveVariantsByProduct :many
SELECT *
FROM product_variants
WHERE product_id = $1 AND active = true AND archived_at IS NULL
ORDER BY price ASC, name ASC;


-- END --
-- name: GetVariant :one
SELECT 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display
FROM product_variants
WHERE id = $1 AND archived_at IS NULL;

-- name: GetVariantWithOptions :one
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, 
    pv.is_subscription, pv.archived_at, pv.created_at, pv.updated_at,
    pv.stripe_product_id, pv.stripe_price_onetime_id, pv.stripe_price_14day_id,
    pv.stripe_price_21day_id, pv.stripe_price_30day_id, pv.stripe_price_60day_id,
    pv.options_display,
    COALESCE(
        json_agg(
            json_build_object(
                'option_id', po.id,
                'option_key', po.option_key,
                'value_id', pov.id,
                'value', pov.value
            ) ORDER BY po.option_key
        ) FILTER (WHERE po.id IS NOT NULL), 
        '[]'::json
    )::text as options
FROM product_variants pv
LEFT JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
LEFT JOIN product_options po ON pvo.product_option_id = po.id
LEFT JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
WHERE pv.id = $1 AND pv.archived_at IS NULL
GROUP BY pv.id;


-- name: GetVariantsInStock :many
SELECT 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display
FROM product_variants
WHERE product_id = $1 AND active = true AND stock > 0 AND archived_at IS NULL
ORDER BY price ASC, name ASC;

-- name: GetVariantByStripeProductID :one
SELECT 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display
FROM product_variants
WHERE stripe_product_id = $1 AND archived_at IS NULL;

-- name: CreateVariant :one
INSERT INTO product_variants (
    product_id, name, price, stock, active, is_subscription, options_display
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: UpdateVariant :one
UPDATE product_variants
SET
    name = COALESCE(NULLIF(@name, ''), name),
    price = COALESCE(NULLIF(@price, 0), price),
    stock = COALESCE(NULLIF(@stock, -1), stock),
    active = COALESCE(@active, active),
    is_subscription = COALESCE(@is_subscription, is_subscription),
    options_display = COALESCE(@options_display, options_display),
    updated_at = NOW()
WHERE id = @id AND archived_at IS NULL
RETURNING *;

-- name: UpdateVariantStock :one
UPDATE product_variants
SET
    stock = $2,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: IncrementVariantStock :one
UPDATE product_variants
SET
    stock = stock + $2,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: DecrementVariantStock :one
UPDATE product_variants
SET
    stock = GREATEST(stock - $2, 0),
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL AND stock >= $2
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: ActivateVariant :one
UPDATE product_variants
SET
    active = true,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: DeactivateVariant :one
UPDATE product_variants
SET
    active = false,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: ArchiveVariant :one
UPDATE product_variants
SET
    archived_at = NOW(),
    active = false,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: UnarchiveVariant :one
UPDATE product_variants
SET
    archived_at = NULL,
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NOT NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- Stripe integration queries

-- name: UpdateVariantStripeIDs :one
UPDATE product_variants
SET
    stripe_product_id = COALESCE($2, stripe_product_id),
    stripe_price_onetime_id = COALESCE($3, stripe_price_onetime_id),
    stripe_price_14day_id = COALESCE($4, stripe_price_14day_id),
    stripe_price_21day_id = COALESCE($5, stripe_price_21day_id),
    stripe_price_30day_id = COALESCE($6, stripe_price_30day_id),
    stripe_price_60day_id = COALESCE($7, stripe_price_60day_id),
    updated_at = NOW()
WHERE id = $1 AND archived_at IS NULL
RETURNING 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display;

-- name: GetVariantsNeedingStripeSync :many
SELECT 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display
FROM product_variants
WHERE active = true 
  AND archived_at IS NULL 
  AND stripe_product_id IS NULL
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: GetVariantsWithStripeProducts :many
SELECT 
    id, product_id, name, price, stock, active, is_subscription,
    archived_at, created_at, updated_at, stripe_product_id,
    stripe_price_onetime_id, stripe_price_14day_id, stripe_price_21day_id,
    stripe_price_30day_id, stripe_price_60day_id, options_display
FROM product_variants
WHERE active = true 
  AND archived_at IS NULL 
  AND stripe_product_id IS NOT NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- Search and filtering queries

-- name: SearchVariants :many
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, 
    pv.is_subscription, pv.archived_at, pv.created_at, pv.updated_at,
    pv.stripe_product_id, pv.stripe_price_onetime_id, pv.stripe_price_14day_id,
    pv.stripe_price_21day_id, pv.stripe_price_30day_id, pv.stripe_price_60day_id,
    pv.options_display,
    p.name as product_name
FROM product_variants pv
JOIN products p ON pv.product_id = p.id
WHERE pv.active = true 
  AND pv.archived_at IS NULL
  AND p.active = true
  AND (pv.name ILIKE $1 OR p.name ILIKE $1 OR pv.options_display ILIKE $1)
ORDER BY
  CASE WHEN pv.name ILIKE $1 THEN 1 
       WHEN p.name ILIKE $1 THEN 2 
       ELSE 3 END,
  p.name, pv.price;

-- name: GetVariantsByPriceRange :many
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, pv.is_subscription,
    pv.archived_at, pv.created_at, pv.updated_at, pv.stripe_product_id,
    pv.stripe_price_onetime_id, pv.stripe_price_14day_id, pv.stripe_price_21day_id,
    pv.stripe_price_30day_id, pv.stripe_price_60day_id, pv.options_display
FROM product_variants pv
JOIN products p ON pv.product_id = p.id
WHERE pv.active = true 
  AND pv.archived_at IS NULL
  AND p.active = true
  AND pv.price >= @min_price 
  AND pv.price <= @max_price
ORDER BY pv.price ASC, pv.name ASC;

-- name: GetSubscriptionVariants :many
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, pv.is_subscription,
    pv.archived_at, pv.created_at, pv.updated_at, pv.stripe_product_id,
    pv.stripe_price_onetime_id, pv.stripe_price_14day_id, pv.stripe_price_21day_id,
    pv.stripe_price_30day_id, pv.stripe_price_60day_id, pv.options_display
FROM product_variants pv
JOIN products p ON pv.product_id = p.id
WHERE pv.active = true 
  AND pv.archived_at IS NULL
  AND p.active = true
  AND pv.is_subscription = true
ORDER BY pv.price ASC, pv.name ASC;

-- name: GetLowStockVariants :many
SELECT 
    pv.id, pv.product_id, pv.name, pv.price, pv.stock, pv.active, 
    pv.is_subscription, pv.archived_at, pv.created_at, pv.updated_at,
    pv.stripe_product_id, pv.stripe_price_onetime_id, pv.stripe_price_14day_id,
    pv.stripe_price_21day_id, pv.stripe_price_30day_id, pv.stripe_price_60day_id,
    pv.options_display,
    p.name as product_name
FROM product_variants pv
JOIN products p ON pv.product_id = p.id
WHERE pv.active = true 
  AND pv.archived_at IS NULL 
  AND pv.stock <= $1
  AND p.active = true
ORDER BY pv.stock ASC, p.name ASC;