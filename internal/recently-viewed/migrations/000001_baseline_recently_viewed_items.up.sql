-- No cleanup job trims this table by visitor_id/age — accepted as a small,
-- deferred technical debt item (see the module's own doc comments) since the
-- catalog and expected traffic are both small; List always caps to the last
-- 20 rows per visitor via ORDER BY viewed_at DESC LIMIT, so an unbounded
-- table doesn't affect response shape, only storage growth over time.
CREATE TABLE IF NOT EXISTS recently_viewed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id TEXT NOT NULL,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (visitor_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_recently_viewed_items_visitor_id ON recently_viewed_items (visitor_id, viewed_at DESC);
