ALTER TABLE attribute_definitions DROP CONSTRAINT attribute_definitions_data_type_check;
ALTER TABLE attribute_definitions ADD CONSTRAINT attribute_definitions_data_type_check
    CHECK (data_type IN ('number', 'text', 'boolean', 'select'));
