-- Reverses 000002: recreates the reviews table exactly as
-- 000001_baseline_reviews.up.sql originally defined it.
CREATE TABLE IF NOT EXISTS reviews (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id     UUID REFERENCES products(id) ON DELETE CASCADE,
    service_id     UUID REFERENCES services(id) ON DELETE CASCADE,
    rating         SMALLINT NOT NULL,
    comment        TEXT NOT NULL,
    reviewer_name  VARCHAR(100) NOT NULL,
    reviewer_phone VARCHAR(30),
    is_published   BOOLEAN NOT NULL DEFAULT true,
    source_ip      VARCHAR(64),
    user_agent     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_review_single_entity CHECK (
        (product_id IS NOT NULL)::int + (service_id IS NOT NULL)::int = 1
    ),
    CONSTRAINT chk_review_rating CHECK (rating BETWEEN 1 AND 5)
);

CREATE INDEX IF NOT EXISTS idx_reviews_product_id ON reviews (product_id) WHERE product_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_reviews_service_id ON reviews (service_id) WHERE service_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_reviews_is_published ON reviews (is_published);
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews (created_at DESC);
