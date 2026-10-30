-- province_code / ward_code pointed at the shippingzone module's provinces/
-- wards reference tables, which were removed. The denormalized
-- province_name / ward_name / postal_code columns (what the SEO PostalAddress
-- actually reads) stay. Applied in a deploy AFTER the code stopped selecting
-- these columns.
ALTER TABLE branches
    DROP COLUMN IF EXISTS province_code,
    DROP COLUMN IF EXISTS ward_code;
