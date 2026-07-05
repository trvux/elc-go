# Module: system-page

**Status**: Migrated to Go. Admin SEO management screen and public page metadata generation in `elc-tem` go through Go now.
**Purpose**: SEO metadata (title/description) for a fixed, admin-seeded set of hub pages (home, tin-tuc, du-an, dich-vu, san-pham, thong-tin, co-so-ha-tang). Rows are never created or deleted through this module — only `meta_title`/`meta_description` are editable.

## What it replaced

Before migration, this lived in `elc-tem` (Next.js) as `modules/system-page/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. The frontend `application/` and `infrastructure/SupabaseSystemPageRepository.ts` have been deleted. `modules/system-page/domain` and `modules/system-page/presentation/{actions.ts,components/}` are all that remain.

## Data model

Table `system_pages` (already existed in Supabase before migration — the Go migration in `internal/system-page/migrations/000001_baseline_system_pages.up.sql` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column             | Type        | Notes |
|--------------------|-------------|-------|
| `id`               | uuid        | PK, `gen_random_uuid()` |
| `name`             | text        | not null |
| `slug`             | text        | not null, **plain UNIQUE** (`system_pages_slug_key`) |
| `meta_title`       | text        | nullable |
| `meta_description` | text        | nullable |
| `created_at`       | timestamptz | not null |
| `updated_at`       | timestamptz | not null |

No `deleted_at` column and no DB triggers on this table — confirmed via `\d system_pages` on the live DB. Verified the 7 live rows (home, co-so-ha-tang, dich-vu, du-an, thong-tin, san-pham, tin-tuc) are the full, fixed set before writing any Go code.

## HTTP API

Mounted at `/system-pages`.

| Method | Path                       | Body | Response |
|--------|----------------------------|------|----------|
| GET    | `/system-pages`            | —    | `[]SystemPageDTO` |
| GET    | `/system-pages/slug/{slug}`| —    | `SystemPageDTO` (404 if not found) |
| PUT    | `/system-pages/{id}`       | `{meta_title, meta_description}` | `SystemPageDTO` (404 if not found, 400 on validation) |

`SystemPageDTO` (JSON):
```json
{
  "id": "5ffa6a2a-3dbf-4220-a084-51c42811e76d",
  "name": "Trang chủ",
  "slug": "home",
  "meta_title": "Máy lạnh, Hệ thống khí tươi & Dự án trọn gói",
  "meta_description": "Điện máy ELC chuyên cung cấp...",
  "created_at": "2026-06-22T13:10:34.319342Z",
  "updated_at": "2026-06-22T13:10:34.319342Z"
}
```

Validation (mirrors the old TS zod schema): `meta_title` max 70 chars, `meta_description` max 160 chars.

## How `elc-tem` calls this

`modules/system-page/presentation/actions.ts` has been rewritten to call the Go API via `fetch`, following the same pattern as `settings`/`page`:
- `getSystemPagesAction` — `GET /system-pages`, used by the admin `SystemPageManagement` table.
- `getSystemPageBySlugAction` — `GET /system-pages/slug/{slug}`.
- `updateSystemPageAction` — `PUT /system-pages/{id}`, revalidates `/admin/system-pages` and the public path for the edited slug (`/` for `home`), then purges the Cloudflare cache.

`shared/lib/cached-system-page.ts` (a `"use cache"` helper, not a Server Action itself — used by every public page's `generateMetadata`) previously imported the TS `application`/`infrastructure` layer and the `systemPageRepo` singleton directly, bypassing Server Actions entirely. It's been rewritten to call `getSystemPageBySlugAction` instead.

## Testing

- `go test ./internal/system-page/...` — domain (`UpdateMeta` length validation) + application use case unit tests (fake repo).
- `go test -tags=integration ./internal/system-page/infrastructure/...` — integration test against the live DB. Since this module has no `Create`/`Delete`, the test reads an existing row (`co-so-ha-tang`), updates its meta fields, verifies, then restores the original values in a `defer` (no insert/delete round-trip like other modules).
- Manually verified end-to-end: started `air` (Go, :8090) + `pnpm dev` (Next.js, :3000), confirmed the homepage's `<title>`/`<meta name="description">` match the `home` row via the new Go-backed path, and exercised all three HTTP endpoints (list, get-by-slug incl. 404, update incl. validation error and not-found) with `curl`.
