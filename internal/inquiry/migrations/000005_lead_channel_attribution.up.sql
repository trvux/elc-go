-- Adds contact-channel + ads-attribution capture to inquiries. Until now
-- inquiries only ever came from the on-site form (channel implicitly
-- 'form') with no gclid/UTM captured anywhere — there was no way to tie a
-- lead back to the ad/campaign that produced it. channel records HOW the
-- visitor reached out (form/zalo/messenger/hotline), independent of
-- lead_type (WHAT they were looking at) — same "what vs how" split as
-- lead_type/sub_type, see internal/inquiry/domain/types.go's ContactChannel
-- doc comment. gclid/utm_*/ga_client_id are captured client-side from the
-- visitor's own attribution cookie (elc-temp's middleware.ts) at the
-- moment the inquiry is created. conversion_value/ads_conversion_synced_at
-- are set later, when staff mark the lead 'converted' and the outcome is
-- pushed to GA4 (see internal/inquiry/application/update_inquiry_status.go
-- — that push isn't implemented yet as of this migration, the columns are
-- added now so the later change doesn't need another migration).
ALTER TABLE inquiries
    ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'form',
    ADD COLUMN gclid VARCHAR(255),
    ADD COLUMN utm_source VARCHAR(100),
    ADD COLUMN utm_medium VARCHAR(100),
    ADD COLUMN utm_campaign VARCHAR(150),
    ADD COLUMN utm_term VARCHAR(150),
    ADD COLUMN utm_content VARCHAR(150),
    ADD COLUMN ga_client_id VARCHAR(64),
    ADD COLUMN conversion_value NUMERIC(14, 2),
    ADD COLUMN ads_conversion_synced_at TIMESTAMPTZ;

ALTER TABLE inquiries
    ADD CONSTRAINT chk_inquiry_channel CHECK (channel IN ('form', 'zalo', 'messenger', 'hotline'));

CREATE INDEX IF NOT EXISTS idx_inquiries_channel ON inquiries (channel);

-- Partial: gclid is NULL for the vast majority of rows (organic/direct
-- traffic, or leads that predate this column) — indexing only the non-null
-- subset keeps it small, for the future "find by gclid" offline-conversion
-- sync/backfill lookup.
CREATE INDEX IF NOT EXISTS idx_inquiries_gclid ON inquiries (gclid) WHERE gclid IS NOT NULL;
