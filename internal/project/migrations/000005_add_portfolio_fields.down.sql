ALTER TABLE projects
    DROP COLUMN IF EXISTS client_name,
    DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS completed_at,
    DROP COLUMN IF EXISTS testimonial_quote,
    DROP COLUMN IF EXISTS testimonial_author;
