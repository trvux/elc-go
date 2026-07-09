ALTER TABLE products ADD COLUMN IF NOT EXISTS images_old TEXT[] NOT NULL DEFAULT '{}';

UPDATE products
SET images_old = COALESCE(
    (SELECT array_agg(elem->>'url') FROM jsonb_array_elements(images) AS elem),
    '{}'::text[]
);

ALTER TABLE products DROP COLUMN images;
ALTER TABLE products RENAME COLUMN images_old TO images;
