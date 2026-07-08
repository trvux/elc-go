ALTER TABLE news ADD COLUMN IF NOT EXISTS image TEXT NOT NULL DEFAULT '';

UPDATE news
SET image = images->0->>'url'
WHERE jsonb_array_length(images) > 0;

ALTER TABLE news DROP COLUMN images;
