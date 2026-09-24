-- Lets the admin choose left/center/right alignment for the page title —
-- see internal/news/migrations/000007_add_title_align.up.sql for the
-- original pattern this mirrors. Defaults to 'left' (today's fixed
-- behavior), so existing rows render unchanged.
ALTER TABLE pages ADD COLUMN IF NOT EXISTS title_align TEXT NOT NULL DEFAULT 'left';
ALTER TABLE pages ADD CONSTRAINT pages_title_align_check
    CHECK (title_align IN ('left', 'center', 'right'));
