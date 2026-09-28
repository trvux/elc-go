-- Adds "group" and "category" to the owner_type set — /san-pham/may-lanh
-- (group) is getting the same content+FAQ depth work brand pages already
-- got (2026-09-28), and category pages (treo tường, âm trần...) are next
-- on the same roadmap, so adding both now avoids a second migration cycle
-- for a near-term, already-agreed need.
ALTER TABLE faqs DROP CONSTRAINT chk_faq_owner_type;
ALTER TABLE faqs ADD CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
    'system_page', 'product', 'project', 'service', 'news', 'branch', 'page', 'brand', 'group', 'category'
]));
