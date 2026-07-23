-- One-time sync: dev's category thumbnail images (image_url column) were
-- uploaded to R2 (bucket elc-images, shared between dev and production)
-- but never pushed to production's categories table. Written 2026-07-21.
-- Matches by slug, not id -- same reasoning as
-- sync-category-brand-content.sql next to this file.
--
-- Usage: run once against production.
--   PGPASSWORD=... psql -h <host> -U elc -d elc -f scripts/sync-category-images.sql
--
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720150040-3efeea865507cc11.webp' WHERE slug = 'bang-dieu-khien';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720150139-51adc23ce8de85fd.webp' WHERE slug = 'cam-bien-thong-minh';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720151213-65985e709fc9301c.webp' WHERE slug = 'cong-tac-thong-minh';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145909-3f8425b399f9b3bf.webp' WHERE slug = 'may-cap-khi-tuoi-loc-khong-khi';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145449-6c818bdbb902ac03.webp' WHERE slug = 'may-lanh-am-tran-da-huong-thoi';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145825-77a0f24fade8e21f.webp' WHERE slug = 'may-lanh-ap-tran';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145543-a7cb74e7c189f17d.webp' WHERE slug = 'may-lanh-giau-tran-noi-ong-gio';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145337-de4f7e54d09d14b6.webp' WHERE slug = 'may-lanh-treo-tuong';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145748-8da2636fc7cdf097.webp' WHERE slug = 'may-lanh-tu-dung';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720150007-3540fa99c4fe5892.webp' WHERE slug = 'may-loc-nuoc-ro-3-in-1';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720145936-cd9bea50bd6076d9.webp' WHERE slug = 'phu-kien-dong-bo-cua-he-thong-cap-gio-tuoi';
UPDATE categories SET image_url = 'https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/categories/20260720150215-39d73b74a2155e1b.webp' WHERE slug = 'remote-cam-tay';
