-- The system-page module is gone and its FAQs were deleted, so "system_page"
-- is no longer a valid owner kind. Fails if any row still uses it.
ALTER TABLE faqs DROP CONSTRAINT chk_faq_owner_type;
ALTER TABLE faqs ADD CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
    'product', 'project', 'service', 'news', 'branch', 'page', 'brand', 'group', 'category', 'hp_page'
]));
