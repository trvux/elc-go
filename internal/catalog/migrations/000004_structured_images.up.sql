-- images TEXT[] -> JSONB array of {url, alt, caption} — matches schema.org
-- ImageObject / Shopify's {src, altText} pattern (audit finding).
ALTER TABLE products ADD COLUMN IF NOT EXISTS images_new JSONB NOT NULL DEFAULT '[]';

UPDATE products
SET images_new = COALESCE(
    (SELECT jsonb_agg(jsonb_build_object('url', u)) FROM unnest(images) AS u),
    '[]'::jsonb
)
WHERE images_new = '[]';

ALTER TABLE products DROP COLUMN images;
ALTER TABLE products RENAME COLUMN images_new TO images;
