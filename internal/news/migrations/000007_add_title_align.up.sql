-- Lets the admin choose left/center/right alignment for the article title,
-- rendered as a separate structured field outside the Tiptap content (never
-- inside the body itself, which stays limited to H2/H3) — see the FE's
-- HEADING_LEVELS comment in shared/lib/tiptap-render.ts for why title stays
-- its own field. Defaults to 'left' (today's fixed behavior), so existing
-- rows render unchanged.
ALTER TABLE news ADD COLUMN IF NOT EXISTS title_align TEXT NOT NULL DEFAULT 'left';
ALTER TABLE news ADD CONSTRAINT news_title_align_check
    CHECK (title_align IN ('left', 'center', 'right'));
