-- session_id (the visitor's elc_session_id cookie, see modules/event's
-- getOrCreateSessionId in elc-temp) lets the click endpoint
-- (POST /inquiries/clicks) collapse repeated Zalo/Messenger/Hotline clicks
-- from the same visitor into one open lead per (session_id, channel)
-- instead of one row per click — a click-origin lead has no name/phone to
-- dedup by otherwise. Only populated for click-origin leads for now (the
-- on-site form already has name/phone as its identity, dedup isn't its
-- problem) — nullable, not part of any NOT NULL/CHECK constraint.
ALTER TABLE inquiries
    ADD COLUMN session_id VARCHAR(64);

-- Composite, in this column order, because the only query that uses it is
-- "find the most recent OPEN row for this session+channel"
-- (WHERE session_id = $1 AND channel = $2 AND status IN ('new','contacted')
-- ORDER BY created_at DESC LIMIT 1) — status last since it's the lowest-
-- selectivity filter of the three.
CREATE INDEX IF NOT EXISTS idx_inquiries_session_channel_status
    ON inquiries (session_id, channel, status)
    WHERE session_id IS NOT NULL;
