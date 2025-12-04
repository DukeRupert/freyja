-- +goose Up
-- +goose StatementBegin

-- Seed example coffee products for the master tenant
-- These demonstrate the coffee-specific attributes and serve as test data

-- Product 1: Ethiopia Yirgacheffe (Light Roast, Fruity)
INSERT INTO products (
    id, tenant_id, name, slug, description, short_description,
    origin, region, producer, process, roast_level,
    elevation_min, elevation_max, variety, harvest_year,
    tasting_notes, status, visibility, sort_order
) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Ethiopia Yirgacheffe',
    'ethiopia-yirgacheffe',
    'A stunning single-origin from the birthplace of coffee. This lot comes from smallholder farmers in the Gedeb district, where coffee grows wild among native forest. Expect a complex cup with vibrant acidity and a tea-like body.',
    'Bright and floral with notes of bergamot and stone fruit.',
    'Ethiopia', 'Yirgacheffe, Gedeb', 'Gedeb Smallholder Farmers', 'washed', 'light',
    1900, 2100, 'Heirloom', 2024,
    ARRAY['bergamot', 'peach', 'jasmine', 'honey'],
    'active', 'public', 1
);

-- Product 2: Colombia Huila (Medium Roast, Balanced)
INSERT INTO products (
    id, tenant_id, name, slug, description, short_description,
    origin, region, producer, process, roast_level,
    elevation_min, elevation_max, variety, harvest_year,
    tasting_notes, status, visibility, sort_order
) VALUES (
    'a0000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'Colombia Huila',
    'colombia-huila',
    'From the lush mountains of Huila, this coffee exemplifies the best of Colombian specialty. Grown by third-generation farmers at high altitude, it delivers a perfectly balanced cup with rich sweetness and clean finish.',
    'Sweet and balanced with caramel and citrus notes.',
    'Colombia', 'Huila, Pitalito', 'Finca La Esperanza', 'washed', 'medium',
    1700, 1900, 'Caturra, Castillo', 2024,
    ARRAY['caramel', 'orange', 'milk chocolate', 'brown sugar'],
    'active', 'public', 2
);

-- Product 3: Guatemala Antigua (Medium-Dark, Chocolatey)
INSERT INTO products (
    id, tenant_id, name, slug, description, short_description,
    origin, region, producer, process, roast_level,
    elevation_min, elevation_max, variety, harvest_year,
    tasting_notes, status, visibility, sort_order
) VALUES (
    'a0000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Guatemala Antigua',
    'guatemala-antigua',
    'Grown in the shadow of three volcanoes, Antigua coffees are renowned for their full body and complex spice notes. This lot is carefully processed at the mill and dried on raised beds for even, consistent flavor.',
    'Rich and full-bodied with dark chocolate and spice.',
    'Guatemala', 'Antigua', 'Finca San Sebastian', 'washed', 'medium-dark',
    1500, 1700, 'Bourbon, Catuai', 2024,
    ARRAY['dark chocolate', 'cinnamon', 'walnut', 'dried fig'],
    'active', 'public', 3
);

-- Product 4: Sumatra Mandheling (Dark Roast, Earthy)
INSERT INTO products (
    id, tenant_id, name, slug, description, short_description,
    origin, region, producer, process, roast_level,
    elevation_min, elevation_max, variety, harvest_year,
    tasting_notes, status, visibility, sort_order
) VALUES (
    'a0000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    'Sumatra Mandheling',
    'sumatra-mandheling',
    'A bold Indonesian coffee processed using the traditional wet-hull method unique to Sumatra. This gives it the characteristic earthy, full-bodied profile that Sumatran coffees are famous for. Perfect for those who love a heavy, syrupy cup.',
    'Bold and earthy with herbal and cedar notes.',
    'Indonesia', 'North Sumatra, Lintong', 'Mandheling Cooperative', 'wet-hulled', 'dark',
    1200, 1500, 'Typica, Catimor', 2024,
    ARRAY['cedar', 'dark cocoa', 'herbs', 'tobacco'],
    'active', 'public', 4
);

