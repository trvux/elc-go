-- The `seo` jsonb field (title/description/noindex) is no longer read by
-- any Go code — the frontend's whole SEO metadata/JSON-LD generation system
-- was deleted pending a from-scratch rebuild. meta_title/meta_description
-- stay untouched, still actively used.
ALTER TABLE services DROP COLUMN IF EXISTS seo;
