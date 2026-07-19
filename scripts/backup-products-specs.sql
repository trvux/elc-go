-- One-off pre-migration snapshot of the legacy free-text products.specs
-- column, taken right before cmd/migrate-specs-to-attributes runs. Purpose:
-- let a human (or a follow-up SQL query) reconcile old schema vs new
-- structured attribute_definitions/product_attribute_values after the
-- migration — never relies on memory of what specs looked like "before".
--
-- IF NOT EXISTS + no DROP: idempotent on purpose. First deploy captures the
-- true pre-migration state; any later re-deploy is a safe no-op that does
-- NOT overwrite the original snapshot with already-migrated data. Same
-- sentinel-style idempotency as scripts/migrate-postgres.sh's
-- .postgres-migrated gate, just via table existence instead of a file.
--
-- 2026-07-19: wrapped in a DO block with EXECUTE (found via cutover
-- rehearsal against a production backup) — plain
-- "CREATE TABLE IF NOT EXISTS ... AS SELECT ... specs ..." still parses/
-- validates the SELECT even when the table already exists (Postgres
-- doesn't short-circuit CTAS before type-checking), so this broke outright
-- once products.specs got dropped by a later migration in the same deploy
-- chain (internal/product/migrations/000010_drop_seo_column and friends).
-- The snapshot's entire useful window was before that column drop; on any
-- environment reaching this point today the table already exists from
-- when this ran historically (both dev and production have it) — dynamic
-- SQL inside EXECUTE is opaque to the planner until the IF is true, so the
-- now-invalid SELECT never gets evaluated once the table is already there.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables
        WHERE schemaname = 'public' AND tablename = 'products_specs_pre_migration_backup'
    ) THEN
        EXECUTE '
            CREATE TABLE products_specs_pre_migration_backup AS
            SELECT
                id AS product_id,
                category_id,
                name,
                specs,
                created_at,
                updated_at,
                now() AS backed_up_at
            FROM products
            WHERE deleted_at IS NULL
        ';
    END IF;
END $$;

SELECT count(*) AS backed_up_products FROM products_specs_pre_migration_backup;
