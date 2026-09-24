ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_align_check;
ALTER TABLE products DROP COLUMN IF EXISTS name_align;
