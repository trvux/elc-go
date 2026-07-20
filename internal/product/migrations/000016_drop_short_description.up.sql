-- short_description (added in 000005_add_variants) sat unused/empty across
-- the live dataset and duplicated meta_description/description on every
-- page that rendered it — no replacement, just drop.
ALTER TABLE products DROP COLUMN short_description;
