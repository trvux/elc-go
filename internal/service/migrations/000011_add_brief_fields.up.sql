-- Expand step: typed replacement for brief_prompts (docs/rfc-service-restructure.md in elc-temp).
--   brief_fields  what the customer is asked for before staff can quote, as typed fields:
--                 [{"key":"area_m2","label":"Diện tích cần làm lạnh","kind":"area"}, ...]
--                 kind is text | number | choice | yesno | area; a stable key lets a lead's
--                 answer keep its meaning when admin rewords the label.
-- brief_prompts stays: readers fall back to it while brief_fields is empty, and it is
-- dropped in the contract step once every service has typed fields.
ALTER TABLE services
    ADD COLUMN IF NOT EXISTS brief_fields jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE services DROP CONSTRAINT IF EXISTS services_brief_fields_is_array;
ALTER TABLE services
    ADD CONSTRAINT services_brief_fields_is_array CHECK (jsonb_typeof(brief_fields) = 'array');
