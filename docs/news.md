# Module: news

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: Public "tin tức" (news article) records shown on the public site (`/tin-tuc`, `/tin-tuc/{slug}`), surfaced as "related articles" on category/group/product-list pages via name-matching, counted/listed on the admin dashboard, and managed in the admin dashboard. The simplest module migrated so far in this shape — a single flat table, no join tables, one optional FK to `categories` that's never actually populated in real data (0/103 rows have a `category_id` as of this migration).

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/news/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/news/domain/` (TS types + zod validators, still used by `useNewsForm.ts`'s client-side form validation) and `modules/news/presentation/` (Server Actions + admin/public components) are all that remain.

## Data model

Table `news` (already existed in Supabase before migration — the Go migration in `internal/news/migrations/` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column | Type | Notes |
|---------------------|-------------|-------|
| `id` | uuid | PK, `gen_random_uuid()` default |
| `title` | text | not null |
| `slug` | text | not null, **globally UNIQUE** (named `services_slug_key` — a leftover from a copy-pasted migration, unrelated to the `services` table) — unlike `brand`/`group`/`category`'s partial "unique among non-deleted rows" index, this applies even to soft-deleted rows |
| `content` | jsonb | not null, default `'{}'` — Tiptap rich text |
| `image` | text | not null, default `''` |
| `is_published` | boolean | not null, default `false` |
| `order_index` | integer | not null, default `0` |
| `meta_title`/`meta_description` | text | nullable |
| `category_id` | uuid | nullable, FK → `categories(id)` ON DELETE SET NULL — never populated by any real row as of this migration (confirmed via `psql`); the admin UI has a field for it but nobody has used it |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

103 rows total, 3 soft-deleted, as of this migration (confirmed via `psql`, not `database.types.ts`).

## Business rules that are easy to miss

1. **Resurrect-on-create is mandatory, not a business choice** — same reasoning as `project` (see `docs/project.md`): `news.slug`'s plain `UNIQUE` constraint means a plain `INSERT` on a reused slug would fail outright while the old soft-deleted row still exists. `Create()` always checks for a soft-deleted row with the same slug first and resurrects it (same id) instead of inserting.
2. **Two redundant triggers** (`update_news_modtime` and `update_news_updated_at`) both call the same `update_updated_at_column()` function on every `UPDATE` — harmless (idempotent), left as-is, not something the Go migration should try to clean up.
3. **The old TS repository's `create()` had a dead `?? true` fallback** for `is_published` (`input.isPublished ?? true`) that could never fire — `createNewsSchema.parse()` already applies Zod's `.default(false)` before the repository ever sees the input, so `input.isPublished` is always a real boolean by the time it reaches the repository. Not ported; Go's `CreateNewsInput.IsPublished` is a plain `bool`.
4. **`categoryId` has no format validation** in the old TS Zod schema (`z.string().nullable().optional().or(z.literal(""))` — no UUID check) — Go mirrors this by not validating `CategoryID` at all, unlike `project`'s required `CategoryID` (which does validate non-empty).
5. **`updated_at` is set by both DB triggers**, not something Go needs to reproduce — Go still sets it in-memory for consistency, but queries use `RETURNING` so the client always gets the real post-trigger value.

## HTTP API

Mounted at `/news`.

| Method | Path | Body/Query | Response |
|--------|------|------|----------|
| GET | `/news` | query: `is_published`, `category_id`, `exclude_id`, `search`, `limit`, `offset`, `include_deleted`, `sort_by` (`created_at`\|`order_index`), `sort_order` (`asc`\|`desc`) | `[]newsResponse` |
| GET | `/news/count` | same filter query params | `{"count": n}` |
| GET | `/news/{id}` | — | `newsResponse` or `404` |
| GET | `/news/slug/{slug}` | — | `newsResponse` or `404` |
| POST | `/news` | full create body | `201 newsResponse` |
| PUT | `/news/{id}` | partial update body | `200 newsResponse` |
| DELETE | `/news/{id}` | — (soft delete) | `204` |
| POST | `/news/{id}/restore` | — | `204` |

`newsResponse` (JSON, snake_case): `id, title, slug, image, content, category_id, is_published, meta_title, meta_description, order_index, created_at, updated_at, deleted_at`.

## How `elc-tem` calls this

`modules/news/presentation/actions.ts` calls the Go API over HTTP using `fetch` with `process.env.GO_API_URL`. Every Server Action wraps its fetch call in `try/catch` (build-time safety when `GO_API_URL` is undefined — see PR #588 lesson referenced by every prior migration doc).

Consumers fixed during this cutover (all previously imported `newsRepo`/application functions directly, bypassing Server Actions):
- `app/(public)/tin-tuc/page.tsx` — news hub listing, now calls `getNewsAction(...).then(unwrapActionResult)`.
- `app/(public)/tin-tuc/[slug]/page.tsx` — detail page, now calls `getNewsAction`/`getNewsBySlugAction`.
- `modules/catalog/presentation/components/public/ProductListModule.tsx` — "related articles" cross-link, now calls `getNewsAction`.
- `modules/dashboard/application/getDashboardStats.ts` — dropped `NewsRepository`/`newsRepo` from `DashboardRepositories` entirely (same precedent as `products`/`project`, see the function's own doc comment); `DashboardRepositories` is now gone as a parameter since it had no remaining members. Calls `getNewsAction`/`countNewsAction` directly.
- `modules/dashboard/presentation/actions.ts` — no longer imports `newsRepo`, no longer passes a `repos` argument to `getDashboardStats`.

## Testing

- `go test ./internal/news/...` — domain + application unit tests (fake repository).
- `go test -tags=integration ./internal/news/infrastructure/...` — CRUD round-trip and mandatory resurrect-on-slug-reuse, run against the real DB.
- Verified with `npx tsc --noEmit` and `npx next build` on the `elc-tem` side after the cutover — both clean, `/tin-tuc` and `/tin-tuc/[slug]` build successfully.
