-- news.image (single TEXT) -> news.images (JSONB array).
ALTER TABLE news ADD COLUMN IF NOT EXISTS images JSONB NOT NULL DEFAULT '[]';

UPDATE news
SET images = CASE
    WHEN image IS NOT NULL AND image != '' THEN jsonb_build_array(jsonb_build_object('url', image))
    ELSE '[]'::jsonb
END
WHERE images = '[]';

ALTER TABLE news DROP COLUMN image;
