-- Replaces the is_published bool with an explicit lifecycle state machine
-- (draft -> proposed -> published -> archived) now that products go through
-- an employee-submits / owner-approves workflow instead of a single toggle.
-- rejection_reason carries feedback when an owner sends a proposed product
-- back to draft — there is no separate "rejected" state, it folds into draft
-- (the only actionable next step from a rejection is revise-and-resubmit).
CREATE TYPE product_status AS ENUM ('draft', 'proposed', 'published', 'archived');

ALTER TABLE products ADD COLUMN status product_status NOT NULL DEFAULT 'draft';
ALTER TABLE products ADD COLUMN rejection_reason text;

UPDATE products SET status = (CASE WHEN is_published THEN 'published' ELSE 'draft' END)::product_status;

-- "Public read" is a leftover Supabase-era RLS policy gating on
-- is_published alone; it's dead weight, not load-bearing — Postgres OR's
-- permissive policies for the same command together, and the other
-- existing policy (public_read, deleted_at IS NULL only) already permits
-- every row this one does, so dropping it changes no actual visibility. It
-- must go before is_published can be dropped (Postgres blocks dropping a
-- column an active policy still references).
DROP POLICY IF EXISTS "Public read" ON products;

ALTER TABLE products DROP COLUMN is_published;

-- labels (Mới về/Nổi bật/Bán chạy/Giảm giá) dropped with no replacement:
-- Sale/New are computable from display_price/created_at if ever needed for
-- display, Best Seller had no real sales data behind it, and Nổi bật already
-- duplicated is_featured. Only 5/197 rows had any value at migration time.
ALTER TABLE products DROP COLUMN labels;
