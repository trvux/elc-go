-- Lets the admin choose left/center/right alignment for the branch name —
-- see internal/news/migrations/000007_add_title_align.up.sql for the
-- original pattern this mirrors. Branch's title-equivalent field is called
-- "name" (not "title"), so this is name_align, same left/center/right
-- vocabulary. Defaults to 'left' (today's fixed behavior), so existing rows
-- render unchanged.
ALTER TABLE branches ADD COLUMN IF NOT EXISTS name_align TEXT NOT NULL DEFAULT 'left';
ALTER TABLE branches ADD CONSTRAINT branches_name_align_check
    CHECK (name_align IN ('left', 'center', 'right'));
