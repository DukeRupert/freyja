-- Seed data for development
-- This creates a test tenant with sample coffee products, price lists, and SKUs

BEGIN;

-- Create a test tenant
INSERT INTO tenants (id, name, slug, email, phone, website, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Coffee Roasters',
    'demo',
    'hello@democoffee.com',
    '555-0100',
    'https://democoffee.com',
    'active'
);

-- Create default retail price list
INSERT INTO price_lists (id, tenant_id, name, description, list_type, is_active)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'Retail',
    'Standard retail pricing',
    'default',
    TRUE
);

-- Product 1: Ethiopia Yirgacheffe (Light Roast)
INSERT INTO products (
    id,
    tenant_id,
    name,
    slug,
    description,
    short_description,
    origin,
    region,
    producer,
    process,
    roast_level,
    elevation_min,
    elevation_max,
    variety,
    harvest_year,
    tasting_notes,
    status,
    visibility,
    sort_order
) VALUES (
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000001',
    'Ethiopia Yirgacheffe',
    'ethiopia-yirgacheffe',
    E'A stunning light roast from the birthplace of coffee. This natural process Yirgacheffe offers incredible complexity with pronounced fruit-forward notes.\n\nGrown at high altitude in the Gedeo Zone, this coffee showcases the terroir that has made Ethiopian coffees world-renowned. The natural processing method allows the coffee cherry to dry around the bean, imparting distinctive fruity characteristics.\n\nPerfect for pour-over, this coffee shines when brewed at lower temperatures (195-200°F) to preserve its delicate aromatics.',
    'Bright and fruity natural process coffee with notes of blueberry, jasmine, and bergamot',
    'Ethiopia',
    'Gedeo Zone, Yirgacheffe',
    'Worka Cooperative',
    'natural',
    'light',
    1900,
    2200,
    'Heirloom',
    2024,
    ARRAY['blueberry', 'jasmine', 'bergamot', 'dark chocolate'],
    'active',
    'public',
    1
);

-- Product 1 Image
INSERT INTO product_images (tenant_id, product_id, url, alt_text, is_primary, sort_order)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000100',
    '/images/products/ethiopia-yirgacheffe.jpg',
    'Ethiopia Yirgacheffe coffee bag',
    TRUE,
    0
);

