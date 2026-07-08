-- branches.image_url (single TEXT) -> branches.images (JSONB array).
ALTER TABLE branches ADD COLUMN IF NOT EXISTS images JSONB NOT NULL DEFAULT '[]';

UPDATE branches
SET images = CASE
    WHEN image_url IS NOT NULL AND image_url != '' THEN jsonb_build_array(jsonb_build_object('url', image_url))
    ELSE '[]'::jsonb
END
WHERE images = '[]';

ALTER TABLE branches DROP COLUMN image_url;
