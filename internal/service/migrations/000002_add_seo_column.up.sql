ALTER TABLE services ADD COLUMN IF NOT EXISTS seo JSONB NOT NULL DEFAULT '{}';

UPDATE services
SET seo = jsonb_strip_nulls(jsonb_build_object(
    'title', meta_title,
    'description', meta_description
))
WHERE (meta_title IS NOT NULL OR meta_description IS NOT NULL)
  AND seo = '{}';
