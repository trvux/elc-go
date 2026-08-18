-- Phase 2 (SSE streaming): a mid-stream error can cut a reply short after
-- some of it already reached the client — that partial text is persisted
-- (rather than discarded) marked incomplete, instead of pretending the full
-- answer was delivered. See docs/rfc/2026-08-18-ai-chat-phase2-sse-streaming.md §2.4.
ALTER TABLE ai_messages ADD COLUMN IF NOT EXISTS incomplete BOOLEAN NOT NULL DEFAULT false;
