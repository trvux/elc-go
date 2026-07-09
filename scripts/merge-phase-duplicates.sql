-- One-off data-cleanup script — merges 17 pairs of Daikin ducted/cassette/
-- ceiling AC products that are the same physical indoor unit listed twice
-- (once per electrical phase, "1 pha"/"3 pha", different outdoor unit +
-- price) into ONE product each, with a new "Điện áp" option and 2 variants.
-- Every pair below was individually reviewed against real DB data on
-- 2026-07-08 (not derived by blind regex) — see
-- docs/catalog-audit-2026-07-08.md finding 1. One pair (FDMNQ36MV1) had a
-- mislabeled HP in its "3 pha" row's name ("3.5HP" should be "4HP", same
-- indoor unit, same price as the correctly-labeled "4HP (1 pha)" row) —
-- resolves itself automatically since only the survivor's (already correct)
-- name is kept.
--
-- Effect per pair:
--   1. Survivor = the "1 pha" product row. Its name loses the "( 1 pha )"
--      suffix. Its EXISTING variant (from today's earlier backfill) is kept
--      as-is and linked to a new "Điện áp: 1 pha" option value.
--   2. A new variant is added under the survivor using the "3 pha" product's
--      mpn/sku/price/stock, linked to "Điện áp: 3 pha". Its `mpn` is
--      temporarily set to the old combined sku string (guaranteed unique —
--      was a unique product sku) since the real indoor-unit mpn is already
--      taken by the survivor's variant; cmd/bundle-split-variant-sku (next
--      script) resolves this properly by splitting out the real per-part
--      MPNs.
--   3. The "3 pha" product row AND its own existing variant are soft-deleted.
--
-- Run once against a DB with the product v2 schema + today's variant
-- backfill already applied:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/merge-phase-duplicates.sql
--
-- NOT safely re-runnable blind (unlike the additive scripts today) — running
-- twice would try to re-soft-delete already-gone rows (harmless no-ops) but
-- would also try to re-insert a second "3 pha" variant under an already-
-- merged survivor. Each product_line/option-value insert IS idempotent
-- (partial unique indexes), so the real risk is only the variant re-insert;
-- guarded below by checking the survivor doesn't already have 2 variants.

BEGIN;

CREATE TEMP TABLE phase_pairs (survivor_id uuid, loser_id uuid) ON COMMIT DROP;
INSERT INTO phase_pairs (survivor_id, loser_id) VALUES
    ('adc4d3eb-7fa5-415c-84e6-93e758e16a89', '5625fb65-2723-49fa-8142-553c34946023'), -- FBA100BVMA9
    ('2aeaeb34-b939-4e3f-8ba2-8f2099508a2c', '62b8a5ef-1c1d-4e66-8967-1b3177bcb117'), -- FBA125BVMA9
    ('0bf9f4f1-ca83-45d1-8d72-5a19f26ab5c9', '7b4b0338-cd6a-472a-bba3-7836c092631f'), -- FBA140BVMA9
    ('c474b377-df01-4051-8aef-dce81fef35d7', 'a22f3267-9d9a-4a63-aa0a-c637a075b2c5'), -- FBA71BVMA9
    ('ec9a4ed6-2af1-4fa5-9493-94ebb5637f53', 'bb3542fd-f788-4d85-96be-6b6770920088'), -- FBFC100DVM9
    ('1ec73519-aec2-41b9-8dd6-94ba5f64865a', '9f053214-4a6b-43b5-8356-45d2362693c7'), -- FBFC71DVM9
    ('501f9836-7c7e-403f-9bcb-22946bf65596', '6f708546-0e12-446c-9ce5-4ae5e4c70c21'), -- FBFC85DVM9
    ('d5afde8f-a0a0-4027-8a24-6b96da396a4b', '944deab6-4817-4ecb-9fdb-5e41b14b9189'), -- FCF100CVM
    ('9f20d412-3dfc-47b7-a189-0f3a443ce977', 'c173e46f-7daf-4edd-bac4-e12e56edfd02'), -- FCFC100DVM
    ('548cb3c4-f430-44e8-a0b8-01b8fbc4fd07', '48f6aaf5-9a7e-4290-8e81-b8a93c6835fb'), -- FCFC71DVM
    ('7bf18423-46e6-4e3f-8d63-14e5f7bac4ba', '644c93eb-d51d-4b7f-84cb-3ba4c5732462'), -- FCFC85DVM
    ('9ac01365-e729-4e51-ba99-881feb53e0c2', '63cb97e0-1fc4-4c2b-a11f-d23b9d03474a'), -- FDMNQ26MV1
    ('5863d6bd-ec72-41fa-8bdc-65d65fb62850', 'b19d2490-9505-4e05-843c-de0bae1b1ba0'), -- FDMNQ30MV1
    ('82e5a304-fb6c-47ae-8497-c539480a5f5b', '1d487ed3-d0fb-4b78-93ec-342a7d89249c'), -- FDMNQ36MV1
    ('3afc0c09-bd77-4848-b8bb-e8d164efb96d', '12cb11fd-6c8e-46c9-b4fc-0d4782cd022e'), -- FHA100CVMV
    ('f02aebfa-1f29-48ee-8508-3cb36acf9c7c', 'b841b9cc-44bf-4b92-a36e-c3d06c6211b8'), -- FHA125CVMA
    ('01fcd580-c064-4156-83fb-3a7a40f525b7', 'a5fd9166-ef7d-44ee-a1e9-5467e9cd37e0'); -- FHA140CVMA

