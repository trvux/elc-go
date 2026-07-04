# Module: branch

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: Branch records representing physical branch locations, contact info, and Google Maps embeddings, shown on the public site and managed in the admin dashboard.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/branch/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/branch/domain/` (TS types + zod validators, still used by client-side form validation) and `modules/branch/presentation/` (Server Actions + admin components) are all that remain.

## Data model

Table `branches` (already existed in Supabase before migration — the Go migration in `internal/branch/migrations/` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column              | Type        | Notes |
|---------------------|-------------|-------|
| `id`                | uuid        | PK, `gen_random_uuid()` default |
| `name`              | text        | not null |
| `slug`              | text        | not null, unique index `branches_slug_key` |
| `address`           | text        | not null |
| `phone`             | text        | not null |
| `email`             | text        | not null |
| `maps_url`          | text        | not null |
| `maps_embed`        | text        | not null |
| `description`       | jsonb       | nullable — rich text, passed through opaquely as `json.RawMessage` |
| `image_url`         | text        | nullable |
| `is_published`      | boolean     | default `false` |
| `order_index`       | integer     | default `0` |
| `meta_title`        | text        | nullable |
| `meta_description`  | text        | nullable |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | columns present in DB schema |

**Hard Delete Constraint**: Although the DB schema contains a `deleted_at` column, the old Next.js codebase deleted rows using a hard `.delete()` query. To preserve this behavior and maintain exact consistency, the Go backend implements a hard `DELETE` via `DELETE FROM branches WHERE id = $1`.

## HTTP API

Mounted at `/branches`.

| Method | Path                  | Body                                                                    | Response |
|--------|-----------------------|--------------------------------------------------------------------------|----------|
| GET    | `/branches`             | query: `search`, `limit`, `offset`, `is_published`                      | `[]branchResponse` |
| GET    | `/branches/count`       | query: `search`, `is_published`                                          | `{"count": count}` |
| POST   | `/branches`             | `{name, slug, address, phone, email, maps_url, maps_embed, description, image_url, is_published, order_index, meta_title, meta_description}` | `201 branchResponse` |
| GET    | `/branches/{id}`        | —                                                                          | `branchResponse` or `404` |
| GET    | `/branches/slug/{slug}` | —                                                                          | `branchResponse` or `404` |
| PUT    | `/branches/{id}`        | any subset of the above fields (partial update)                            | `200 branchResponse` |
| DELETE | `/branches/{id}`        | — (hard delete)                                                            | `204` |
| PUT    | `/branches/{id}/order`  | `{order_index}` (sets explicit sort index for drag-and-drop order updates)  | `204` |

`branchResponse` (JSON, snake_case):
```json
{
  "id": "uuid",
  "name": "ELC Q1",
  "slug": "elc-q1",
  "address": "123 Le Loi, Q1",
  "phone": "0901234567",
  "email": "q1@elc.vn",
  "maps_url": "https://maps.google.com/q1",
  "maps_embed": "<iframe></iframe>",
  "description": { "text": "Mo ta" },
  "image_url": "https://.../branch.webp",
  "is_published": true,
  "order_index": 1,
  "meta_title": null,
  "meta_description": null,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "deleted_at": null
}
```

## How `elc-tem` calls this

`modules/branch/presentation/actions.ts` has been rewritten:
- Serves as the server-side proxy calling the Go API endpoint using `fetch()`.
- Explicitly handles `isPrerenderError(error)` to propagate Next.js static prerender abort signals rather than logging them as errors, preventing build-time console warnings.
- Synchronizes routes and static components via revalidation tags.

## Testing

- `go test ./internal/branch/...` — domain + application unit tests (fake repository, no network).
- `go test -tags=integration ./internal/branch/infrastructure/...` — real DB roundtrip verifying CRUD, count, order index update, slug retrieval, and deletion.
