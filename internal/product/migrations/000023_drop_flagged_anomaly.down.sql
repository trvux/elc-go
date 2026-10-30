ALTER TABLE product_attribute_values ADD COLUMN IF NOT EXISTS flagged_anomaly BOOLEAN NOT NULL DEFAULT false;
