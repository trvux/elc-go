# Module: category

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: Product category records (e.g. "Máy lạnh treo tường"), optionally grouped under a `group` (`group_categories`), shown on the public site and managed in the admin dashboard.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/category/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/category/domain/` (TS types + zod validators, still used by `useCategoryForm.ts`'s client-side form validation) and `modules/category/presentation/` (Server Actions + admin components) are all that remain.

## Data model

Table `categories` (already existed in Supabase before migration — the Go migration in `internal/category/migrations/` is an `IF NOT EXISTS` baseline confirmed column-for-column against the live DB via `\d categories`, not a fresh creation):

| Column | Type | Notes |
|---------------------|-------------|-------|
| `id` | uuid | PK, `gen_random_uuid()` default |
| `name` | text | not null |
| `group_id` | uuid | nullable, FK → `group_categories(id)` **ON DELETE SET NULL** |
| `slug` | text | not null, unique only among **non-deleted** rows (partial index `category_slug_unique_active`) |
| `image_url` | text | nullable |
| `meta_title` | text | nullable |
| `meta_description` | text | nullable |
| `is_featured` | boolean | default `false`, not null |
| `order_index` | integer | default `0`, not null |
| `content` | jsonb | nullable — Tiptap rich text, passed through opaquely as `json.RawMessage`, Go never parses its structure |
| `faq` | jsonb | default `'[]'::jsonb`, nullable — `[]FAQItem{Question, Answer}` |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

**Referencing FKs** (confirmed live): `news.category_id`/`services.category_id` (`ON DELETE SET NULL`), `products.category_id` (`ON DELETE RESTRICT`), `project_category.category_id`/`project_type_category.category_id` (`ON DELETE CASCADE` at the DB level). The `ON DELETE CASCADE` on the join tables only fires on a hard `DELETE` — this module's `SoftDelete` never issues one, so it replicates the old TS code's manual cleanup instead (see below) rather than relying on that DB cascade.

**Soft Delete**: `SoftDelete(categoryID)` is wrapped in a single database transaction:
1. `categories` row has `deleted_at` set.
2. Referencing rows in the `project_type_category` join table (for this one category) are hard deleted (`DELETE FROM project_type_category WHERE category_id = $1`).

This transactional implementation fixes a bug in the old Next.js codebase where these two queries ran separately, without a transaction. `category` itself has no child rows to cascade into (unlike `group`, whose `SoftDelete` cascades into `categories` — see `docs/group.md`), so this is a single-ID version of that same pattern.

## Business rules that are easy to miss

1. **No unique constraint on `name`**: duplicate category names are allowed, same as `group`.
2. **`updated_at` auto-updated by DB trigger**: `update_category_updated_at` (`BEFORE UPDATE`) sets `updated_at = now()` on every row update. The Go domain still updates `updatedAt` in-memory, but the DB trigger wins; repository queries use `RETURNING` to fetch the actual value, keeping Go and DB synchronized.
3. **Resurrect-on-create**: creating a category with a `slug` matching an existing soft-deleted category resurrects (restores) that row instead of inserting a new one — retains the same UUID and history. Kept exactly as implemented in the original TS code.
4. **`group_id` is optional**: unlike `group`/`brand`, a category can exist with no parent group (`group_id IS NULL`) — the joined `group` field in the API response is `null` in that case.

## pgx simple-protocol gotcha for `jsonb` struct slices

The shared pool runs with `pgx.QueryExecModeSimpleProtocol` (see `docs/contact.md`). `internal/category/infrastructure/postgres_repository.go` hand-marshals the `faq` column via `marshalFAQ`/`unmarshalFAQ`, returning `json.RawMessage` (not `[]byte`), same as `group`/`brand`.

## HTTP API

Mounted at `/categories`.

| Method | Path | Body | Response |
|--------|------------------------|--------------------------------------------------------------------------|----------|
| GET | `/categories` | query: `search`, `group_id`, `limit`, `offset`, `include_deleted` | `[]categoryResponse` |
| GET | `/categories/count` | query: `search`, `group_id`, `include_deleted` | `{"count": number}` |
| GET | `/categories/{id}` | — | `categoryResponse` or `404` |
| GET | `/categories/slug/{slug}` | — | `categoryResponse` or `404` |
| POST | `/categories` | `{name, slug, group_id, image_url, meta_title, meta_description, is_featured, order_index, content, faq}` | `201 categoryResponse` |
| PUT | `/categories/{id}` | any subset of the above fields (partial update) | `200 categoryResponse` |
| DELETE | `/categories/{id}` | — (transactional soft delete + join-table cleanup) | `204` |
| POST | `/categories/{id}/restore` | — | `204` |

`categoryResponse` (JSON, snake_case) — `group` is `null` when `group_id` is null:
```json
{
  "id": "uuid",
  "name": "Máy lạnh treo tường",
  "slug": "may-lanh-treo-tuong",
  "group_id": "uuid",
  "group": {
    "id": "uuid",
    "name": "Máy lạnh",
    "slug": "may-lanh",
    "image_url": null,
    "meta_title": null,
    "meta_description": null,
    "is_featured": true,
    "order_index": 1
  },
  "image_url": "https://.../image.webp",
  "meta_title": null,
  "meta_description": null,
  "is_featured": true,
  "order_index": 1,
  "content": { "type": "doc", "content": [] },
  "faq": [{ "question": "...", "answer": "..." }],
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "deleted_at": null
}
```

## How `elc-tem` calls this

`modules/category/presentation/actions.ts` calls the Go API over HTTP using `fetch` with `process.env.GO_API_URL`. Unlike `group`, `getCategoriesAction` is also called from inside `"use cache"` render functions (`app/(public)/san-pham/page.tsx`, `modules/catalog/presentation/components/public/ProductListModule.tsx`), so it follows `catalog`'s `isPrerenderError`/rethrow convention in addition to the usual `try/catch` guard — a real fetch failure must propagate so Next.js keeps serving the stale cache instead of silently caching an empty category list.

Consumers fixed during this cutover:
- `modules/dashboard/application/getDashboardStats.ts` / `modules/dashboard/presentation/actions.ts` — dropped the DIP-injected `CategoryRepository`; now takes a pre-fetched `categoriesCount: number` (via `GET /categories/count`, same precedent as `services`/`branches`/`pages` counts) and `allCategories` (via `getCategoriesAction()`).
- `app/(public)/san-pham/page.tsx` — switched from a direct `getCategories(categoryRepo)` call to `getCategoriesAction()`.
- `modules/catalog/presentation/resolveProductPath.ts` — the `"category"` slug-registry branch switched from a raw `supabase.from("categories")` read to `getCategoryByIdAction(id)`. This also fixes a latent bug where `entity.data.groupId` was read off a snake_case raw row cast to the camelCase `Category` type and was always `undefined` — `getCategoryByIdAction` does proper snake→camel mapping including the nested `group`.
- `modules/catalog/presentation/components/public/ProductListModule.tsx` — dropped two raw Supabase reads (`group_categories` for the breadcrumb parent, `categories` for group→category-id expansion) in favor of the already-fetched, Go-backed `allCategories` list and the joined `group` field on `CategoryWithGroup`.

## Dead code found during audit

None — every method exposed on the old `CategoryRepository` interface (`getAll`, `getById`, `create`, `update`, `delete`) had at least one real caller once bypass call sites were accounted for. `count()` had callers only in the dashboard module, now served by the dedicated `/categories/count` endpoint instead of a TS-side wrapper.

## Not migrated (out of scope for this module)

- `catalog`'s own hard-coded popularity-sort category-slug tiebreaker list (`internal/catalog/infrastructure/postgres_repository.go`) is untouched — it's `catalog`'s own logic, not `category`'s.
- `news`/`services`/`project`/`project-type` modules' own FKs/joins into `categories` are untouched beyond the consumer fixes listed above; those modules' own migration to Go (if/when it happens) is separate work.

## Testing

- `go test ./internal/category/...` — domain + application unit tests (using fake repository).
- `go test -tags=integration ./internal/category/infrastructure/...` — integration test verifying CRUD, the joined `group` read (and that it's `nil` for a `group_id`-less category), soft delete + restore, and resurrect-on-create. Run and passing against the live Supabase Postgres as of this migration.
