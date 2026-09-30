ALTER TABLE faqs DROP CONSTRAINT chk_faq_owner_type;
ALTER TABLE faqs ADD CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
    'system_page', 'product', 'project', 'service', 'news', 'branch', 'page', 'brand', 'group', 'category'
]));
