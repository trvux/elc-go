DROP INDEX IF EXISTS idx_product_lines_mpn_prefixes;
ALTER TABLE product_lines DROP COLUMN IF EXISTS mpn_prefixes;
