# Backup & restore

- **Status**: Active — flow redesigned and restore-tested end-to-end 2026-09-01
  (previous ad-hoc backups accumulated since April with no retention policy
  and were never restore-tested; all deleted, this is the fresh start).

## What gets backed up, and why

| Layer | Protects against | Retention |
|---|---|---|
| Docker volume `elc-postgres` | Container/image churn | N/A — live data, not a backup |
| Local dump on VPS (`/var/www/elc-go/backups/postgres-*.dump`) | Fast same-day restore, no network round trip | `BACKUP_RETENTION_DAYS` (default 30) |
| Off-site copy on Cloudflare R2 (`postgres-backups/`) | VPS itself disappearing (disk failure, `docker volume rm`, host lost) | Same window, pruned by `scripts/backup-postgres.sh` |

This is the standard **3-2-1** shape: 3 copies (volume + local dump + R2),
2 storage types (Docker volume / plain files), 1 off-site (R2 is a
completely separate infra from the VPS).

`scripts/sync-dev-db-from-prod.sh` is **not** a backup — it's a one-directional,
on-demand refresh of a developer's local DB from prod, keeps no history.

## How it runs

Cron on the VPS, `0 3 * * *` (3am `Asia/Ho_Chi_Minh`, i.e. **not** midnight):

```
0 3 * * * /var/www/elc-go/scripts/backup-postgres.sh >> /var/www/elc-go/backups/backup.log 2>&1
```

Each run: full `pg_dump` (custom format, no schema restriction — see gotcha
below) → integrity-check via `pg_restore --list` (fails loudly, does not
upload or prune if the dump can't even be listed) → upload to R2 → prune
local + R2 copies older than `BACKUP_RETENTION_DAYS`.

R2 pruning is done by parsing the date out of the filename
(`postgres-YYYYMMDD-HHMMSS.dump`), not a native R2 lifecycle rule — the R2
API token in use is object read/write only, not bucket-admin
(`GetBucketLifecycleConfiguration` returns `AccessDenied`). If the token
ever gets bucket-admin scope, switch to a real lifecycle rule instead —
it stays correct even on a night this script fails to run at all.

## Restoring

```sh
# 1. New target database
docker exec elc-postgres psql -U elc -d postgres -c "CREATE DATABASE <target>;"

# 2. REQUIRED, easy to miss: production's `elc` database has
#    `ALTER DATABASE elc SET search_path = public, extensions` configured.
#    This is database-level config, not schema/data — pg_dump/pg_restore
#    never capture it. Without this, restore fails on the very first table
#    that uses immutable_unaccent() (see gotcha below).
docker exec elc-postgres psql -U elc -d postgres -c \
  "ALTER DATABASE <target> SET search_path = public, extensions;"

# 3. Copy the dump into the postgres container first — it can't see the
#    host's bind-mounted /var/www/elc-go/backups/ path directly.
docker cp <path-to-dump> elc-postgres:/tmp/restore.dump

# 4. Restore
docker exec elc-postgres pg_restore -U elc -d <target> \
  --no-owner --no-privileges /tmp/restore.dump

# 5. Sanity check row counts against a table or two you know roughly the
#    size of, then drop the target database once satisfied (or point the
#    app's DATABASE_URL at it, if this was a real recovery).
```

## Gotcha: `immutable_unaccent` needed schema-qualifying

Found 2026-09-01 by actually restore-testing a backup into a scratch
database — the exact kind of check this system had never had before.

`internal/product/migrations/000001_baseline_products.up.sql` originally
defined:

```sql
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
```

The inner `unaccent(...)` call is unqualified, relying on the caller's
`search_path` to resolve it. This worked fine in normal production use
because of the `ALTER DATABASE elc SET search_path = public, extensions`
above — but a fresh restore target doesn't have that set (step 2 exists
because of this), and even with it set, `pg_restore`'s own session doesn't
reliably inherit a database's configured default the same way an app
connection does. Either way, `CREATE TABLE products` failed immediately
(the `search_vector` generated column calls `immutable_unaccent`, which
Postgres validates at table-creation time).

Fixed in `internal/product/migrations/000020_fix_immutable_unaccent_schema_qualify.up.sql`
by schema-qualifying the inner call (`public.unaccent(...)`), which makes
it correct regardless of caller search_path — general best practice for
anything referenced inside a function body. Step 2 above is still worth
doing regardless (belt and suspenders, and other future functions may not
be as careful).

## Verifying a backup is actually restorable

Not automated (would need a disposable Postgres instance in CI, judged not
worth the complexity at this DB's size — ~2MB). Do it by hand occasionally
using the restore procedure above against a scratch database on the same
VPS (`elc-postgres` has spare capacity for a throwaway DB; drop it when
done). The whole point of a backup nobody has ever restored is that you
don't find out it's broken until the day you need it most — this is
exactly how the `immutable_unaccent` gotcha above was found.
