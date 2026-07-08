-- excerpt is the listing-page teaser, distinct from seo.description (SERP
-- snippet, different length/purpose) — see docs/SEO-REDESIGN.md audit.
ALTER TABLE news ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT '';
