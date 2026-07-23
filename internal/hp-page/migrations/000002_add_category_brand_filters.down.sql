ALTER TABLE hp_pages DROP COLUMN IF EXISTS brand_ids;
ALTER TABLE hp_pages DROP COLUMN IF EXISTS category_ids;

ALTER TABLE hp_pages ALTER COLUMN attribute_code SET DEFAULT 'phan_khuc_hp';
UPDATE hp_pages SET attribute_code = 'phan_khuc_hp' WHERE attribute_code IS NULL;
ALTER TABLE hp_pages ALTER COLUMN attribute_code SET NOT NULL;
UPDATE hp_pages SET attribute_values = '{}' WHERE attribute_values IS NULL;
ALTER TABLE hp_pages ALTER COLUMN attribute_values SET NOT NULL;
