ALTER TABLE branches ADD COLUMN IF NOT EXISTS image_url TEXT;

UPDATE branches
SET image_url = images->0->>'url'
WHERE jsonb_array_length(images) > 0;

ALTER TABLE branches DROP COLUMN images;
