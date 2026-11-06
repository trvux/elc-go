-- Restores the columns (empty). The dropped data is not recoverable from here; use a backup.
ALTER TABLE services
    ADD COLUMN IF NOT EXISTS original_price bigint,
    ADD COLUMN IF NOT EXISTS sale_price bigint,
    ADD COLUMN IF NOT EXISTS discount_percent integer,
    ADD COLUMN IF NOT EXISTS price_display_text text,
    ADD COLUMN IF NOT EXISTS is_featured boolean DEFAULT false,
    ADD COLUMN IF NOT EXISTS category_id uuid REFERENCES categories(id) ON DELETE SET NULL;
