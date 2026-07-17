-- Adds a real is_hidden flag so staff-created catch-all categories (e.g.
-- "Chưa phân loại") can be excluded from public navigation/listings via a
-- boolean instead of matching on display name, which breaks if renamed.
ALTER TABLE categories ADD COLUMN is_hidden BOOLEAN NOT NULL DEFAULT false;
UPDATE categories SET is_hidden = true WHERE name ILIKE '%chưa phân loại%' AND deleted_at IS NULL;
