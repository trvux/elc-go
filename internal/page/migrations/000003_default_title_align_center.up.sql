-- See internal/news/migrations/000008_default_title_align_center.up.sql
-- for the full rationale — same change, mirrored across every module with
-- a titleAlign/nameAlign field.
ALTER TABLE pages ALTER COLUMN title_align SET DEFAULT 'center';
UPDATE pages SET title_align = 'center' WHERE title_align = 'left';
