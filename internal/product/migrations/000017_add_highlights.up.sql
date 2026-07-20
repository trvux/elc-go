-- Đặc điểm nổi bật — a short bullet list of key selling points, shown above
-- the description article on the product detail page. Plain text[] rather
-- than the description column's rich-text jsonb: each row is a standalone
-- one-line claim (e.g. "Công suất làm lạnh 1 HP phù hợp phòng dưới 15m²"),
-- not prose that needs formatting — an admin editing "the 3rd bullet" maps
-- directly onto array index, no rich-text traversal needed.
ALTER TABLE products ADD COLUMN highlights TEXT[];
