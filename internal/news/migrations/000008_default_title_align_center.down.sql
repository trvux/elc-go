-- Data backfill is not reversible here — rows this migration flipped from
-- 'left' to 'center' can't be distinguished from rows an admin explicitly
-- set to 'center' after the fact. Down only reverts the column default.
ALTER TABLE news ALTER COLUMN title_align SET DEFAULT 'left';
