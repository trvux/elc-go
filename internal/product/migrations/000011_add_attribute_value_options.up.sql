-- Storage for the multiselect data_type (0..N chosen values, not 1 scalar
-- like value_text/value_number/value_boolean) — see
-- attribute module's 000002_add_multiselect_data_type migration.
ALTER TABLE product_attribute_values ADD COLUMN IF NOT EXISTS value_options TEXT[] NOT NULL DEFAULT '{}';
