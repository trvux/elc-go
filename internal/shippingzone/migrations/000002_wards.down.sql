DROP TABLE IF EXISTS shipping_zone_wards;
DROP TABLE IF EXISTS wards;

CREATE TABLE IF NOT EXISTS shipping_zone_keywords (
    zone_id UUID NOT NULL REFERENCES shipping_zones(id) ON DELETE CASCADE,
    keyword TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shipping_zone_keywords_zone ON shipping_zone_keywords (zone_id);
