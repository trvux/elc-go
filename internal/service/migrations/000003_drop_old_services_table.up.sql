-- old_services is a leftover from the pre-service-group table rename: 0 code
-- references anywhere in Go or elc-tem. Content audited before drop (5 rows:
-- 4 real published articles confirmed stale/not needed, 1 soft-deleted test
-- row) — user confirmed drop. A pg_dump backup of the data was taken outside
-- git before running this migration.
DROP TABLE IF EXISTS old_services;
