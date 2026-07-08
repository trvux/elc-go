-- projects.category_id has no FK constraint (unlike products.category_id)
-- and every write path (elc-tem's admin form) sends the all-zero placeholder
-- UUID — real category assignment has always gone through project_category
-- (many-to-many, condition-aware, real FK). Confirmed via psql + grep before
-- dropping, not assumed from the baseline migration file.
ALTER TABLE projects DROP COLUMN IF EXISTS category_id;
