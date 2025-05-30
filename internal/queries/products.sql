-- name: CreateProduct :one
INSERT INTO products (
    title, handle, subtitle, description, thumbnail, status, is_giftcard, discountable,
    origin_country, region, farm, altitude_min, altitude_max, processing_method, roast_level,
    flavor_notes, varietal, harvest_date, weight_grams, length_cm, height_cm, width_cm,
    hs_code, mid_code, material, external_id, product_type_id, collection_id, metadata,
    subscription_enabled, subscription_intervals, min_subscription_quantity, max_subscription_quantity,
    subscription_discount_percentage, subscription_priority
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20, $21, $22,
    $23, $24, $25, $26, $27, $28, $29,
    $30, $31, $32, $33, $34, $35
) RETURNING *;

-- name: GetProduct :one
SELECT * FROM products 
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProductByHandle :one
SELECT * FROM products 
WHERE handle = $1 AND deleted_at IS NULL;

-- name: ListProducts :many
SELECT * FROM products 
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListProductsByStatus :many
SELECT * FROM products 
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListProductsByOrigin :many
SELECT * FROM products 
WHERE origin_country = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListProductsByRoastLevel :many
SELECT * FROM products 
WHERE roast_level = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchProducts :many
SELECT * FROM products 
WHERE (
    title ILIKE '%' || $1 || '%' OR
    description ILIKE '%' || $1 || '%' OR
    origin_country ILIKE '%' || $1 || '%' OR
    region ILIKE '%' || $1 || '%' OR
    farm ILIKE '%' || $1 || '%' OR
    varietal ILIKE '%' || $1 || '%'
) AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProduct :one
UPDATE products SET
    title = COALESCE($2, title),
    subtitle = COALESCE($3, subtitle),
    description = COALESCE($4, description),
    thumbnail = COALESCE($5, thumbnail),
    status = COALESCE($6, status),
    is_giftcard = COALESCE($7, is_giftcard),
    discountable = COALESCE($8, discountable),
    origin_country = COALESCE($9, origin_country),
    region = COALESCE($10, region),
    farm = COALESCE($11, farm),
    altitude_min = COALESCE($12, altitude_min),
    altitude_max = COALESCE($13, altitude_max),
    processing_method = COALESCE($14, processing_method),
    roast_level = COALESCE($15, roast_level),
    flavor_notes = COALESCE($16, flavor_notes),
    varietal = COALESCE($17, varietal),
    harvest_date = COALESCE($18, harvest_date),
    weight_grams = COALESCE($19, weight_grams),
    metadata = COALESCE($20, metadata),
    subscription_enabled = COALESCE($21, subscription_enabled),
    subscription_intervals = COALESCE($22, subscription_intervals),
    min_subscription_quantity = COALESCE($23, min_subscription_quantity),
    max_subscription_quantity = COALESCE($24, max_subscription_quantity),
    subscription_discount_percentage = COALESCE($25, subscription_discount_percentage),
    subscription_priority = COALESCE($26, subscription_priority),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateProductStatus :one
UPDATE products SET
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountProducts :one
SELECT COUNT(*) FROM products WHERE deleted_at IS NULL;

-- name: CountProductsByStatus :one
SELECT COUNT(*) FROM products WHERE status = $1 AND deleted_at IS NULL;

-- name: GetProductsWithinAltitudeRange :many
SELECT * FROM products 
WHERE altitude_min >= $1 AND altitude_max <= $2 AND deleted_at IS NULL
ORDER BY altitude_min ASC
LIMIT $3 OFFSET $4;

-- name: GetProductsByProcessingAndRoast :many
SELECT * FROM products 
WHERE processing_method = $1 AND roast_level = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListSubscribableProducts :many
SELECT * FROM products 
WHERE subscription_enabled = true AND status = 'published' AND deleted_at IS NULL
ORDER BY subscription_priority DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetSubscribableProduct :one
SELECT * FROM products 
WHERE id = $1 AND subscription_enabled = true AND status = 'published' AND deleted_at IS NULL;

-- name: ListProductsBySubscriptionInterval :many
SELECT * FROM products 
WHERE subscription_enabled = true 
    AND status = 'published' 
    AND $1 = ANY(subscription_intervals) 
    AND deleted_at IS NULL
ORDER BY subscription_priority DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProductSubscriptionStatus :one
UPDATE products SET
    subscription_enabled = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateProductSubscriptionSettings :one
UPDATE products SET
    subscription_intervals = $2,
    min_subscription_quantity = $3,
    max_subscription_quantity = $4,
    subscription_discount_percentage = $5,
    subscription_priority = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND subscription_enabled = true AND deleted_at IS NULL
RETURNING *;

-- name: CountSubscribableProducts :one
SELECT COUNT(*) FROM products 
WHERE subscription_enabled = true AND status = 'published' AND deleted_at IS NULL;