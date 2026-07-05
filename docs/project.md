# Module: project

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: Public "dự án" (completed project/case study) records shown on the public site (`/du-an`, `/du-an/{slug}`), homepage marquee, search, and dashboard, and managed in the admin dashboard. The richest module migrated so far apart from `catalog` — two join tables (one with a composite, condition-scoped key), a price-aggregation read path, and a cross-module read-only dependency on the not-yet-migrated `project-type` module.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/project/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/project/domain/` (TS types + zod validators, still used by `useProjectForm.ts`'s client-side form validation) and `modules/project/presentation/` (Server Actions + admin/public components) are all that remain, plus one new file: `modules/project/presentation/resolveProjectPath.ts` (see "Not migrated" below).

## Data model

Tables `projects`, `project_category`, `project_service` (already existed in Supabase before migration — the Go migration in `internal/project/migrations/` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column | Type | Notes |
|---------------------|-------------|-------|
| `id` | uuid | PK, `gen_random_uuid()` default |
| `category_id` | uuid | not null — the project's single "primary" category, distinct from the many-to-many `project_category` join below |
| `title` | text | not null |
| `slug` | text | not null, **globally UNIQUE** (`projects_slug_key`) — unlike brand/group/category's partial "unique among non-deleted rows" index, this applies even to soft-deleted rows |
| `description` | jsonb | not null, default `'{}'` — Tiptap rich text |
| `images` | text[] | not null, default `'{}'` |
| `is_published` | boolean | nullable, default `true` |
| `order_index` | integer | nullable, default `0` |
| `is_featured` | boolean | nullable, default `false` |
| `meta_title`/`meta_description` | text | nullable |
| `project_type_id` | uuid | nullable, FK → `project_type(id)` ON DELETE SET NULL (no NOT NULL/SET NULL contradiction here, unlike `docs/brand.md`'s `products.brand_id`) |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

`project_category` (many-to-many, join): `project_id`, `category_id`, `condition` (Postgres enum `product_condition`, `'new'`/`'used'`), `created_at`. **PK is composite `(project_id, category_id, condition)`** — the same project can attach the same category twice, once per condition.

`project_service` (many-to-many, join): `project_id`, `service_id`, `created_at`. Simple PK `(project_id, service_id)`, no `deleted_at` (hard-delete only).

**`is_published`/`order_index`/`is_featured` are nullable columns with only a DEFAULT, not NOT NULL** — the repository's `SELECT` wraps them in `COALESCE(..., false)` / `COALESCE(..., 0)` to reproduce the old TS mapper's `row.is_published || false` / `row.order_index || 0` fallbacks exactly (confirmed via `\d projects`, not assumed from `database.types.ts`).

## Business rules that are easy to miss

1. **Resurrect-on-create is mandatory, not a business choice** — unlike `group` (where resurrect was a deliberate business decision despite the constraint only being partial), `projects.slug`'s plain `UNIQUE` constraint means a plain `INSERT` on a reused slug would fail outright while the old soft-deleted row still exists. `Create()` always checks for a soft-deleted row with the same slug first and resurrects it (same id, relations cleared and rewritten fresh) instead of inserting.
2. **`SoftDelete` only touches the `projects` row** — the old TS `delete()` never cleaned up `project_category`/`project_service` either. Both join tables have `ON DELETE CASCADE` FKs, but soft-delete is an `UPDATE`, so cascade never fires — orphaned join rows pointing at a soft-deleted project are the accepted, unchanged behavior (same "leftover FK reference" precedent as `docs/brand.md`'s `products.brand_id`).
3. **`Create`/`Update` run the row write + full relation replace (delete-all, insert-new) inside one transaction.** The old TS versions ran the equivalent as several sequential, un-transactioned Supabase calls (find-existing → insert/update → delete old relations → insert new relations) — a real partial-failure risk this fixes, same spirit as `service-group`/`group`'s transactional `SoftDelete`. Verified with an integration test that a bad `category_id` in the relations payload rolls back the whole `projects` row insert too.
4. **`UpdateProjectInput.Categories`/`ServiceIDs` are pointer-to-slice, not plain slices** — `nil` means "leave relations untouched" (field omitted from the JSON body), a non-nil pointer (including one to an empty slice) means "replace with this set" (field present, even as `[]`). This mirrors the old TS repository's `input.categories !== undefined` / `input.serviceIds !== undefined` check exactly. On the `elc-tem` side, `updateProjectAction` only includes the `categories`/`service_ids` JSON keys when the caller actually passed them.
5. **Pricing (`low_price`/`high_price`/`offer_count`) is NOT part of the default read shape.** The old TS `projectRepo.ts`'s `getAll`/`getById`/`getBySlug` never computed these — only `infrastructure/resolveProjectPath.ts` (used solely by the public project detail page) did, via a separate join into `products`. `GetBySlug(ctx, slug, withPricing)` reproduces this exact split with a boolean flag rather than always paying for the extra `LATERAL` join; `GetAll`/`GetByID` never compute pricing (nothing in `elc-tem` needs it there).
6. **`updated_at` is set by a DB trigger (`update_projects_modtime`), same as `group`** — Go still sets it in-memory for consistency, but the trigger overwrites it on every `UPDATE`; queries use `RETURNING` so the value returned to the client is always the real post-trigger value.
7. **`UpdateOrder`/`TogglePublish`/`ToggleFeatured` don't set `updated_at` themselves** — matches the old TS single-column updates exactly; the trigger handles it regardless.

## Price aggregation SQL (the `withPricing=true` path)

```sql
SELECT MIN(price) AS low_price, MAX(price) AS high_price, COUNT(*) AS offer_count
FROM (
  SELECT COALESCE(NULLIF(pr.sale_price, 0), NULLIF(pr.original_price, 0), 0) AS price
  FROM products pr
  WHERE pr.category_id = c.id AND pr.is_published = true AND pr.deleted_at IS NULL
) prices
WHERE price > 0
```
Reproduces the old TS `p.sale_price || p.original_price || 0` (JS treats `0` as falsy too) via `COALESCE`+`NULLIF`, then only ranges/counts strictly-positive prices — same as the old `.filter((p) => p > 0)`. This queries `products` directly (same DB, same pattern `brand`/`group` used to read tables owned by not-yet-migrated modules) — it does **not** call `catalog`'s Go module, matching how the old TS never routed this through the `catalog` module either.

## GetAdjacent — pushed to SQL, not ported verbatim

The old `getAdjacentProjects.ts` loaded every published project into memory and sorted in JS. `GetAdjacent` ports `internal/catalog`'s `GetAdjacent` pattern instead: same "same-`project_type_id` siblings first, fall back to the full published set when fewer than 2 siblings" rule, as two indexed SQL queries at most (`ORDER BY is_featured DESC, order_index ASC`), the fallback only running when the first came back too small.

## HTTP API

Mounted at `/projects`.

| Method | Path | Body/Query | Response |
|--------|------|------|----------|
| GET | `/projects` | query: `category_id`, `project_type_id`, `category_slug(s)`, `service_slug(s)`, `exclude_id`, `is_published`, `is_featured`, `search`, `limit`, `offset`, `include_deleted`, `order_by`, `order_direction` | `[]projectResponse` |
| GET | `/projects/count` | same filter query params (subset) | `{"count": n}` |
| GET | `/projects/{id}` | — | `projectResponse` or `404` |
| GET | `/projects/slug/{slug}` | query: `with_pricing` (`"true"` to include category pricing) | `projectResponse` or `404` |
| GET | `/projects/adjacent` | query: `current_id` (required), `project_type_id` | `{"prev": {...}\|null, "next": {...}\|null}` |
| GET | `/projects/categories-by-project-type/{projectTypeId}` | — | `[]{id,name,slug}` |
| POST | `/projects` | full create body | `201 projectResponse` |
| PUT | `/projects/{id}` | partial update body | `200 projectResponse` |
| DELETE | `/projects/{id}` | — (soft delete) | `204` |
| POST | `/projects/{id}/restore` | — | `204` |
| PATCH | `/projects/{id}/order` | `{"order_index": n}` | `204` |
| PATCH | `/projects/{id}/publish` | `{"value": bool}` | `204` |
| PATCH | `/projects/{id}/featured` | `{"value": bool}` | `204` |

`categories`/`services` request payload shape: `categories: [{"category_id": "uuid", "condition": "new"|"used"}]`, `service_ids: ["uuid", ...]`.

`projectResponse` (JSON, snake_case) carries nested `project_type` (`{id,name,slug}` or `null`), `categories` (`{id,name,slug,group_id,condition,group:{id,name}|null,low_price,high_price,offer_count}[]`), `services` (`{id,title,slug,group:{id,name,slug}|null}[]`).

## How `elc-tem` calls this

`modules/project/presentation/actions.ts` calls the Go API over HTTP using `fetch` with `process.env.GO_API_URL`. Every Server Action wraps its fetch call in `try/catch` (build-time safety when `GO_API_URL` is undefined — see PR #588 lesson referenced by every prior migration doc). New exports beyond the original set: `getAdjacentProjectsAction`, `getFeaturedProjectsAction`, `getCategoriesByProjectTypeIdAction`, `resolveProjectDetailAction` (pricing-enriched, replaces the old `resolveProjectPathFromDb`'s "project" branch).

Consumers fixed during this cutover (all previously imported `projectRepo`/application functions directly, bypassing Server Actions):
- `app/(public)/page.tsx` — homepage, now calls `getProjectsAction(...).then(unwrapActionResult)`.
- `app/(public)/du-an/[slug]/page.tsx` — now calls `resolveProjectPathFromDb` from the new `modules/project/presentation/resolveProjectPath.ts`, and `getAdjacentProjectsAction`.
- `shared/components/layout/user/header/search-actions.ts` — `getCachedProjects()` now calls `getProjectsAction`.
- `modules/dashboard/application/getDashboardStats.ts` — dropped `ProjectRepository`/`projectRepo` from `DashboardRepositories` (same precedent as `products`, see the function's own doc comment), calls `getProjectsAction`/`countProjectsAction` directly.
- `modules/project/presentation/components/public/RelatedProjects.tsx` — dropped its `projectRepo` prop entirely, calls `getProjectsAction`.
- `modules/project/presentation/components/public/ProjectListModule.tsx` — calls `getProjectsAction`/`getCategoriesByProjectTypeIdAction` (still separately depends on the un-migrated `project-type` module for project-type listing/filters — untouched).

## Not migrated (out of scope for this module)

`project-type` is not migrated. `resolveProjectPathFromDb` (moved from `modules/project/infrastructure/resolveProjectPath.ts` to `modules/project/presentation/resolveProjectPath.ts` since `infrastructure/` no longer exists) resolves **two** entity types from one `slug_registry` lookup: its `project_type` branch is untouched and still queries Supabase directly (categories/group_categories joins included); only its `project` branch was replaced, now calling `resolveProjectDetailAction` (Go, with pricing). This file is the one remaining place in the `project` module that still talks to Supabase directly, and that's intentional — it will collapse to pure Go once `project-type` migrates.

## Dead code found during audit

- `getProjectsByIds`/`ProjectRepository.getByIds` (old `modules/project/application/getProjectById.ts`) had **zero callers anywhere** in `elc-tem` — not even internally (unlike `getProjectById`, which `deleteProjectAction`/`toggleProjectPublishAction`/`toggleProjectFeaturedAction`/`updateProjectOrderAction` all called internally to fetch the slug for cache revalidation). Not ported to Go — genuinely unused, unlike the `getGroupById`/`getProjectById` precedent (implemented for REST completeness) documented in `docs/group.md`.
- `updateProjectAction` (old `modules/project/presentation/actions.ts`) contained a debug leftover writing `input.description` to `/Users/tranvux/Documents/elc-tem/scratch/received-payload.json` on every update — removed during the rewrite, not part of any real feature.

## Testing

- `go test ./internal/project/...` — domain + application unit tests (fake repository), including the nil-vs-empty relations distinction (rule 4 above).
- `go test -tags=integration ./internal/project/infrastructure/...` — CRUD round-trip, mandatory resurrect-on-slug-reuse, transaction rollback on a bad relation, `GetAdjacent` fallback, `GetBySlug` with/without pricing — all run against the real DB.
- Manually verified via `curl` against a running `air`-reloaded server, and via the real Next.js dev server (homepage, `/du-an` listing, `/du-an/{slug}` detail — pager + related projects + `Product` schema.org pricing all confirmed rendering with real data).