-- Product 5: House Blend (Medium, Crowd Pleaser)
INSERT INTO products (
    id, tenant_id, name, slug, description, short_description,
    origin, region, producer, process, roast_level,
    elevation_min, elevation_max, variety, harvest_year,
    tasting_notes, status, visibility, sort_order
) VALUES (
    'a0000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000001',
    'House Blend',
    'house-blend',
    'Our signature blend combines coffees from Central and South America for a perfectly balanced, approachable cup. Roasted to bring out sweetness and smooth body, this is the coffee you''ll want to drink every day.',
    'Smooth and approachable with nutty sweetness.',
    NULL, NULL, NULL, NULL, 'medium',
    NULL, NULL, NULL, NULL,
    ARRAY['hazelnut', 'milk chocolate', 'toffee'],
    'active', 'public', 0
);

-- SKUs for Ethiopia Yirgacheffe
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, weight_grams) VALUES
    ('b0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'ETH-YRG-12-WB', 12, 'oz', 'whole_bean', 1895, 50, 340),
    ('b0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'ETH-YRG-12-GR', 12, 'oz', 'ground', 1895, 30, 340),
    ('b0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'ETH-YRG-2LB-WB', 2, 'lb', 'whole_bean', 3495, 20, 907);

-- SKUs for Colombia Huila
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, weight_grams) VALUES
    ('b0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000002', 'COL-HUI-12-WB', 12, 'oz', 'whole_bean', 1695, 60, 340),
    ('b0000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000002', 'COL-HUI-12-GR', 12, 'oz', 'ground', 1695, 40, 340),
    ('b0000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000002', 'COL-HUI-2LB-WB', 2, 'lb', 'whole_bean', 2995, 25, 907);

-- SKUs for Guatemala Antigua
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, weight_grams) VALUES
    ('b0000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000003', 'GUA-ANT-12-WB', 12, 'oz', 'whole_bean', 1795, 45, 340),
    ('b0000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000003', 'GUA-ANT-12-GR', 12, 'oz', 'ground', 1795, 35, 340),
    ('b0000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000003', 'GUA-ANT-2LB-WB', 2, 'lb', 'whole_bean', 3195, 15, 907);

-- SKUs for Sumatra Mandheling
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, weight_grams) VALUES
    ('b0000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000004', 'SUM-MAN-12-WB', 12, 'oz', 'whole_bean', 1695, 40, 340),
    ('b0000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000004', 'SUM-MAN-12-GR', 12, 'oz', 'ground', 1695, 30, 340),
    ('b0000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000004', 'SUM-MAN-2LB-WB', 2, 'lb', 'whole_bean', 2995, 20, 907);

-- SKUs for House Blend (more options, it's the popular one)
INSERT INTO product_skus (id, tenant_id, product_id, sku, weight_value, weight_unit, grind, base_price_cents, inventory_quantity, weight_grams) VALUES
    ('b0000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000005', 'HB-12-WB', 12, 'oz', 'whole_bean', 1495, 100, 340),
    ('b0000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000005', 'HB-12-GR', 12, 'oz', 'ground', 1495, 80, 340),
    ('b0000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000005', 'HB-2LB-WB', 2, 'lb', 'whole_bean', 2695, 50, 907),
    ('b0000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000005', 'HB-5LB-WB', 5, 'lb', 'whole_bean', 5995, 30, 2268);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Delete in reverse order due to foreign keys
DELETE FROM product_skus WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
    AND id IN (
        'b0000000-0000-0000-0000-000000000001',
        'b0000000-0000-0000-0000-000000000002',
        'b0000000-0000-0000-0000-000000000003',
        'b0000000-0000-0000-0000-000000000004',
        'b0000000-0000-0000-0000-000000000005',
        'b0000000-0000-0000-0000-000000000006',
        'b0000000-0000-0000-0000-000000000007',
        'b0000000-0000-0000-0000-000000000008',
        'b0000000-0000-0000-0000-000000000009',
        'b0000000-0000-0000-0000-000000000010',
        'b0000000-0000-0000-0000-000000000011',
        'b0000000-0000-0000-0000-000000000012',
        'b0000000-0000-0000-0000-000000000013',
        'b0000000-0000-0000-0000-000000000014',
        'b0000000-0000-0000-0000-000000000015',
        'b0000000-0000-0000-0000-000000000016'
    );

DELETE FROM products WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
    AND id IN (
        'a0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000003',
        'a0000000-0000-0000-0000-000000000004',
        'a0000000-0000-0000-0000-000000000005'
    );

-- +goose StatementEnd
