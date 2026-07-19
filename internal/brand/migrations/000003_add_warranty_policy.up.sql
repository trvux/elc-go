-- elc is a reseller (Daikin/LG/Menred/Acis), not the manufacturer — it
-- brings a customer's unit back to the manufacturer for warranty service,
-- it doesn't run its own service center. That process/policy is set per
-- brand (each manufacturer has its own requirements), not per product, so
-- it lives here rather than as free text repeated on every Product row —
-- see internal/product/migrations/000014_drop_condition_and_warranty.up.sql.
ALTER TABLE brands ADD COLUMN warranty_policy text;
