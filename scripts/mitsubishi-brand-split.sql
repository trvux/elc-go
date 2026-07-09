-- Mitsubishi Electric and Mitsubishi Heavy Industries are two distinct
-- companies with separate AC product lines (confirmed via web research
-- 2026-07-08) — our brands table had a single "Mitsubishi" row (0 products
-- attached, so this rename is risk-free). Split into two real brand rows
-- before populating product_lines, so tiers attach to the correct brand.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/mitsubishi-brand-split.sql

BEGIN;

UPDATE brands
SET name = 'Mitsubishi Electric', slug = 'mitsubishi-electric', updated_at = now()
WHERE name = 'Mitsubishi' AND deleted_at IS NULL;

INSERT INTO brands (name, slug)
SELECT 'Mitsubishi Heavy Industries', 'mitsubishi-heavy-industries'
WHERE NOT EXISTS (
    SELECT 1 FROM brands WHERE name = 'Mitsubishi Heavy Industries' AND deleted_at IS NULL
);

SELECT name, slug FROM brands WHERE name LIKE 'Mitsubishi%' AND deleted_at IS NULL;

COMMIT;
