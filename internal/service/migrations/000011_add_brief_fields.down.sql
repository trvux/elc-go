ALTER TABLE services DROP CONSTRAINT IF EXISTS services_brief_fields_is_array;
ALTER TABLE services DROP COLUMN IF EXISTS brief_fields;
