-- attribute_definitions replaces free-text spec labels (products.specs jsonb,
-- admin hand-typed a new label per product) with a controlled, reusable
-- schema — same "define once, reuse everywhere" pattern Shopify's Metafield
-- Definitions use, scoped per category the same way Shopify's Taxonomy
-- attaches a fixed attribute set to each product Category. See
-- docs/product-v2-design.md for the full rationale (real audit found the
-- same concept — "Độ ồn" — stored under 3 different label strings across
-- products, silently breaking facet filtering).
--
-- category_id nullable: null = applies across every category (e.g. "Xuất
-- xứ"/"Bảo hành" are universal, not AC-specific).
CREATE TABLE IF NOT EXISTS attribute_definitions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id  uuid REFERENCES categories(id) ON DELETE CASCADE,
    code         text NOT NULL,
    name         text NOT NULL,
    -- group_label sections the admin form / PDP display (e.g. "Dàn lạnh" /
    -- "Dàn nóng" / NULL = chung) — mirrors the "THÔNG TIN DÀN LẠNH"/"THÔNG
    -- TIN DÀN NÓNG" section headers already present in real spec data.
    group_label  text,
    data_type    text NOT NULL,
    unit         text,
    -- options: only meaningful for data_type = 'select'.
    options      text[] NOT NULL DEFAULT '{}',
    order_index  integer NOT NULL DEFAULT 0,
    is_required  boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    CONSTRAINT attribute_definitions_data_type_check
        CHECK (data_type IN ('number', 'text', 'boolean', 'select'))
);

-- Same "unique while active" pattern as product_lines_brand_code_unique_active
-- — code is stable/reusable per category, but a soft-deleted definition's
-- code can be reused by a new one. Two partial indexes (not one COALESCE'd
-- index) because Postgres never treats two NULLs as equal in a unique
-- constraint — a single UNIQUE(category_id, code) would silently let two
-- global (category_id IS NULL) definitions share the same code.
CREATE UNIQUE INDEX IF NOT EXISTS attribute_definitions_category_code_unique_active
    ON attribute_definitions (category_id, code)
    WHERE deleted_at IS NULL AND category_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS attribute_definitions_global_code_unique_active
    ON attribute_definitions (code)
    WHERE deleted_at IS NULL AND category_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_attribute_definitions_category_id ON attribute_definitions (category_id);
