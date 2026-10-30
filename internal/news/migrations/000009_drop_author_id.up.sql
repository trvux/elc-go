-- The author feature was removed (no authors in prod, no article ever had a
-- byline) and no Go code reads or writes news.author_id any more. Dropping
-- the column also drops its foreign key to authors, which lets the authors
-- table itself be dropped afterwards.
ALTER TABLE news DROP COLUMN IF EXISTS author_id;
