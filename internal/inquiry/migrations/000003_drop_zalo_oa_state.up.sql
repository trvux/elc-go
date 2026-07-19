-- The Zalo OA webhook/OAuth integration was removed entirely (Go module and
-- frontend both stopped using it — lead notifications now always go through
-- LogLeadNotifier). See internal/inquiry/infrastructure for what remains.
DROP TABLE IF EXISTS zalo_oa_followers;
DROP TABLE IF EXISTS zalo_oa_tokens;
