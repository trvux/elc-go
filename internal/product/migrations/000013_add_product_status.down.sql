ALTER TABLE products ADD COLUMN is_published boolean NOT NULL DEFAULT true;
ALTER TABLE products ADD COLUMN labels text[] DEFAULT '{}';

UPDATE products SET is_published = (status = 'published');

ALTER TABLE products DROP COLUMN status;
ALTER TABLE products DROP COLUMN rejection_reason;

DROP TYPE product_status;
