-- Dàn lạnh/dàn nóng/mặt nạ "Kích thước" values are stored as bare
-- "H x W x D" triples (e.g. "990 x 940 x 320") with no order indicated,
-- forcing installers to guess which number is height vs depth when
-- planning ceiling/wall cutouts. Confirmed H x W x D against Daikin's own
-- published spec sheets (matches user's manufacturer-spec audit,
-- 2026-07-18). "Kích thước ống đồng Gas" is a diameter, not H x W x D, so
-- it gets a "Ø" marker instead.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-kich-thuoc-dimension-order.sql

UPDATE attribute_definitions SET name = 'Kích thước (Cao x Rộng x Sâu)', updated_at = now()
WHERE code IN ('kich_thuoc_dan_lanh', 'kich_thuoc_dan_nong', 'kich_thuoc_mat_na');

UPDATE attribute_definitions SET name = 'Kích thước ống đồng Gas (Ø)', updated_at = now()
WHERE code = 'kich_thuoc_ong_dong_gas';
