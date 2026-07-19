ALTER TABLE products ADD COLUMN condition product_condition NOT NULL DEFAULT 'new';
ALTER TABLE products ADD COLUMN warranty_months integer;
ALTER TABLE products ADD COLUMN warranty_terms text;

UPDATE products p
SET condition = (CASE WHEN 'Cũ' = ANY(pav.value_options) THEN 'used' ELSE 'new' END)::product_condition
FROM product_attribute_values pav
JOIN attribute_definitions ad ON ad.id = pav.attribute_definition_id
WHERE pav.product_id = p.id AND ad.code = 'tinh_trang_san_pham';

DELETE FROM product_attribute_values
WHERE attribute_definition_id IN (
    SELECT id FROM attribute_definitions WHERE code IN ('tinh_trang_san_pham', 'bao_hanh_tong')
);
