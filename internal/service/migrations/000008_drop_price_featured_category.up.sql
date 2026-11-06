-- Contract step of the service restructure (docs/rfc-service-restructure.md in elc-temp).
-- Services are quoted over Zalo and never show a price; nothing reads or writes these
-- columns any more (code deployed first). The category_id foreign key goes with its column.
ALTER TABLE services
    DROP COLUMN IF EXISTS original_price,
    DROP COLUMN IF EXISTS sale_price,
    DROP COLUMN IF EXISTS discount_percent,
    DROP COLUMN IF EXISTS price_display_text,
    DROP COLUMN IF EXISTS is_featured,
    DROP COLUMN IF EXISTS category_id;
