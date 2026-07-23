ALTER TABLE branches
    DROP COLUMN IF EXISTS province_code,
    DROP COLUMN IF EXISTS province_name,
    DROP COLUMN IF EXISTS ward_code,
    DROP COLUMN IF EXISTS ward_name,
    DROP COLUMN IF EXISTS postal_code;
