# Module: group

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: Group category records used to group product categories shown on the public site and managed in the admin dashboard.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/group/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/group/domain/` (TS types + zod validators, still used by `useGroupForm.ts`'s client-side form validation) and `modules/group/presentation/` (Server Actions + admin components) are all that remain.

## Data model

Table `group_categories` (already existed in Supabase before migration — the Go migration in `internal/group/migrations/` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column | Type | Notes |
|---------------------|-------------|-------|
| `id` | uuid | PK, `gen_random_uuid()` default |
| `name` | text | not null |
| `slug` | text | not null, unique only among **non-deleted** rows (partial index `group_categories_slug_unique_active`) |
| `image_url` | text | nullable |
| `meta_title` | text | nullable |
| `meta_description` | text | nullable |
| `is_featured` | boolean | default `false`, not null |
| `order_index` | integer | default `0`, not null |
| `content` | jsonb | nullable — Tiptap rich text, passed through opaquely as `json.RawMessage`, Go never parses its structure |
| `faq` | jsonb | default `'[]'::jsonb`, nullable — `[]FAQItem{Question, Answer}` |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

**Soft Delete Cascade**: `SoftDelete(groupID)` implements a 2-tier cascading soft-delete wrapped in a single database transaction. This ensures consistency across three tables:
1. `group_categories` row has `deleted_at` set.
2. Active `categories` rows (where `deleted_at IS NULL` and `group_id = groupID`) have `deleted_at` set to the same timestamp.
3. Referencing rows for those categories in the `project_type_category` join table are hard deleted (using `DELETE FROM project_type_category WHERE category_id = ANY(...)`).

This transactional implementation fixes a bug in the old Next.js codebase where these four queries were executed separately without a transaction, risking inconsistent data states if any query failed mid-execution.

## Business rules that are easy to miss

1. **No unique constraint on `name` — unlike `brand`**: `brand` has a unique constraint on `name` which applies globally. `group_categories` does not have any unique constraint on the `name` column, meaning duplicate names are allowed.
2. **`updated_at` auto-updated by DB trigger**: The table has a `BEFORE UPDATE` trigger `update_groups_updated_at` which sets `updated_at = now()` on every row update. Although the Go domain still updates `updatedAt` in-memory, the DB trigger will overwrite it. The repository queries use `RETURNING` to fetch the actual value set by the DB, keeping Go and DB synchronized.
3. **Resurrect-on-create**: If a group category is created with a `slug` that matches an existing, soft-deleted group category, the repository will resurrect (restore) that soft-deleted row by resetting its `deleted_at` to `NULL` and updating its columns (retaining the same UUID and history). This business requirement was kept as it was actively implemented in the original TS code.

## pgx simple-protocol gotcha for `jsonb` struct slices

The shared pool runs with `pgx.QueryExecModeSimpleProtocol` (required to avoid PgBouncer prepared-statement gotcha, see `docs/contact.md`). That mode can infer how to encode/decode a `jsonb` column into `json.RawMessage` but cannot infer OIDs for Go struct slices like `[]domain.FAQItem`.
To work around this, `internal/group/infrastructure/postgres_repository.go` handles JSON marshalling/unmarshalling of the `faq` column explicitly using `marshalFAQ`/`unmarshalFAQ`, returning `json.RawMessage` specifically (not `[]byte`), since pgx handles `json.RawMessage` natively.

## HTTP API

Mounted at `/groups`.

| Method | Path | Body | Response |
|--------|-----------------------|--------------------------------------------------------------------------|----------|
| GET | `/groups` | query: `search`, `limit`, `offset`, `include_deleted` | `[]groupResponse` |
| GET | `/groups/{id}` | — | `groupResponse` or `404` |
| GET | `/groups/slug/{slug}` | — | `groupResponse` or `404` |
| POST | `/groups` | `{name, slug, image_url, meta_title, meta_description, is_featured, order_index, content, faq}` | `201 groupResponse` |
| PUT | `/groups/{id}` | any subset of the above fields (partial update) | `200 groupResponse` |
| DELETE | `/groups/{id}` | — (cascade soft delete) | `204` |
| POST | `/groups/{id}/restore`| — | `204` |

`groupResponse` (JSON, snake_case):
```json
{
  "id": "uuid",
  "name": "Air Conditioners",
  "slug": "air-conditioners",
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

`modules/group/presentation/actions.ts` is updated to call the Go API endpoints over HTTP using `fetch` with `process.env.GO_API_URL`. Every Server Action wraps its fetch call in a `try/catch` block to prevent build crashes in CI when `GO_API_URL` is undefined.

Consumers fixed during this cutover:
- `app/(public)/tin-tuc/[slug]/page.tsx` — Switched from `getGroups(groupRepo)` call to `getGroupsAction()` Server Action.

## Dead code found during audit

- `getGroupById` (old `modules/group/application/index.ts`) had zero callers in `elc-tem` and was never exposed as a Server Action. We still implemented the corresponding handler and endpoint in the Go backend for REST completeness, but it is not called or exposed as a Server Action.

## Not migrated (out of scope for this module)

`catalog` itself is not migrated yet. Its own infrastructure queries the `group_categories` table directly via `supabase-js` as part of building category-group relationships:
- `modules/catalog/infrastructure/resolveProductPath.ts` — reads `group_categories` directly to resolve canonical URLs.
This will move to Go when `catalog` itself migrates.

## Testing

- `go test ./internal/group/...` — domain + application unit tests (using fake repository).
- `go test -tags=integration ./internal/group/infrastructure/...` — integration tests verifying CRUD, resurrect-on-create, and the transactional cascade soft-delete effects (soft-deleting group category, soft-deleting sub-categories, and hard-deleting project type category associations).
