# Module: project-type

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem` go through Go now.
**Purpose**: The "loại hình công trình" (project type — Villa, Nhà xưởng, Cơ sở giáo dục, ...) taxonomy used to classify `projects` and to drive the public `/du-an` filter UI and `/[slug]` category landing pages. Simple in shape (one table + one join table), but it closes the last cross-module gap `project` had left open — see "How `elc-tem` calls this" below.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as `modules/project-type/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. `application/` and `infrastructure/` have been deleted from `elc-tem` — `modules/project-type/domain/` (TS types + zod validators, still used by `useProjectTypeForm.ts`'s client-side form validation) and `modules/project-type/presentation/` (Server Actions + admin components) are all that remain.

## Data model

Tables `project_type`, `project_type_category` (already existed in Supabase before migration — the Go migration in `internal/project-type/migrations/` is an `IF NOT EXISTS` baseline, not a fresh creation). Column/constraint/trigger names still carry the table's pre-rename identity (`service_type`) — it was renamed to `project_type` at some point but `service_type_pkey`, `service_type_slug_unique`, etc. were never renamed:

| Column | Type | Notes |
|---------------------|-------------|-------|
| `id` | uuid | PK, `gen_random_uuid()` default |
| `name` | varchar(255) | not null |
| `slug` | text | not null, **globally UNIQUE** (`service_type_slug_unique`) — unlike brand/group/category's partial "unique among non-deleted rows" index, this applies even to soft-deleted rows |
| `image` | text | nullable |
| `meta_title`/`meta_description` | text | nullable |
| `is_featured` | boolean | nullable, default `false` |
| `order_index` | integer | nullable, default `0` |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

`project_type_category` (many-to-many, join): `project_type_id`, `category_id`, `created_at`. Composite PK `(project_type_id, category_id)`.

`projects.project_type_id` is a nullable FK → `project_type(id)` ON DELETE SET NULL (see `docs/project.md`).

## Business rules that are easy to miss

1. **Resurrect-on-create is mandatory, not a business choice** — `project_type.slug`'s plain `UNIQUE` constraint means a plain `INSERT` on a reused slug would fail outright while the old soft-deleted row still exists. `Create()` always checks for a soft-deleted row with the same slug first and resurrects it (same id, `project_type_category` relations cleared and rewritten fresh) instead of inserting — same pattern as `project`/`brand` in the plain-unique case.
2. **`SoftDelete` is a real 3-table transaction, not a single-row soft delete** — the old TS `delete()` ran this as three sequential, un-transactioned Supabase calls: (a) soft-delete the `project_type` row, (b) `UPDATE projects SET project_type_id = NULL WHERE project_type_id = id` (the FK's own `ON DELETE SET NULL` never fires here because a soft delete is an `UPDATE`, not a real `DELETE`), (c) hard-delete this project type's `project_type_category` rows. Go wraps all three in one `pool.Begin`/`tx.Commit` — the same atomicity fix pattern as `project`/`category`'s transactional writes.
3. **No restore action exists** — unlike category/brand/group/project, the old TS `ProjectTypeRepository` interface never had a `restore()` method and no admin UI action called it. Go's `domain.ProjectTypeRepository` deliberately omits `Restore` from the interface (though the domain entity still has a `Restore()` mutator for consistency/testability, same as `MarkDeleted` — neither is called by the infrastructure layer, which writes `deleted_at` via raw SQL).
4. **Nested categories are filtered by `deleted_at` at the JOIN, not in application code** — the old TS `mapToDomainWithCategories` fetched `categories(*, group_categories(*))` per join row and then dropped any category that was itself soft-deleted (`if (!cat || cat.deleted_at) return null`). Go's `fetchCategoriesForProjectTypes` adds `AND c.deleted_at IS NULL` to the `JOIN categories c` clause instead, which is equivalent — `category`'s own `SoftDelete` already hard-deletes `project_type_category` rows transactionally (see `docs/category.md`), so a soft-deleted category should never actually appear via this join in practice; the filter is defense-in-depth, not load-bearing.
5. **`UpdateProjectTypeInput.CategoryIDs` is pointer-to-slice, not a plain slice** — `nil` means "leave relations untouched" (field omitted from the JSON body), a non-nil pointer (including one to an empty slice) means "replace with this set". Mirrors `project`'s `Categories`/`ServiceIDs` convention exactly, and the old TS repository's `input.categoryIds !== undefined` check.
6. **`updated_at` is set by a DB trigger (`update_service_type_updated_at`, legacy name)** — Go still sets it in-memory for consistency, but the trigger overwrites it on every `UPDATE`; queries use `RETURNING` so the value returned to the client is always the real post-trigger value.

## Production bug found and fixed during migration audit

`project_type` had **two** triggers both syncing into the shared `slug_registry` table on every INSERT/UPDATE/DELETE:
- `trg_sync_service_type_slug` (legacy name) → inserted `entity_type = 'service_type'`.
- `trigger_sync_project_type_slug` (current name) → inserts `entity_type = 'project_type'`.

`slug_registry_entity_type_check` only allows `'project_type'`, not `'service_type'` — so **every insert into `project_type` was failing outright**, in production, independent of any Go code (reproduced with a raw `INSERT INTO project_type (name, slug) VALUES (...)` via `psql` before writing a single line of Go). `trigger_sync_project_type_slug` already fully superseded the legacy trigger's job with the correct `entity_type`, so the fix was to drop the broken trigger and its now-unused function rather than patch it — see `internal/project-type/migrations/000002_drop_broken_legacy_slug_trigger.up.sql`. Audited `slug_registry` first: zero existing rows with `entity_type = 'service_type'`, and every active `project_type` row already had a correct `entity_type = 'project_type'` registry entry — pure DDL fix, no data repair needed. Verified fixed via `psql` raw insert and the Go integration tests both before (failing) and after (passing) applying the migration.

## HTTP API

Mounted at `/project-types`.

| Method | Path | Body/Query | Response |
|--------|------|------|----------|
| GET | `/project-types` | query: `search`, `limit`, `offset`, `include_deleted` | `[]projectTypeResponse` |
| GET | `/project-types/count` | same filter query params | `{"count": n}` |
| GET | `/project-types/{id}` | — | `projectTypeResponse` or `404` |
| POST | `/project-types` | full create body | `201 projectTypeResponse` |
| PUT | `/project-types/{id}` | partial update body | `200 projectTypeResponse` |
| DELETE | `/project-types/{id}` | — (soft delete) | `204` |

Create/update body: `{"name", "slug", "image", "meta_title", "meta_description", "is_featured", "order_index", "category_ids": ["uuid", ...]}` (all but `name`/`slug` optional on update). No `/slug/{slug}` or `/{id}/restore` routes — the old TS module never had either.

`projectTypeResponse` (JSON, snake_case) carries nested `categories` (`{id,name,group_id,slug,image_url,meta_title,meta_description,is_featured,order_index,created_at,updated_at,deleted_at,group:{id,name,slug,image_url,meta_title,meta_description,is_featured,order_index}|null}[]`). Create/Update responses don't include this join (bare entity, empty `categories: []`) — same convention as `category`'s `toBareCategoryResponse`.

## How `elc-tem` calls this

`modules/project-type/presentation/actions.ts` calls the Go API over HTTP using `fetch` with `process.env.GO_API_URL`, wrapped in `try/catch` (PR #588 lesson). Exports: `getProjectTypesAction`, `getProjectTypeByIdAction` (new — see below), `createProjectTypeAction`, `updateProjectTypeAction`, `deleteProjectTypeAction`.

Consumers fixed during this cutover (previously imported `projectTypeRepo`/application functions directly, bypassing Server Actions):
- `modules/project/presentation/components/public/ProjectListModule.tsx` — was calling `getProjectTypes(projectTypeRepo)` directly; now calls `getProjectTypesAction().then(unwrapActionResult)`.
- `modules/project/presentation/resolveProjectPath.ts` — this was the one place `docs/project.md` flagged as "will collapse to pure Go once project-type migrates": its `project_type` branch was querying `project_type`/`project_type_category`/`categories`/`group_categories` directly via Supabase (a ~150-line inline mapper) to resolve a `project_type` entity by id after a `slug_registry` lookup. Replaced with a single `getProjectTypeByIdAction(id)` call — added specifically for this caller, since the old TS module never needed a `getById` action (the admin UI only ever called `getAll`). `slug_registry` itself is still read directly from Supabase here — it's a shared cross-module table with no owning Go module yet, same reasoning as `docs/project.md`'s note on this file.

Read-only direct-Supabase queries against `project_type` in cross-cutting utility endpoints (`app/llms.txt/route.ts`, `app/llms-full.txt/route.ts`, `app/sitemap/posts.xml/route.ts`, `shared/components/layout/user/header/search-actions.ts`) were **not** rewired to the Go API — this is the same accepted, pre-existing pattern already applied to every other fully-migrated table (`categories`, `products`, `services`, `news`, etc. are read the same way in these same files). Not a gap introduced by this migration.

## Testing

- `go test ./internal/project-type/...` — domain + application unit tests (fake repository), including the nil-vs-empty categories distinction (rule 5 above).
- `go test -tags=integration ./internal/project-type/infrastructure/...` — CRUD round-trip with a real attached category, `SoftDelete`'s 3-table transaction (join table cleared, referencing `projects.project_type_id` nulled) — run against the real DB.
- Manually verified via `curl` against a running `air`-reloaded server (create with `category_ids`, get-by-id showing nested category+group, update, delete, 404-after-delete), and via `pnpm build` (full static generation, including `/du-an`, `/du-an/[slug]`, `/admin/project-types`).
