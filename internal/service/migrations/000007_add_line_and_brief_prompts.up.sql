-- Expand step of the service restructure (docs/rfc-service-restructure.md in elc-temp).
--   line          which of the two offerings a service belongs to; drives the hub section,
--                 the call-to-action wording and a tracking dimension. Replaces service_groups.
--   brief_prompts what the customer should send in the Zalo conversation so staff can quote
--                 right away; shown on the page and copied into the Zalo message.
-- The old group/price columns stay until the contract step.
ALTER TABLE services
    ADD COLUMN IF NOT EXISTS line text NOT NULL DEFAULT 'dich-vu',
    ADD COLUMN IF NOT EXISTS brief_prompts text[] NOT NULL DEFAULT '{}';

ALTER TABLE services DROP CONSTRAINT IF EXISTS services_line_check;
ALTER TABLE services
    ADD CONSTRAINT services_line_check CHECK (line IN ('thi-cong', 'dich-vu'));

-- Everything that sat in the old "Cung cấp & lắp đặt" group is system installation work.
-- Guarded: service_groups belongs to another module and is dropped in the contract step,
-- so a database built from scratch later must not fail here.
DO $do$
BEGIN
    IF to_regclass('public.service_groups') IS NOT NULL THEN
        UPDATE services s
           SET line = 'thi-cong'
          FROM service_groups g
         WHERE g.id = s.group_id AND g.slug = 'cung-cap-lap-dat';
    END IF;
END
$do$;
