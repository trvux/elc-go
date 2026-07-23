-- Shipping zones: admin-configured delivery fee/time per area, since the
-- shop delivers with its own fleet (no third-party carrier API).
--
-- provinces is seeded with the current (post-July-2025 merger) list of
-- Vietnam's 34 provinces/centrally-run cities, compiled from public sources
-- at the time this migration was written. Vietnam's administrative
-- boundaries changed recently and this list is NOT guaranteed authoritative
-- — review/correct it via the admin "Khu vực giao hàng" screen (which reads
-- straight from this table) rather than editing this migration after it has
-- shipped.
CREATE TABLE IF NOT EXISTS provinces (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shipping_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    fee_vnd BIGINT NOT NULL DEFAULT 0,
    min_days INTEGER NOT NULL,
    max_days INTEGER NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Only one non-deleted zone may be the global default (used as the fallback
-- for provinces with no zone configured, and as the static value shown in
-- product-page JSON-LD).
CREATE UNIQUE INDEX IF NOT EXISTS shipping_zone_single_default
    ON shipping_zones ((true)) WHERE is_default AND deleted_at IS NULL;

-- A province can belong to more than one zone (a keyword-zone plus a
-- catch-all zone for the same province) — see internal/shippingzone/application/lookup_zone.go.
CREATE TABLE IF NOT EXISTS shipping_zone_provinces (
    zone_id UUID NOT NULL REFERENCES shipping_zones(id) ON DELETE CASCADE,
    province_code TEXT NOT NULL REFERENCES provinces(code) ON DELETE CASCADE,
    PRIMARY KEY (zone_id, province_code)
);
CREATE INDEX IF NOT EXISTS idx_shipping_zone_provinces_province ON shipping_zone_provinces (province_code);

CREATE TABLE IF NOT EXISTS shipping_zone_keywords (
    zone_id UUID NOT NULL REFERENCES shipping_zones(id) ON DELETE CASCADE,
    keyword TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shipping_zone_keywords_zone ON shipping_zone_keywords (zone_id);

INSERT INTO provinces (code, name) VALUES
    ('an-giang', 'An Giang'),
    ('bac-ninh', 'Bắc Ninh'),
    ('ca-mau', 'Cà Mau'),
    ('cao-bang', 'Cao Bằng'),
    ('can-tho', 'Cần Thơ'),
    ('da-nang', 'Đà Nẵng'),
    ('dak-lak', 'Đắk Lắk'),
    ('dien-bien', 'Điện Biên'),
    ('dong-nai', 'Đồng Nai'),
    ('dong-thap', 'Đồng Tháp'),
    ('gia-lai', 'Gia Lai'),
    ('ha-noi', 'Hà Nội'),
    ('ha-tinh', 'Hà Tĩnh'),
    ('hai-phong', 'Hải Phòng'),
    ('hung-yen', 'Hưng Yên'),
    ('khanh-hoa', 'Khánh Hòa'),
    ('lai-chau', 'Lai Châu'),
    ('lam-dong', 'Lâm Đồng'),
    ('lang-son', 'Lạng Sơn'),
    ('lao-cai', 'Lào Cai'),
    ('nghe-an', 'Nghệ An'),
    ('ninh-binh', 'Ninh Bình'),
    ('phu-tho', 'Phú Thọ'),
    ('quang-ngai', 'Quảng Ngãi'),
    ('quang-ninh', 'Quảng Ninh'),
    ('quang-tri', 'Quảng Trị'),
    ('son-la', 'Sơn La'),
    ('tay-ninh', 'Tây Ninh'),
    ('thai-nguyen', 'Thái Nguyên'),
    ('thanh-hoa', 'Thanh Hóa'),
    ('thanh-pho-ho-chi-minh', 'Thành phố Hồ Chí Minh'),
    ('tuyen-quang', 'Tuyên Quang'),
    ('vinh-long', 'Vĩnh Long'),
    ('hue', 'Huế')
ON CONFLICT (code) DO NOTHING;

-- Default global zone so LookupZone always has a fallback even before an
-- admin configures anything province-specific. Values are placeholders —
-- correct them via the admin screen for the shop's real baseline fee/time.
INSERT INTO shipping_zones (name, fee_vnd, min_days, max_days, is_default)
VALUES ('Toàn quốc (mặc định)', 0, 3, 7, true)
ON CONFLICT DO NOTHING;
