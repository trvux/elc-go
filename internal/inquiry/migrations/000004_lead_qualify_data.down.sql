ALTER TABLE inquiries DROP CONSTRAINT IF EXISTS chk_inquiry_lead_type;
DROP INDEX IF EXISTS idx_inquiries_lead_type;

ALTER TABLE inquiries
    DROP COLUMN IF EXISTS lead_type,
    DROP COLUMN IF EXISTS sub_type,
    DROP COLUMN IF EXISTS qualify_data,
    DROP COLUMN IF EXISTS attachments;
