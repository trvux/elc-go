ALTER TABLE projects ADD COLUMN IF NOT EXISTS images_new JSONB NOT NULL DEFAULT '[]';

UPDATE projects
SET images_new = COALESCE(
    (SELECT jsonb_agg(jsonb_build_object('url', u)) FROM unnest(images) AS u),
    '[]'::jsonb
)
WHERE images_new = '[]';

ALTER TABLE projects DROP COLUMN images;
ALTER TABLE projects RENAME COLUMN images_new TO images;
