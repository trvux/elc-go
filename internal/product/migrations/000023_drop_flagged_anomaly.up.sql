-- flagged_anomaly was written by cmd/detect-attribute-anomalies and read only
-- by the (since removed) AI chat's search_products tool. Both are gone; the
-- column is dead weight. Applied in a deploy AFTER the code stopped selecting
-- it, so the previous binary never queried a missing column.
ALTER TABLE product_attribute_values DROP COLUMN IF EXISTS flagged_anomaly;
