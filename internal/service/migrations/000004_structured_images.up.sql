-- services.image (single TEXT) -> services.images (JSONB array) — fixes both
-- the missing alt/caption AND the inconsistent single-image cardinality vs
-- products/projects (audit finding).
ALTER TABLE services ADD COLUMN IF NOT EXISTS images JSONB NOT NULL DEFAULT '[]';

UPDATE services
SET images = CASE
    WHEN image IS NOT NULL AND image != '' THEN jsonb_build_array(jsonb_build_object('url', image))
    ELSE '[]'::jsonb
END
WHERE images = '[]';

ALTER TABLE services DROP COLUMN image;
