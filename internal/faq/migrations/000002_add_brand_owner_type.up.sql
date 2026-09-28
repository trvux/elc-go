-- Adds "brand" to the closed owner_type set — brand pages (e.g. LG, Daikin)
-- get real content depth work starting 2026-09-28 (dienmayelc.com.vn), and
-- need FAQPage schema eligibility the same way product/service already have
-- it, not just visible FAQ text.
ALTER TABLE faqs DROP CONSTRAINT chk_faq_owner_type;
ALTER TABLE faqs ADD CONSTRAINT chk_faq_owner_type CHECK (owner_type = ANY(ARRAY[
    'system_page', 'product', 'project', 'service', 'news', 'branch', 'page', 'brand'
]));