-- Product 1 SKUs
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, is_active, weight_grams)
VALUES
    ('00000000-0000-0000-0000-000000001001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000100', 'ETH-YRG-12OZ-WB', 12, 'oz', 'whole_bean', 1800, 50, TRUE, 340),
    ('00000000-0000-0000-0000-000000001002', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000100', 'ETH-YRG-12OZ-MED', 12, 'oz', 'medium', 1800, 50, TRUE, 340),
    ('00000000-0000-0000-0000-000000001003', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000100', 'ETH-YRG-12OZ-FINE', 12, 'oz', 'fine', 1800, 50, TRUE, 340);

-- Product 1 Pricing
INSERT INTO price_list_entries (tenant_id, price_list_id, product_sku_id, price_cents, is_available)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000001001', 1800, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000001002', 1800, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000001003', 1800, TRUE);

-- Product 2: Colombia Huila (Medium Roast)
INSERT INTO products (
    id,
    tenant_id,
    name,
    slug,
    description,
    short_description,
    origin,
    region,
    producer,
    process,
    roast_level,
    elevation_min,
    elevation_max,
    variety,
    harvest_year,
    tasting_notes,
    status,
    visibility,
    sort_order
) VALUES (
    '00000000-0000-0000-0000-000000000200',
    '00000000-0000-0000-0000-000000000001',
    'Colombia Huila',
    'colombia-huila',
    E'A balanced and approachable medium roast from the renowned Huila region of Colombia. This washed process coffee offers classic Colombian characteristics with a modern twist.\n\nThe Huila region is known for producing some of Colombia''s finest specialty coffees. Grown in volcanic soil at high altitude, these beans develop exceptional sweetness and clarity.\n\nThis versatile coffee works beautifully in any brewing method, from espresso to drip. Our medium roast profile highlights the inherent sweetness while maintaining bright acidity.',
    'Classic Colombian coffee with balanced sweetness, caramel, and citrus notes',
    'Colombia',
    'Huila',
    'Various smallholders',
    'washed',
    'medium',
    1700,
    1900,
    'Caturra, Castillo',
    2024,
    ARRAY['caramel', 'orange', 'milk chocolate', 'almond'],
    'active',
    'public',
    2
);

-- Product 2 Image
INSERT INTO product_images (tenant_id, product_id, url, alt_text, is_primary, sort_order)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000200',
    '/images/products/colombia-huila.jpg',
    'Colombia Huila coffee bag',
    TRUE,
    0
);

-- Product 2 SKUs (multiple sizes)
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, is_active, weight_grams)
VALUES
    ('00000000-0000-0000-0000-000000002001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000200', 'COL-HUI-12OZ-WB', 12, 'oz', 'whole_bean', 1600, 100, TRUE, 340),
    ('00000000-0000-0000-0000-000000002002', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000200', 'COL-HUI-12OZ-MED', 12, 'oz', 'medium', 1600, 100, TRUE, 340),
    ('00000000-0000-0000-0000-000000002003', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000200', 'COL-HUI-5LB-WB', 5, 'lb', 'whole_bean', 6500, 25, TRUE, 2268);

-- Product 2 Pricing
INSERT INTO price_list_entries (tenant_id, price_list_id, product_sku_id, price_cents, is_available)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000002001', 1600, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000002002', 1600, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000002003', 6500, TRUE);

-- Product 3: House Blend (Medium-Dark Roast)
INSERT INTO products (
    id,
    tenant_id,
    name,
    slug,
    description,
    short_description,
    origin,
    region,
    process,
    roast_level,
    tasting_notes,
    status,
    visibility,
    sort_order
) VALUES (
    '00000000-0000-0000-0000-000000000300',
    '00000000-0000-0000-0000-000000000001',
    'House Blend',
    'house-blend',
    E'Our signature blend designed for everyday drinking. This crowd-pleasing medium-dark roast combines beans from multiple origins to create a balanced, full-bodied cup.\n\nBlending allows us to highlight the best characteristics of each origin while creating a consistent flavor profile year-round. We carefully source and roast each component to contribute specific qualities to the final cup.\n\nPerfect for drip coffee makers and French press. This blend also makes excellent cold brew.',
    'Our signature everyday blend - smooth, balanced, and consistently delicious',
    'Blend',
    NULL,
    'washed',
    'medium-dark',
    ARRAY['chocolate', 'nuts', 'brown sugar', 'roasted'],
    'active',
    'public',
    3
);

-- Product 3 Image
INSERT INTO product_images (tenant_id, product_id, url, alt_text, is_primary, sort_order)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000300',
    '/images/products/house-blend.jpg',
    'House Blend coffee bag',
    TRUE,
    0
);

-- Product 3 SKUs
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, is_active, weight_grams)
VALUES
    ('00000000-0000-0000-0000-000000003001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000300', 'BLEND-12OZ-WB', 12, 'oz', 'whole_bean', 1400, 200, TRUE, 340),
    ('00000000-0000-0000-0000-000000003002', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000300', 'BLEND-12OZ-MED', 12, 'oz', 'medium', 1400, 200, TRUE, 340),
    ('00000000-0000-0000-0000-000000003003', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000300', 'BLEND-12OZ-FINE', 12, 'oz', 'fine', 1400, 200, TRUE, 340),
    ('00000000-0000-0000-0000-000000003004', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000300', 'BLEND-5LB-WB', 5, 'lb', 'whole_bean', 5500, 50, TRUE, 2268);

-- Product 3 Pricing
INSERT INTO price_list_entries (tenant_id, price_list_id, product_sku_id, price_cents, is_available)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000003001', 1400, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000003002', 1400, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000003003', 1400, TRUE),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000003004', 5500, TRUE);

COMMIT;
