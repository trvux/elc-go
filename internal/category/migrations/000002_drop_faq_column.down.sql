-- Reverses 000002: re-adds the faq column exactly as
-- 000001_baseline_categories.up.sql originally defined it.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS faq JSONB DEFAULT '[]'::jsonb;
