-- Portfolio/case-study fields standard on contractor marketing sites (Daikin/
-- Carrier dealer portfolios) — increases trust + feeds Review/local SEO.
-- See docs/SEO-REDESIGN audit finding #6.
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS client_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS location TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS completed_at DATE,
    ADD COLUMN IF NOT EXISTS testimonial_quote TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS testimonial_author TEXT NOT NULL DEFAULT '';
