-- Real structured location (tỉnh/thành + phường/xã, reusing the same
-- reference codes as internal/shippingzone's provinces/wards tables) plus a
-- real postal code, replacing the old approach of guessing locality/region/
-- postcode by regex-splitting the free-text `address` column (see
-- shared/lib/seo-schema.ts's parseAddress on the frontend). No FK to
-- provinces/wards here: those tables belong to a different module migrated
-- independently, and `branch` sorts alphabetically before `shippingzone` in
-- migrate-up-all's directory loop, so a hard FK would fail against a fresh
-- database. `address` keeps its column name but now only needs to carry the
-- street-level remainder (số nhà, tên đường, hẻm).
ALTER TABLE branches
    ADD COLUMN IF NOT EXISTS province_code TEXT,
    ADD COLUMN IF NOT EXISTS province_name TEXT,
    ADD COLUMN IF NOT EXISTS ward_code TEXT,
    ADD COLUMN IF NOT EXISTS ward_name TEXT,
    ADD COLUMN IF NOT EXISTS postal_code TEXT;

-- One-time backfill for the 4 real branches that exist today, using values
-- already looked up and verified this session (Quyết định 2334/QĐ-BKHCN
-- postal codes, confirmed against two independent sources).
UPDATE branches SET
    province_code = 'thanh-pho-ho-chi-minh',
    province_name = 'Thành phố Hồ Chí Minh',
    ward_code = '26884',
    ward_name = 'Phường Gò Vấp',
    postal_code = '71424'
WHERE slug = 'van-phong' AND deleted_at IS NULL;

UPDATE branches SET
    province_code = 'thanh-pho-ho-chi-minh',
    province_name = 'Thành phố Hồ Chí Minh',
    ward_code = '26767',
    ward_name = 'Phường An Phú Đông',
    postal_code = '71516'
WHERE slug IN ('tru-so-van-phong', 'kho-bai-ky-thuat', 'kho-hang') AND deleted_at IS NULL;
