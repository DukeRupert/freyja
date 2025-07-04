-- +goose Up
-- +goose StatementBegin
-- Junction table linking variants to their option value combinations
CREATE TABLE product_variant_options (
    id SERIAL PRIMARY KEY,
    product_variant_id INTEGER NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    product_option_id INTEGER NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    product_option_value_id INTEGER NOT NULL REFERENCES product_option_values(id) ON DELETE CASCADE,
    
    -- Ensure each variant has unique option combinations
    CONSTRAINT uq_variant_option_combination UNIQUE (product_variant_id, product_option_id)
);

-- Function to validate option value belongs to option (application-level check)
CREATE OR REPLACE FUNCTION validate_option_value_belongs_to_option()
RETURNS TRIGGER AS $$
DECLARE
    value_option_id INTEGER;
BEGIN
    -- Get the option_id for this option value
    SELECT product_option_id INTO value_option_id
    FROM product_option_values
    WHERE id = NEW.product_option_value_id;
    
    -- Ensure the option value belongs to the specified option
    IF value_option_id != NEW.product_option_id THEN
        RAISE EXCEPTION 'Option value % does not belong to option %', 
            NEW.product_option_value_id, NEW.product_option_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to validate option value belongs to option
CREATE TRIGGER validate_option_value_belongs_to_option_trigger
    BEFORE INSERT OR UPDATE ON product_variant_options
    FOR EACH ROW EXECUTE FUNCTION validate_option_value_belongs_to_option();

-- Function to validate that variant has complete option set
CREATE OR REPLACE FUNCTION validate_variant_options()
RETURNS TRIGGER AS $$
DECLARE
    expected_option_count INTEGER;
    actual_option_count INTEGER;
    variant_product_id INTEGER;
BEGIN
    -- Get the product_id for this variant
    SELECT pv.product_id INTO variant_product_id 
    FROM product_variants pv 
    WHERE pv.id = NEW.product_variant_id;
    
    -- Count expected options for this product
    SELECT COUNT(*) INTO expected_option_count
    FROM product_options po
    WHERE po.product_id = variant_product_id;
    
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

-- Trigger to validate variant option completeness
CREATE TRIGGER validate_variant_options_trigger
    AFTER INSERT OR UPDATE ON product_variant_options
    FOR EACH ROW EXECUTE FUNCTION validate_variant_options();

-- Additional function to prevent deletion of options that would break variants
CREATE OR REPLACE FUNCTION prevent_option_deletion_with_variants()
RETURNS TRIGGER AS $$
DECLARE
    variant_count INTEGER;
BEGIN
    -- Check if any variants use this option
    SELECT COUNT(*) INTO variant_count
    FROM product_variant_options pvo
    WHERE pvo.product_option_id = OLD.id;
    
    IF variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option that is used by % variant(s). Archive variants first.', variant_count;
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger to prevent option deletion when variants exist
CREATE TRIGGER prevent_option_deletion_trigger
    BEFORE DELETE ON product_options
    FOR EACH ROW EXECUTE FUNCTION prevent_option_deletion_with_variants();

-- Function to prevent deletion of option values that would break variants
CREATE OR REPLACE FUNCTION prevent_option_value_deletion_with_variants()
RETURNS TRIGGER AS $$
DECLARE
    variant_count INTEGER;
BEGIN
    -- Check if any variants use this option value
    SELECT COUNT(*) INTO variant_count
    FROM product_variant_options pvo
    WHERE pvo.product_option_value_id = OLD.id;
    
    IF variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option value that is used by % variant(s). Archive variants first.', variant_count;
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger to prevent option value deletion when variants exist
CREATE TRIGGER prevent_option_value_deletion_trigger
    BEFORE DELETE ON product_option_values
    FOR EACH ROW EXECUTE FUNCTION prevent_option_value_deletion_with_variants();

-- Indexes for efficient joins and lookups
CREATE INDEX idx_variant_options_variant_id ON product_variant_options(product_variant_id);
CREATE INDEX idx_variant_options_option_id ON product_variant_options(product_option_id);
CREATE INDEX idx_variant_options_value_id ON product_variant_options(product_option_value_id);

-- Composite index for efficient combination lookups
CREATE INDEX idx_variant_options_combination ON product_variant_options(product_option_id, product_option_value_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS prevent_option_value_deletion_trigger ON product_option_values;
DROP FUNCTION IF EXISTS prevent_option_value_deletion_with_variants();
DROP TRIGGER IF EXISTS prevent_option_deletion_trigger ON product_options;
DROP FUNCTION IF EXISTS prevent_option_deletion_with_variants();
DROP TRIGGER IF EXISTS validate_variant_options_trigger ON product_variant_options;
DROP FUNCTION IF EXISTS validate_variant_options();
DROP TRIGGER IF EXISTS validate_option_value_belongs_to_option_trigger ON product_variant_options;
DROP FUNCTION IF EXISTS validate_option_value_belongs_to_option();
DROP INDEX IF EXISTS idx_variant_options_combination;
DROP INDEX IF EXISTS idx_variant_options_value_id;
DROP INDEX IF EXISTS idx_variant_options_option_id;
DROP INDEX IF EXISTS idx_variant_options_variant_id;
DROP TABLE IF EXISTS product_variant_options;
-- +goose StatementEnd