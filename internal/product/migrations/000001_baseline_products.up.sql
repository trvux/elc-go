-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- the CREATE TABLE itself is a no-op against the real DB. Only useful for
-- spinning up a fresh scratch Postgres for local testing. The two NEW
-- columns (search_vector, normalized_specs) and their indexes below DO need
-- to actually run against the real DB — this migration's real job.
--
-- pg_trgm and unaccent are already enabled on the live DB (confirmed via
-- `SELECT extname FROM pg_extension` before writing this) — CREATE EXTENSION
-- IF NOT EXISTS is harmless but not required there; included so a fresh
-- scratch DB also gets them.
--
-- The triggers `trg_product_slug_registry` (syncs into the shared
-- slug_registry table — same shared platform infrastructure as
-- trg_brand_slug_registry/trigger_sync_service_slug) and
-- `update_products_modtime` (sets updated_at on every UPDATE) already exist
-- on the live DB and are intentionally NOT recreated here — see
-- docs/catalog.md.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "unaccent";

DO $$ BEGIN
    CREATE TYPE product_condition AS ENUM ('new', 'used');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    sku TEXT NOT NULL,
    slug TEXT NOT NULL,
    description JSONB NOT NULL DEFAULT '{}',
    specs JSONB NOT NULL DEFAULT '{}',
    images TEXT[] DEFAULT '{}',
    labels TEXT[] DEFAULT '{}',
    original_price NUMERIC(15, 0) DEFAULT 0,
    sale_price NUMERIC(15, 0),
    discount_percent NUMERIC(5, 2) DEFAULT 0,
    is_featured BOOLEAN DEFAULT false,
    is_published BOOLEAN DEFAULT true,
    order_index INTEGER DEFAULT 0,
    stock_status TEXT DEFAULT 'in_stock',
    condition product_condition NOT NULL DEFAULT 'new',
    meta_title TEXT,
    meta_description TEXT,
    mpn TEXT,
    gtin TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Postgres has no `ADD CONSTRAINT IF NOT EXISTS` for a plain UNIQUE
-- constraint, so this needs the same exception-catching DO block as the enum
-- type above to stay a genuine no-op against the real DB, where
-- products_sku_key already exists (confirmed via `\d products`). A UNIQUE
-- constraint is backed by an index of the same name, so the actual error
-- raised for a name collision is duplicate_table (42P07), not
-- duplicate_object (42710) — catch both to be safe.
DO $$ BEGIN
    ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN duplicate_table THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS products_slug_unique_active ON products (slug) WHERE deleted_at IS NULL;

-- normalized_specs holds the write-time-computed "UILabel::Value" facet
-- strings (see internal/catalog/domain/spec_normalizer.go) — matched at
-- query time with a plain array overlap (&&), never recomputed on read.
ALTER TABLE products ADD COLUMN IF NOT EXISTS normalized_specs TEXT[] DEFAULT '{}';

CREATE INDEX IF NOT EXISTS products_normalized_specs_idx ON products USING gin (normalized_specs);

-- unaccent() is STABLE, not IMMUTABLE (Postgres considers it possibly
-- sensitive to dictionary changes), so it cannot be used directly inside a
-- GENERATED ALWAYS AS expression — Postgres rejects that with "generation
-- expression is not immutable" (hit this for real running this migration
-- against the live DB before adding the wrapper below). The standard,
-- widely-used fix is a thin SQL wrapper function explicitly marked
-- IMMUTABLE — safe in practice because the 'unaccent' dictionary this
-- project uses is never altered at runtime. Query-time uses of unaccent()
-- elsewhere (internal/catalog/infrastructure/postgres_repository.go's search
-- WHERE/ORDER BY) don't need this wrapper — the immutability requirement
-- only applies to generated columns and index expressions, not plain query
-- expressions.
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

-- search_vector covers name + sku only (not specs/description) — full-text
-- search here is deliberately narrow in scope; spec filtering is a separate
-- exact-facet mechanism via normalized_specs, not part of this tsvector. Uses
-- the 'simple' text search config (no language-specific stemming — Vietnamese
-- isn't a supported stemming config) combined with unaccent so accented and
-- unaccented queries both match. See docs/catalog.md.
ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', immutable_unaccent(coalesce(name, '') || ' ' || coalesce(sku, '')))) STORED;

CREATE INDEX IF NOT EXISTS products_search_vector_idx ON products USING gin (search_vector);

-- Trigram index on name backs the pg_trgm similarity() fallback used
-- alongside search_vector for fuzzy/partial matches full-text alone misses.
CREATE INDEX IF NOT EXISTS products_name_trgm_idx ON products USING gin (name gin_trgm_ops);
