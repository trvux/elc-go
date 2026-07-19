-- Reverses 000002: re-adds the faq column exactly as
-- 000001_baseline_group_categories.up.sql originally defined it.
ALTER TABLE group_categories ADD COLUMN IF NOT EXISTS faq JSONB DEFAULT '[]'::jsonb;
