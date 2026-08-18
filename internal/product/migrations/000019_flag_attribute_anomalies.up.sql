-- cmd/detect-attribute-anomalies computes, per number-type attribute, an
-- IQR-based outlier bound across the whole catalog's existing values (no
-- admin-defined range) and flags rows outside it, so AI grounding
-- (internal/ai's search_products tool) can skip a value that's a strong
-- statistical outlier instead of asserting it as fact. See
-- docs/rfc/2026-08-18-product-data-anomaly-detection.md.
ALTER TABLE product_attribute_values ADD COLUMN IF NOT EXISTS flagged_anomaly BOOLEAN NOT NULL DEFAULT false;
