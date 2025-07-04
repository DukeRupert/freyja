-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_default_options_display()
RETURNS TRIGGER AS $$
BEGIN
    -- Set options_display to "Default" for new variants
    IF NEW.options_display IS NULL THEN
        NEW.options_display = 'Default';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_default_options_display_trigger
    BEFORE INSERT ON product_variants
    FOR EACH ROW EXECUTE FUNCTION set_default_options_display();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_default_options_display_trigger ON product_variants;
DROP FUNCTION IF EXISTS set_default_options_display();
-- +goose StatementEnd
