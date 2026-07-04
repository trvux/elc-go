# Module: page

**Status**: Migrated to Go. Custom dynamic static pages are handled via the Go API now.
**Purpose**: Dynamic content page management (e.g. Terms of Service, Privacy Policy, Custom About pages).

## What it replaced

Before migration, this lived in `elc-tem` (Next.js) as `modules/page/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. The frontend `application/` and `infrastructure/` folders have been deleted. `modules/page/domain/` and `modules/page/presentation/actions.ts` are all that remain.

> [!NOTE]
> `AboutBlock` and the `about_blocks` table were found to be dead code (no references outside the unused local repository). Therefore, they were excluded from the Go migration.

## Data model

Table `pages`:

| Column             | Type        | Notes |
|--------------------|-------------|-------|
| `id`               | uuid        | PK |
| `title`            | text        | |
| `slug`             | text        | |
| `content`          | jsonb       | Tiptap rich-text structure |
| `is_published`     | boolean     | |
| `meta_title`       | text        | nullable |
| `meta_description` | text        | nullable |
| `order_index`      | integer     | |
| `created_at`       | timestamptz | |
| `updated_at`       | timestamptz | |
| `deleted_at`       | timestamptz | nullable (Soft delete) |

A partial unique index is defined on `slug` where `deleted_at IS NULL` to prevent active slug conflicts.

## HTTP API

Mounted at `/pages`.

| Method | Path               | Body                     | Response |
|--------|--------------------|--------------------------|----------|
| GET    | `/pages`           | —                        | `[]PageDTO` |
| GET    | `/pages/count`     | —                        | `{"count": int}` |
| POST   | `/pages`           | `CreatePageInput`        | `PageDTO` |
| GET    | `/pages/{id}`      | —                        | `PageDTO` |
| GET    | `/pages/slug/{s}`  | —                        | `PageDTO` |
| PUT    | `/pages/{id}`      | `UpdatePageInput`        | `PageDTO` |
| DELETE | `/pages/{id}`      | —                        | `204` (soft delete) |
| POST   | `/pages/{id}/restore` | —                     | `204` |

## How `elc-tem` calls this

`modules/page/presentation/actions.ts` has been rewritten:
- `getPagesAction` calls `GET /pages`
- `getPageBySlugAction` calls `GET /pages/slug/{slug}`
- `createPageAction` calls `POST /pages`
- `updatePageAction` calls `PUT /pages/{id}`
- `deletePageAction` calls `DELETE /pages/{id}`

The following consumers were refactored:
- **Public Page View** (`app/(public)/[slug]/page.tsx`): Updated to fetch from `getPageBySlugAction(slug)`.
- **Public Layout Cache** (`modules/settings/application/getPublicLayoutData.ts`): Updated to resolve active pages from `getPagesAction()` rather than issuing raw Supabase client queries.
- **Dashboard Stats** (`modules/dashboard/presentation/actions.ts` & `modules/dashboard/application/getDashboardStats.ts`): Updated to retrieve page counts via the `/pages/count` HTTP request.

## Testing

- `go test ./internal/page/...` — domain + application use case unit tests.
- `go test -tags=integration ./internal/page/infrastructure/...` — integration tests verifying CRUD, search filtering, soft delete, count, and restoration.
