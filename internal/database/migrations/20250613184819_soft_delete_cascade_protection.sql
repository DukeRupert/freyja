-- +goose Up
-- +goose StatementBegin

-- Enhanced function to prevent option deletion when active variants exist
CREATE OR REPLACE FUNCTION prevent_option_deletion_with_active_variants()
RETURNS TRIGGER AS $$
DECLARE
    active_variant_count INTEGER;
    variant_names TEXT[];
BEGIN
    -- Check if any ACTIVE (non-archived) variants use this option
    SELECT COUNT(*), ARRAY_AGG(pv.name)
    INTO active_variant_count, variant_names
    FROM product_variant_options pvo
    JOIN product_variants pv ON pvo.product_variant_id = pv.id
    WHERE pvo.product_option_id = OLD.id
    AND pv.archived_at IS NULL
    AND pv.active = true;
    
    IF active_variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option "%" because % active variant(s) are using it: %. Please archive these variants first.', 
            OLD.option_key, 
            active_variant_count, 
            ARRAY_TO_STRING(variant_names, ', ');
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Enhanced function to prevent option value deletion when active variants exist
CREATE OR REPLACE FUNCTION prevent_option_value_deletion_with_active_variants()
RETURNS TRIGGER AS $$
DECLARE
    active_variant_count INTEGER;
    variant_names TEXT[];
    option_key TEXT;
BEGIN
    -- Get the option key for better error messaging
    SELECT po.option_key INTO option_key
    FROM product_options po
    WHERE po.id = OLD.product_option_id;
    
    -- Check if any ACTIVE (non-archived) variants use this option value
    SELECT COUNT(*), ARRAY_AGG(pv.name)
    INTO active_variant_count, variant_names
    FROM product_variant_options pvo
    JOIN product_variants pv ON pvo.product_variant_id = pv.id
    WHERE pvo.product_option_value_id = OLD.id
    AND pv.archived_at IS NULL
    AND pv.active = true;
    
    IF active_variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option value "%" from option "%" because % active variant(s) are using it: %. Please archive these variants first.', 
            OLD.value,
            option_key, 
            active_variant_count, 
            ARRAY_TO_STRING(variant_names, ', ');
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Function to cascade archive variants when an option is being "archived" (if you add archive functionality)
CREATE OR REPLACE FUNCTION cascade_archive_variants_on_option_change()
RETURNS TRIGGER AS $$
DECLARE
    affected_variants INTEGER[];
BEGIN
    -- This function can be used if you add an "archived" field to options
    -- For now, it's a placeholder for future functionality
    
    -- If an option becomes inactive, you might want to archive dependent variants
    -- This is optional and depends on your business logic
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to validate that archived variants don't have invalid option combinations
CREATE OR REPLACE FUNCTION validate_variant_archive_integrity()
RETURNS TRIGGER AS $$
DECLARE
    invalid_option_count INTEGER;
    missing_options TEXT[];
BEGIN
    -- Only validate when archiving a variant (setting archived_at)
    IF OLD.archived_at IS NULL AND NEW.archived_at IS NOT NULL THEN
        RETURN NEW; -- Archiving is always allowed
    END IF;
    
    -- Only validate when un-archiving a variant (clearing archived_at)
    IF OLD.archived_at IS NOT NULL AND NEW.archived_at IS NULL THEN
        -- Check if all required options still exist and are valid
        SELECT COUNT(*)
        INTO invalid_option_count
        FROM product_variant_options pvo
        JOIN product_options po ON pvo.product_option_id = po.id
        JOIN product_option_values pov ON pvo.product_option_value_id = pov.id
        WHERE pvo.product_variant_id = NEW.id
        AND (po.product_id != NEW.product_id OR pov.product_option_id != po.id);
        
        IF invalid_option_count > 0 THEN
            RAISE EXCEPTION 'Cannot un-archive variant "%" because it has invalid option combinations. Please recreate the variant instead.', NEW.name;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update the existing triggers to use the enhanced functions
DROP TRIGGER IF EXISTS prevent_option_deletion_trigger ON product_options;
CREATE TRIGGER prevent_option_deletion_trigger
    BEFORE DELETE ON product_options
    FOR EACH ROW EXECUTE FUNCTION prevent_option_deletion_with_active_variants();

DROP TRIGGER IF EXISTS prevent_option_value_deletion_trigger ON product_option_values;
CREATE TRIGGER prevent_option_value_deletion_trigger
    BEFORE DELETE ON product_option_values
    FOR EACH ROW EXECUTE FUNCTION prevent_option_value_deletion_with_active_variants();

-- Add trigger for variant archive validation
CREATE TRIGGER validate_variant_archive_integrity_trigger
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION validate_variant_archive_integrity();

-- Helper function to safely archive all variants for a product (utility function)
CREATE OR REPLACE FUNCTION archive_all_product_variants(target_product_id INTEGER)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    -- Archive all active variants for a product
    UPDATE product_variants
    SET archived_at = NOW(),
        active = false
    WHERE product_id = target_product_id
    AND archived_at IS NULL;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Helper function to safely archive variants using specific option values
CREATE OR REPLACE FUNCTION archive_variants_using_option_value(target_option_value_id INTEGER)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    -- Archive all active variants that use a specific option value
    UPDATE product_variants
    SET archived_at = NOW(),
        active = false
    WHERE id IN (
        SELECT DISTINCT pvo.product_variant_id
        FROM product_variant_options pvo
        WHERE pvo.product_option_value_id = target_option_value_id
    )
    AND archived_at IS NULL;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop helper functions
DROP FUNCTION IF EXISTS archive_variants_using_option_value(INTEGER);
DROP FUNCTION IF EXISTS archive_all_product_variants(INTEGER);

-- Drop triggers
DROP TRIGGER IF EXISTS validate_variant_archive_integrity_trigger ON product_variants;
DROP TRIGGER IF EXISTS prevent_option_value_deletion_trigger ON product_option_values;
DROP TRIGGER IF EXISTS prevent_option_deletion_trigger ON product_options;

-- Drop functions (keep the original simpler versions if they exist)
DROP FUNCTION IF EXISTS validate_variant_archive_integrity();
DROP FUNCTION IF EXISTS cascade_archive_variants_on_option_change();
DROP FUNCTION IF EXISTS prevent_option_value_deletion_with_active_variants();
DROP FUNCTION IF EXISTS prevent_option_deletion_with_active_variants();

-- Recreate the original simpler functions from the previous migration
CREATE OR REPLACE FUNCTION prevent_option_deletion_with_variants()
RETURNS TRIGGER AS $$
DECLARE
    variant_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO variant_count
    FROM product_variant_options pvo
    WHERE pvo.product_option_id = OLD.id;
    
    IF variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option that is used by % variant(s). Archive variants first.', variant_count;
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_option_value_deletion_with_variants()
RETURNS TRIGGER AS $$
DECLARE
    variant_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO variant_count
    FROM product_variant_options pvo
    WHERE pvo.product_option_value_id = OLD.id;
    
    IF variant_count > 0 THEN
        RAISE EXCEPTION 'Cannot delete option value that is used by % variant(s). Archive variants first.', variant_count;
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Recreate original triggers
CREATE TRIGGER prevent_option_deletion_trigger
    BEFORE DELETE ON product_options
    FOR EACH ROW EXECUTE FUNCTION prevent_option_deletion_with_variants();

CREATE TRIGGER prevent_option_value_deletion_trigger
    BEFORE DELETE ON product_option_values
    FOR EACH ROW EXECUTE FUNCTION prevent_option_value_deletion_with_variants();

-- +goose StatementEnd