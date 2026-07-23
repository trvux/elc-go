-- Generalizes hp_pages beyond attribute-only filtering: a landing page can
-- now ALSO (or instead) scope to specific categories and/or brands — e.g.
-- "Máy lạnh Daikin" (category=máy lạnh's 5 sub-categories, brand=Daikin),
-- distinct from the plain brand page (all of Daikin's products, whatever
-- categories that spans). Combined with attribute_code/attribute_values,
-- all three filters AND together — same as ProductFilter already does for
-- categoryIds/brandIds/attributeTokens (internal/product's own filter).
--
-- attribute_code/attribute_values are relaxed from NOT NULL/required (see
-- 000001) to optional — a page can now be pure category+brand with no
-- attribute filter at all. Domain layer enforces "at least one of the
-- three filters must be set" instead of the DB.
ALTER TABLE hp_pages ALTER COLUMN attribute_code DROP NOT NULL;
ALTER TABLE hp_pages ALTER COLUMN attribute_code DROP DEFAULT;
ALTER TABLE hp_pages ALTER COLUMN attribute_values DROP NOT NULL;

ALTER TABLE hp_pages ADD COLUMN IF NOT EXISTS category_ids UUID[] NOT NULL DEFAULT '{}';
ALTER TABLE hp_pages ADD COLUMN IF NOT EXISTS brand_ids UUID[] NOT NULL DEFAULT '{}';
