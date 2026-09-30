-- Adds "hp_page" to the owner_type set — HP-tier landing pages (1HP-5.5HP,
-- /san-pham/<hp-slug>) are a real, distinct search-intent cross-cut ("máy
-- lạnh daikin 1hp", "máy lạnh 2hp") that hasOfferCatalog already treats as
-- its own branch alongside brand/group/category (2026-09-30 schema.org
-- audit), and every other owner kind on that same audit already has FAQ
-- content — hp_page was the one gap left at 0/10 pages.
ALTER TABLE faqs DROP CONSTRAINT chk_faq_owner_type;
ALTER TABLE faqs ADD CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
    'system_page', 'product', 'project', 'service', 'news', 'branch', 'page', 'brand', 'group', 'category', 'hp_page'
]));
