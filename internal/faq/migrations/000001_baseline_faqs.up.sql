-- Polymorphic FAQ (question/answer pairs), shared across every content
-- type (system_page/product/project/service/news/branch/page) instead of
-- one nullable FK column per owner kind — see mã 8823's decision doc
-- (internal/faq/domain/types.go's OwnerType comment) for why owner_type/
-- owner_id was chosen over the internal/review-style FK-per-column
-- approach: FAQ's owner set is much larger and open to growing further,
-- and every owner table already soft-deletes (no real DELETE ever runs),
-- so losing ON DELETE CASCADE is a theoretical risk, not a real one.
CREATE TABLE IF NOT EXISTS faqs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type   TEXT NOT NULL,
    owner_id     UUID NOT NULL,
    question     TEXT NOT NULL,
    answer       TEXT NOT NULL,
    order_index  INTEGER NOT NULL DEFAULT 0,
    is_published BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
        'system_page', 'product', 'project', 'service', 'news', 'branch', 'page'
    ]))
);

CREATE INDEX IF NOT EXISTS idx_faqs_owner ON faqs (owner_type, owner_id, order_index);
