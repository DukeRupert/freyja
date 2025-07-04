-- +goose Up
-- +goose StatementBegin
-- Fix the ambiguous product_id reference in validate_variant_options function
CREATE OR REPLACE FUNCTION validate_variant_options()
RETURNS TRIGGER AS $$
DECLARE
    expected_option_count INTEGER;
    actual_option_count INTEGER;
    variant_product_id INTEGER;  -- Fixed variable name to avoid ambiguity
BEGIN
    -- Get the product_id for this variant
    SELECT pv.product_id INTO variant_product_id 
    FROM product_variants pv 
    WHERE pv.id = NEW.product_variant_id;
    
    -- Count expected options for this product
    SELECT COUNT(*) INTO expected_option_count
    FROM product_options po
    WHERE po.product_id = variant_product_id;  -- Use fixed variable name
    
    -- Count actual options for this variant
    SELECT COUNT(*) INTO actual_option_count
    FROM product_variant_options pvo
    WHERE pvo.product_variant_id = NEW.product_variant_id;
    
    -- If this is an INSERT and we now have all required options, validate completeness
    -- If this is an UPDATE, always validate
    IF (TG_OP = 'INSERT' AND actual_option_count = expected_option_count) OR TG_OP = 'UPDATE' THEN
        -- Ensure variant has exactly one value for each product option
        IF actual_option_count != expected_option_count THEN
            RAISE EXCEPTION 'Variant must have exactly one value for each product option. Expected: %, Actual: %', 
                expected_option_count, actual_option_count;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Recreate the trigger to use the fixed function
DROP TRIGGER IF EXISTS validate_variant_options_trigger ON product_variant_options;
CREATE TRIGGER validate_variant_options_trigger
    AFTER INSERT OR UPDATE ON product_variant_options
    FOR EACH ROW EXECUTE FUNCTION validate_variant_options();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore the original problematic function
CREATE OR REPLACE FUNCTION validate_variant_options()
RETURNS TRIGGER AS $$
DECLARE
    expected_option_count INTEGER;
    actual_option_count INTEGER;
    product_id INTEGER;  -- Original problematic variable name
BEGIN
    SELECT pv.product_id INTO product_id 
    FROM product_variants pv 
    WHERE pv.id = NEW.product_variant_id;
    
    SELECT COUNT(*) INTO expected_option_count
    FROM product_options po
    WHERE po.product_id = product_id;
    
    SELECT COUNT(*) INTO actual_option_count
    FROM product_variant_options pvo
    WHERE pvo.product_variant_id = NEW.product_variant_id;
    
    IF (TG_OP = 'INSERT' AND actual_option_count = expected_option_count) OR TG_OP = 'UPDATE' THEN
        IF actual_option_count != expected_option_count THEN
            RAISE EXCEPTION 'Variant must have exactly one value for each product option. Expected: %, Actual: %', 
                expected_option_count, actual_option_count;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS validate_variant_options_trigger ON product_variant_options;
CREATE TRIGGER validate_variant_options_trigger
    AFTER INSERT OR UPDATE ON product_variant_options
    FOR EACH ROW EXECUTE FUNCTION validate_variant_options();
-- +goose StatementEnd