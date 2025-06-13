-- +goose Up
-- +goose StatementBegin
-- Add pre-computed options text to variants table
ALTER TABLE product_variants 
ADD COLUMN options_display TEXT;

-- Function to generate options display text
CREATE OR REPLACE FUNCTION generate_variant_options_display(variant_id INTEGER)
RETURNS TEXT AS $$
DECLARE
    options_text TEXT;
BEGIN
    SELECT STRING_AGG(
        po.option_key || ': ' || pov.value, 
        ', ' 
        ORDER BY po.option_key
    )
    INTO options_text
    FROM product_variant_options pvo
    JOIN product_options po ON pvo.product_option_id = po.id
    JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
    WHERE pvo.product_variant_id = variant_id;
    
    RETURN COALESCE(options_text, 'Default');
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update options display when variant options change
CREATE OR REPLACE FUNCTION update_variant_options_display()
RETURNS TRIGGER AS $$
BEGIN
    -- Handle INSERT/UPDATE of variant options
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE product_variants 
        SET options_display = generate_variant_options_display(NEW.product_variant_id)
        WHERE id = NEW.product_variant_id;
        RETURN NEW;
    END IF;
    
    -- Handle DELETE of variant options
    IF TG_OP = 'DELETE' THEN
        UPDATE product_variants 
        SET options_display = generate_variant_options_display(OLD.product_variant_id)
        WHERE id = OLD.product_variant_id;
        RETURN OLD;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain options display
CREATE TRIGGER update_variant_options_display_trigger
    AFTER INSERT OR UPDATE OR DELETE ON product_variant_options
    FOR EACH ROW EXECUTE FUNCTION update_variant_options_display();

-- Trigger to update when option values change
CREATE OR REPLACE FUNCTION update_variant_display_on_value_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Update all variants that use this option value
    UPDATE product_variants 
    SET options_display = generate_variant_options_display(pvo.product_variant_id)
    FROM product_variant_options pvo
    WHERE pvo.product_option_value_id = NEW.id
    AND product_variants.id = pvo.product_variant_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_variant_display_on_value_change_trigger
    AFTER UPDATE ON product_option_values
    FOR EACH ROW EXECUTE FUNCTION update_variant_display_on_value_change();

-- Index for searching by options text
CREATE INDEX idx_product_variants_options_display ON product_variants 
USING gin(to_tsvector('english', options_display));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop triggers first
DROP TRIGGER IF EXISTS update_variant_display_on_value_change_trigger ON product_option_values;
DROP TRIGGER IF EXISTS update_variant_options_display_trigger ON product_variant_options;

-- Drop functions
DROP FUNCTION IF EXISTS update_variant_display_on_value_change();
DROP FUNCTION IF EXISTS update_variant_options_display();
DROP FUNCTION IF EXISTS generate_variant_options_display(INTEGER);

-- Drop indexes
DROP INDEX IF EXISTS idx_product_variants_options_display;

-- Remove the denormalized column
ALTER TABLE product_variants DROP COLUMN IF EXISTS options_display;
-- +goose StatementEnd
