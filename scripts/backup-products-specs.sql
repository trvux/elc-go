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
CREATE TABLE IF NOT EXISTS products_specs_pre_migration_backup AS
SELECT
    id AS product_id,
    category_id,
    name,
    specs,
    created_at,
    updated_at,
    now() AS backed_up_at
FROM products
WHERE deleted_at IS NULL;

SELECT count(*) AS backed_up_products FROM products_specs_pre_migration_backup;
