-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_product_stock_summary_product_id 
ON product_stock_summary(product_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_stock_summary_product_id;
-- +goose StatementEnd
