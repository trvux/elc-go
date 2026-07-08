CREATE TABLE IF NOT EXISTS product_tags (
    product_id  uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tag_id      uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_product_tags_tag_id ON product_tags (tag_id);
