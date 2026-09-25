-- See internal/news/migrations/000008_default_title_align_center.up.sql
-- for the full rationale — same change, mirrored across every module with
-- a titleAlign/nameAlign field.
ALTER TABLE branches ALTER COLUMN name_align SET DEFAULT 'center';
UPDATE branches SET name_align = 'center' WHERE name_align = 'left';
