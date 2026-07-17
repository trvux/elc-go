-- multiselect is one more data_type value (Akeneo's "Multi Select"), not a
-- separate concept — it stores 0..N chosen values from `options` instead of
-- exactly one scalar like the other four types. Storage for the chosen
-- values lives on product_attribute_values (see product module's
-- 000011_add_attribute_value_options migration), not here.
ALTER TABLE attribute_definitions DROP CONSTRAINT attribute_definitions_data_type_check;
ALTER TABLE attribute_definitions ADD CONSTRAINT attribute_definitions_data_type_check
    CHECK (data_type IN ('number', 'text', 'boolean', 'select', 'multiselect'));
