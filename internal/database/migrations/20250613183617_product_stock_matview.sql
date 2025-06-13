-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW product_stock_summary AS
SELECT 
    p.id as product_id,
    p.name,
    p.description,
    p.active as product_active,
    
    -- Stock aggregations
    COALESCE(SUM(pv.stock), 0) as total_stock,
    COALESCE(COUNT(pv.id) FILTER (WHERE pv.stock > 0), 0) as variants_in_stock,
    COALESCE(COUNT(pv.id), 0) as total_variants,
    
    -- Pricing insights  
    MIN(pv.price) as min_price,
    MAX(pv.price) as max_price,
    
    -- Availability flags
    CASE 
        WHEN SUM(pv.stock) > 0 THEN true 
        ELSE false 
    END as has_stock,
    
    CASE 
        WHEN COUNT(pv.id) FILTER (WHERE pv.stock > 0) = COUNT(pv.id) THEN 'all_in_stock'
        WHEN COUNT(pv.id) FILTER (WHERE pv.stock > 0) > 0 THEN 'partial_stock'
        ELSE 'out_of_stock'
    END as stock_status,
    
    -- Option summary (for display)
    STRING_AGG(DISTINCT po.option_key, ', ' ORDER BY po.option_key) as available_options,
    
    -- Last updated
    MAX(pv.updated_at) as last_stock_update
    
FROM products p
LEFT JOIN product_variants pv ON p.id = pv.product_id 
    AND pv.active = true 
    AND pv.archived_at IS NULL
LEFT JOIN product_variant_options pvo ON pv.id = pvo.product_variant_id
LEFT JOIN product_options po ON pvo.product_option_id = po.id
WHERE p.active = true
GROUP BY p.id, p.name, p.description, p.active;

-- Index for fast queries
CREATE INDEX idx_product_stock_summary_stock_status ON product_stock_summary(stock_status);
CREATE INDEX idx_product_stock_summary_has_stock ON product_stock_summary(has_stock);
CREATE INDEX idx_product_stock_summary_price_range ON product_stock_summary(min_price, max_price);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop indexes first
DROP INDEX IF EXISTS idx_product_stock_summary_price_range;
DROP INDEX IF EXISTS idx_product_stock_summary_has_stock;
DROP INDEX IF EXISTS idx_product_stock_summary_stock_status;

-- Drop the materialized view
DROP MATERIALIZED VIEW IF EXISTS product_stock_summary;
-- +goose StatementEnd
