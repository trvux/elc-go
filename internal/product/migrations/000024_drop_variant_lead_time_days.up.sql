-- lead_time_days only fed the "Giao trong N ngày" card label, which was
-- removed; 0 of 213 prod variants ever had a value. Applied in a deploy AFTER
-- the code stopped selecting it.
ALTER TABLE product_variants DROP COLUMN IF EXISTS lead_time_days;
