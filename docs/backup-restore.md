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

## Gotcha: `immutable_unaccent` needed full schema-qualifying

Found 2026-09-01 by actually restore-testing a backup into a scratch
database — the exact kind of check this system had never had before. Took
two migrations to fully fix; both left in place as separate history rather
than amended, matching this repo's convention of not rewriting shipped
migrations.

`internal/product/migrations/000001_baseline_products.up.sql` originally
defined:

```sql
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
```

Two things here are unqualified and depend on the caller's `search_path`:
the `unaccent(...)` function call itself, and the `'unaccent'` argument —
a string literal that gets implicitly cast to `regdictionary`, referring to
the text search dictionary object created by `CREATE EXTENSION unaccent`.
This worked fine in normal production use because of
`ALTER DATABASE elc SET search_path = public, extensions` above. A fresh
restore target doesn't have that (step 2 above exists because of this) —
but even after setting it, `pg_restore`'s own session still failed on the
dictionary lookup specifically, despite a plain `psql` connection to the
exact same database correctly showing `search_path = public, extensions`
via `SHOW search_path`. `pg_restore` apparently doesn't resolve unqualified
regdictionary literals the same way a normal connection does, independent
of the database's configured default.

- `000020_fix_immutable_unaccent_schema_qualify` — schema-qualified the
  function call: `public.unaccent(...)`. Fixed the "function does not
  exist" error, but restore then failed one step further in with
  `text search dictionary "unaccent" does not exist`.
- `000021_fully_qualify_immutable_unaccent_dictionary` — schema-qualified
  the dictionary argument too: `'public.unaccent'::regdictionary`. Verified
  working even with `search_path` forced down to just `pg_catalog` (i.e.
  with zero help from search_path at all) before trusting it.

Net effect: `immutable_unaccent` no longer depends on search_path in any
way, for either the function or the dictionary it calls — general best
practice for anything referenced inside a function body. Step 2 above is
still worth doing regardless (belt and suspenders, and other functions
added later may not be as careful).

## Verifying a backup is actually restorable

Not automated (would need a disposable Postgres instance in CI, judged not
worth the complexity at this DB's size — ~2MB). Do it by hand occasionally
using the restore procedure above against a scratch database on the same
VPS (`elc-postgres` has spare capacity for a throwaway DB; drop it when
done). The whole point of a backup nobody has ever restored is that you
don't find out it's broken until the day you need it most — this is
exactly how the `immutable_unaccent` gotcha above was found.
