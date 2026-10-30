-- Originally created news_tags (join to the `tags` table). The tags feature was
-- removed (2026-10-31) and its tables are dropped by a later migration; this
-- file is kept as a no-op so the version number stays valid on databases that
-- already applied it, and so a fresh database no longer fails on
-- `REFERENCES tags` now that the tags table is never created.
SELECT 1;
