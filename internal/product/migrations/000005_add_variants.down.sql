ALTER TABLE products DROP COLUMN IF EXISTS price_max;
ALTER TABLE products DROP COLUMN IF EXISTS price_min;
ALTER TABLE products DROP COLUMN IF EXISTS display_stock_status;
ALTER TABLE products DROP COLUMN IF EXISTS display_price;
ALTER TABLE products DROP COLUMN IF EXISTS default_variant_id;
ALTER TABLE products DROP COLUMN IF EXISTS warranty_terms;
ALTER TABLE products DROP COLUMN IF EXISTS warranty_months;
ALTER TABLE products DROP COLUMN IF EXISTS short_description;
ALTER TABLE products DROP COLUMN IF EXISTS product_line_id;

DROP TABLE IF EXISTS product_variant_components;
DROP TABLE IF EXISTS product_variant_option_values;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS product_option_values;
DROP TABLE IF EXISTS product_options;
DROP TABLE IF EXISTS product_lines;

DROP TYPE IF EXISTS product_variant_stock_status;
