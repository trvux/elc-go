-- Restores the column shape only. The original column was NOT NULL with no
-- default and no real data (every row held the same placeholder UUID), so
-- there is nothing meaningful to backfill — added back nullable.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS category_id uuid;
