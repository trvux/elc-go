ALTER TABLE projects ADD COLUMN IF NOT EXISTS images_old TEXT[] NOT NULL DEFAULT '{}';

UPDATE projects
SET images_old = COALESCE(
    (SELECT array_agg(elem->>'url') FROM jsonb_array_elements(images) AS elem),
    '{}'::text[]
);

ALTER TABLE projects DROP COLUMN images;
ALTER TABLE projects RENAME COLUMN images_old TO images;
