-- Which product categories a service applies to (e.g. "Lắp đặt máy lạnh" for every
-- air-conditioner category). Drives the "Lắp đặt và dịch vụ đi kèm" block on product
-- pages; edited per service in admin. Deleting either side removes the link.
CREATE TABLE IF NOT EXISTS service_categories (
    service_id  UUID NOT NULL REFERENCES services(id)   ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (service_id, category_id)
);
CREATE INDEX IF NOT EXISTS service_categories_category_idx ON service_categories (category_id);
