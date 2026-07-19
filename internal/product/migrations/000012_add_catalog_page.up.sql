-- product_catalog_page is a true singleton — exactly one row, never
-- addressed by ID — for "Tất cả sản phẩm" (the catch-all/root product
-- listing)'s own content/SEO. Deliberately NOT a row in the generic
-- system_pages table (that module serves homepage/news-hub/policy pages, an
-- unrelated bounded context) — this config belongs to Product itself, per
-- the redesign's product-model decision.
CREATE TABLE IF NOT EXISTS product_catalog_page (
    content           jsonb,
    meta_title        text,
    meta_description  text,
    updated_at        timestamptz NOT NULL DEFAULT now()
);

INSERT INTO product_catalog_page (content, meta_title, meta_description)
SELECT NULL, NULL, NULL
WHERE NOT EXISTS (SELECT 1 FROM product_catalog_page);
