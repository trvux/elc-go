-- Product v2 core model (see docs/product-v2-design.md). Unlike 000001-000004,
-- none of these tables pre-exist in Supabase — this is genuinely new schema,
-- not an IF-NOT-EXISTS no-op against legacy data. Still written idempotently
-- (IF NOT EXISTS / DO-block-guarded ADD CONSTRAINT) for safe re-runs, same
-- style as every other migration in this project.
--
-- Nothing here removes or renames any existing `products` column — sku,
-- original_price, sale_price, discount_percent, stock_status, mpn, gtin,
-- specs, normalized_specs all stay exactly as-is so elc-tem (still reading
-- them directly) keeps working unmodified through this migration.

DO $$ BEGIN
    CREATE TYPE product_variant_stock_status AS ENUM ('in_stock', 'order_from_supplier', 'discontinued');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- "Dòng sản phẩm" — e.g. Daikin's FTF/FTKB/FTKF/FTKY/FTKZ tiers. A brand can
-- reuse the same line across categories (rare) or scope it to one, hence
-- category_id being nullable rather than required.
CREATE TABLE IF NOT EXISTS product_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    tier_rank INTEGER NOT NULL DEFAULT 0,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Partial unique (not plain) — same resurrect-on-reuse reasoning as
-- brand/category/group slugs, matched here for consistency rather than
-- inventing a third convention for a taxonomy-shaped table.
CREATE UNIQUE INDEX IF NOT EXISTS product_lines_brand_code_unique_active ON product_lines (brand_id, code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_options_product_id ON product_options (product_id);

CREATE TABLE IF NOT EXISTS product_option_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    option_id UUID NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_option_values_option_id ON product_option_values (option_id);

-- The real purchasable unit. mpn is the manufacturer's own code (required —
-- this is what customers actually search for); sku is this business's own
-- internal warehouse code (Go always fills it, auto-generated if the admin
-- leaves it blank, and it is never rendered in the public API response).
-- Both are unique only among non-deleted rows (partial index), same
-- resurrect-on-reuse reasoning already established for brand/category/group
-- slugs in this project — see docs/product-v2-design.md.
CREATE TABLE IF NOT EXISTS product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    mpn TEXT NOT NULL,
    sku TEXT NOT NULL,
    gtin TEXT,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_standalone BOOLEAN NOT NULL DEFAULT true,
    stock_status product_variant_stock_status NOT NULL DEFAULT 'in_stock',
    lead_time_days INTEGER,
    cost_price NUMERIC(15, 0),
    original_price NUMERIC(15, 0) NOT NULL DEFAULT 0,
    sale_price NUMERIC(15, 0),
    discount_percent NUMERIC(5, 2) NOT NULL DEFAULT 0,
    weight NUMERIC(10, 3),
    is_active BOOLEAN NOT NULL DEFAULT true,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_product_variants_product_id ON product_variants (product_id);
CREATE UNIQUE INDEX IF NOT EXISTS product_variants_mpn_unique_active ON product_variants (mpn) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS product_variants_sku_unique_active ON product_variants (sku) WHERE deleted_at IS NULL;
-- At most one active default variant per product.
CREATE UNIQUE INDEX IF NOT EXISTS product_variants_one_default_active ON product_variants (product_id) WHERE is_default AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_variant_option_values (
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    option_value_id UUID NOT NULL REFERENCES product_option_values(id) ON DELETE CASCADE,
    PRIMARY KEY (variant_id, option_value_id)
);

CREATE INDEX IF NOT EXISTS idx_pvov_option_value_id ON product_variant_option_values (option_value_id);

-- Split-system bundles (e.g. Daikin's dàn lạnh + dàn nóng sold as one
-- sellable "Set FTKB25" variant). role is free text, not an enum — generic
-- bundle-part label reusable for any multi-part product, not AC-specific.
CREATE TABLE IF NOT EXISTS product_variant_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    component_variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1,
    role TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_variant_components_not_self CHECK (parent_variant_id <> component_variant_id)
);

DO $$ BEGIN
    ALTER TABLE product_variant_components ADD CONSTRAINT product_variant_components_unique UNIQUE (parent_variant_id, component_variant_id);
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN duplicate_table THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_pvc_component_variant_id ON product_variant_components (component_variant_id);

-- products: marketing-level additions + write-time denormalized read cache.
-- The cache columns (default_variant_id/display_price/display_stock_status/
-- price_min/price_max) exist so list/grid queries never join to
-- product_variants — recomputed at write time whenever a variant changes,
-- same technique already proven for normalized_specs. See
-- docs/product-v2-design.md.
ALTER TABLE products ADD COLUMN IF NOT EXISTS product_line_id UUID REFERENCES product_lines(id) ON DELETE SET NULL;
ALTER TABLE products ADD COLUMN IF NOT EXISTS short_description TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS warranty_months INTEGER;
ALTER TABLE products ADD COLUMN IF NOT EXISTS warranty_terms TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS default_variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL;
ALTER TABLE products ADD COLUMN IF NOT EXISTS display_price NUMERIC(15, 0);
ALTER TABLE products ADD COLUMN IF NOT EXISTS display_stock_status TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS price_min NUMERIC(15, 0);
ALTER TABLE products ADD COLUMN IF NOT EXISTS price_max NUMERIC(15, 0);

CREATE INDEX IF NOT EXISTS idx_products_product_line_id ON products (product_line_id);
CREATE INDEX IF NOT EXISTS products_display_price_idx ON products (display_price) WHERE deleted_at IS NULL;
