DROP INDEX IF EXISTS idx_inquiries_session_channel_status;

ALTER TABLE inquiries
    DROP COLUMN IF EXISTS session_id;
