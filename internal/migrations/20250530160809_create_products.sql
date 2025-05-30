-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    handle VARCHAR(255) UNIQUE NOT NULL,
    subtitle VARCHAR(255),
    description TEXT,
    thumbnail VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'rejected')),
    is_giftcard BOOLEAN NOT NULL DEFAULT false,
    discountable BOOLEAN NOT NULL DEFAULT true,
    
    -- Coffee-specific fields
    origin_country VARCHAR(100),
    region VARCHAR(100),
    farm VARCHAR(255),
    altitude_min INTEGER,
    altitude_max INTEGER,
    processing_method VARCHAR(50) CHECK (processing_method IN ('washed', 'natural', 'honey', 'semi_washed', 'other')),
    roast_level VARCHAR(20) CHECK (roast_level IN ('light', 'medium_light', 'medium', 'medium_dark', 'dark')),
    flavor_notes TEXT[], -- Array of flavor descriptors
    varietal VARCHAR(100), -- Coffee variety (e.g., Bourbon, Typica, Geisha)
    harvest_date DATE,
    
    -- Physical properties
    weight_grams INTEGER,
    length_cm DECIMAL(5,2),
    height_cm DECIMAL(5,2),
    width_cm DECIMAL(5,2),
    
    -- Trade/compliance
    hs_code VARCHAR(20),
    mid_code VARCHAR(20),
    material VARCHAR(100),
    external_id VARCHAR(100),
    
    -- Product relationships (foreign keys)
    product_type_id UUID,
    collection_id UUID,
    
    -- Metadata and timestamps
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for better query performance
CREATE INDEX idx_products_handle ON products(handle);
CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_products_origin_country ON products(origin_country);
CREATE INDEX idx_products_roast_level ON products(roast_level);
CREATE INDEX idx_products_processing_method ON products(processing_method);
CREATE INDEX idx_products_product_type_id ON products(product_type_id);
CREATE INDEX idx_products_collection_id ON products(collection_id);
CREATE INDEX idx_products_created_at ON products(created_at);
CREATE INDEX idx_products_deleted_at ON products(deleted_at) WHERE deleted_at IS NULL;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_products_updated_at 
    BEFORE UPDATE ON products 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_products_updated_at ON products;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS products;
-- +goose StatementEnd