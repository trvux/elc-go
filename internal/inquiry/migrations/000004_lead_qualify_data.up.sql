-- Supports the redesigned branch-first lead-capture form (see elc-temp
-- modules/inquiry/presentation/components/LeadFormScreen.tsx): lead_type is
-- recorded even when the visitor skipped every catalog picker, so admin can
-- always filter by branch regardless of whether a concrete product/project/
-- service got linked. sub_type is the slug of whatever was picked in the
-- branch's first picker step. qualify_data holds every choice-step answer
-- as a flat {stepId: value} map. attachments holds uploaded photo URLs from
-- the new public POST /inquiries/uploads endpoint.
ALTER TABLE inquiries
    ADD COLUMN lead_type VARCHAR(20) NOT NULL DEFAULT 'general',
    ADD COLUMN sub_type VARCHAR(60),
    ADD COLUMN qualify_data JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN attachments JSONB NOT NULL DEFAULT '[]';

ALTER TABLE inquiries
    ADD CONSTRAINT chk_inquiry_lead_type CHECK (lead_type IN ('product', 'service', 'project', 'general'));

CREATE INDEX IF NOT EXISTS idx_inquiries_lead_type ON inquiries (lead_type);