DO $$
DECLARE
    pair RECORD;
    opt_id uuid;
    val_1pha uuid;
    val_3pha uuid;
    survivor_variant_id uuid;
    -- Reads from product_variants, not products — migration 000007
    -- (variant-only pricing) already dropped products.sku/original_price/
    -- sale_price/discount_percent/stock_status by the time this runs; the
    -- loser's real values now live on its own (already-backfilled) default
    -- variant instead.
    loser_variant product_variants%ROWTYPE;
BEGIN
    FOR pair IN SELECT * FROM phase_pairs LOOP
        -- Skip pairs already processed by a prior run of this script.
        IF (SELECT count(*) FROM product_variants WHERE product_id = pair.survivor_id AND deleted_at IS NULL) >= 2 THEN
            CONTINUE;
        END IF;

        SELECT * INTO loser_variant FROM product_variants
        WHERE product_id = pair.loser_id AND deleted_at IS NULL AND is_default = true LIMIT 1;

        -- 1. Strip the phase suffix from the survivor's name.
        UPDATE products
        SET name = trim(regexp_replace(name, '\s*\(\s*1\s*pha\s*\)\s*$', '', 'i'))
        WHERE id = pair.survivor_id;

        -- 2. Create the "Điện áp" option + its 2 values.
        INSERT INTO product_options (product_id, name, order_index)
        VALUES (pair.survivor_id, 'Điện áp', 0)
        RETURNING id INTO opt_id;

        INSERT INTO product_option_values (option_id, value, order_index) VALUES (opt_id, '1 pha', 0) RETURNING id INTO val_1pha;
        INSERT INTO product_option_values (option_id, value, order_index) VALUES (opt_id, '3 pha', 1) RETURNING id INTO val_3pha;

        -- 3. Link the survivor's existing (already-backfilled) variant to "1 pha".
        SELECT id INTO survivor_variant_id FROM product_variants
        WHERE product_id = pair.survivor_id AND deleted_at IS NULL LIMIT 1;

        INSERT INTO product_variant_option_values (variant_id, option_value_id)
        VALUES (survivor_variant_id, val_1pha);

        -- 4. Retire the loser product row and its own (now-redundant) variant
        -- FIRST — its sku/mpn is about to be reused by the new variant below,
        -- and both have a unique-among-ACTIVE-rows index, so the old row
        -- must be soft-deleted before the new one can take the same value.
        UPDATE product_variants SET deleted_at = now() WHERE product_id = pair.loser_id AND deleted_at IS NULL;
        UPDATE products SET deleted_at = now() WHERE id = pair.loser_id;

        -- 5. New variant for the "3 pha" data, linked to "3 pha". mpn is
        -- temporarily the loser's old combined sku (unique, cleaned up by
        -- the bundle-split script next). stock_status is already the
        -- correct enum value (product_variants, not the old products text
        -- column), no remapping needed.
        INSERT INTO product_variants (
            product_id, mpn, sku, is_default, is_standalone, stock_status,
            original_price, sale_price, discount_percent, is_active, order_index
        ) VALUES (
            pair.survivor_id, loser_variant.sku, loser_variant.sku, false, true,
            loser_variant.stock_status,
            loser_variant.original_price, NULLIF(loser_variant.sale_price, 0), loser_variant.discount_percent, true, 1
        )
        RETURNING id INTO survivor_variant_id; -- reused var, now holds the NEW variant's id

        INSERT INTO product_variant_option_values (variant_id, option_value_id)
        VALUES (survivor_variant_id, val_3pha);
    END LOOP;
END $$;

-- Recompute each survivor's denormalized display cache now that it has 2
-- variants (same technique used by the Go repository's RecomputeDisplayCache,
-- reimplemented here in SQL since this is a one-off script, not a Go path).
WITH dv AS (
    SELECT DISTINCT ON (v.product_id) v.product_id, v.id, v.sale_price, v.original_price, v.stock_status
    FROM product_variants v
    JOIN phase_pairs pp ON pp.survivor_id = v.product_id
    WHERE v.deleted_at IS NULL AND v.is_default = true
), agg AS (
    SELECT v.product_id,
           MIN(COALESCE(v.sale_price, v.original_price)) AS price_min,
           MAX(COALESCE(v.sale_price, v.original_price)) AS price_max
    FROM product_variants v
    JOIN phase_pairs pp ON pp.survivor_id = v.product_id
    WHERE v.deleted_at IS NULL AND v.is_active = true
    GROUP BY v.product_id
)
UPDATE products p
SET default_variant_id = dv.id,
    display_price = COALESCE(dv.sale_price, dv.original_price),
    display_stock_status = dv.stock_status::text,
    price_min = agg.price_min,
    price_max = agg.price_max
FROM dv JOIN agg ON agg.product_id = dv.product_id
WHERE p.id = dv.product_id;

-- Report before commit.
SELECT count(*) AS products_merged_away FROM phase_pairs pp JOIN products p ON p.id = pp.loser_id WHERE p.deleted_at IS NOT NULL;
SELECT p.name, p.slug, count(v.id) AS variant_count
FROM products p
JOIN phase_pairs pp ON pp.survivor_id = p.id
JOIN product_variants v ON v.product_id = p.id AND v.deleted_at IS NULL
GROUP BY p.id, p.name, p.slug
ORDER BY p.name;

COMMIT;
