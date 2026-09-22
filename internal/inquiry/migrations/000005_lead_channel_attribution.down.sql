DROP INDEX IF EXISTS idx_inquiries_gclid;
DROP INDEX IF EXISTS idx_inquiries_channel;

ALTER TABLE inquiries DROP CONSTRAINT IF EXISTS chk_inquiry_channel;

ALTER TABLE inquiries
    DROP COLUMN IF EXISTS channel,
    DROP COLUMN IF EXISTS gclid,
    DROP COLUMN IF EXISTS utm_source,
    DROP COLUMN IF EXISTS utm_medium,
    DROP COLUMN IF EXISTS utm_campaign,
    DROP COLUMN IF EXISTS utm_term,
    DROP COLUMN IF EXISTS utm_content,
    DROP COLUMN IF EXISTS ga_client_id,
    DROP COLUMN IF EXISTS conversion_value,
    DROP COLUMN IF EXISTS ads_conversion_synced_at;
