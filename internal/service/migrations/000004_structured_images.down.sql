ALTER TABLE services ADD COLUMN IF NOT EXISTS image TEXT;

UPDATE services
SET image = images->0->>'url'
WHERE jsonb_array_length(images) > 0;

ALTER TABLE services DROP COLUMN images;
