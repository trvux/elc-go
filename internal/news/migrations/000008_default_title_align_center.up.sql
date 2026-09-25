-- Center is now the fallback for titleAlign everywhere it isn't explicitly
-- set — see internal/platform/titlealign.OrDefault's doc comment
-- (2026-09-25): many already-published articles predate this feature
-- (created back when every title was hardcoded left-aligned), and read as
-- visually "off" once other pages in the same section are deliberately
-- centered — readers assumed it was a layout bug. Backfills every existing
-- row still at the old fixed-behavior default; a row already explicitly
-- set to 'left'/'right' by an admin is untouched by definition (this only
-- ever matches rows nobody has customized yet).
ALTER TABLE news ALTER COLUMN title_align SET DEFAULT 'center';
UPDATE news SET title_align = 'center' WHERE title_align = 'left';
