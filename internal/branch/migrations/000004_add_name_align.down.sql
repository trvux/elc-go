ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_name_align_check;
ALTER TABLE branches DROP COLUMN IF EXISTS name_align;
