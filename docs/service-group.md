# Module: service-group

**Status**: Fully migrated. Admin CRUD and every public read path in
`elc-tem` go through Go now.
**Purpose**: Groups of services shown on the public site (`/dich-vu`) and
managed in the admin dashboard.

## Resolved: TS `application/`/`infrastructure/` are deleted

For a while after this module's own cutover, `modules/service-group/
{application,infrastructure}` in `elc-tem` had to stay alive — first
because the not-yet-migrated `service` module depended on them directly,
then (after `service` migrated) because `service`'s
`getPublishedServicesGrouped()` helper still did a cross-repo join reading
both `service` and `service-group` via TS/Supabase, used by the public
`app/(public)/dich-vu/*` pages.

Both were resolved in the same follow-up pass: `getPublishedServicesGrouped`
was rewritten as `getPublishedServicesGroupedAction()` in `modules/service/
presentation/actions.ts`, composing `getServicesAction()` and
`getServiceGroupsAction()` (both already Go-backed) instead of two
Supabase repos. Once every caller was switched over,
`modules/service-group/{application,infrastructure}` had zero remaining
importers and were deleted outright — confirmed via:
```
grep -rln "modules/service-group/application\|modules/service-group/infrastructure" --include="*.ts" --include="*.tsx" .
```
(empty result before deleting).

## Data model

Table `service_groups` (pre-existing in Supabase):

| Column              | Type        | Notes |
|---------------------|-------------|-------|
| `id`                | uuid        | PK, `uuid_generate_v4()` default |
| `name`              | text        | not null |
| `slug`              | text        | not null, **UNIQUE** — see resurrect rule below |
| `image_url`         | text        | nullable |
| `meta_title`        | text        | nullable |
| `meta_description`  | text        | nullable |
| `is_featured`       | boolean     | default `false` |
| `order_index`       | integer     | default `0` |
| `category_ids`      | uuid[]      | plain Postgres array, default `'{}'` — **no FK**, referential integrity is app-level only |
| `created_at`        | timestamptz | default `now()` |
| `updated_at`        | timestamptz | default `now()` at insert only — **no auto-update trigger**, every mutator in the Go entity sets it by hand |
| `deleted_at`        | timestamptz | nullable — soft delete |

**Foreign key pointing in**: `services.group_id` → `service_groups.id`
(`ON DELETE SET NULL`). `service` is also migrated (see `docs/service.md`)
— both modules' Go repositories now share the same underlying tables.

**Trigger**: `trigger_sync_service_group_slug` keeps the shared
`slug_registry` table in sync on INSERT/UPDATE/DELETE. Runs at the DB level
regardless of client (Go or Supabase) — no application code needed for it.
A `check_slug_conflict()` function also exists in the DB (cross-entity slug
uniqueness check) but is **not called anywhere** in the current codebase —
confirmed orphaned, not replicated in Go on purpose.

## Business rules that are easy to miss (both replicated in Go)

1. **"Resurrect" a soft-deleted row on create.** Because `slug` is globally
   unique even for soft-deleted rows, `Create` first checks for an existing
   soft-deleted row with the same slug and updates that row (clearing
   `deleted_at`) instead of inserting — otherwise a repeat slug would hit
   the unique constraint. See `infrastructure/postgres_repository.go`.
2. **Soft delete cascades to `services.group_id = NULL` by hand.** The FK's
   `ON DELETE SET NULL` only fires on a real `DELETE`, not on the `UPDATE
   deleted_at = now()` a soft delete actually is. `SoftDelete` runs both
   writes (the group's `deleted_at` and the `services` cleanup) in **one
   transaction** so they can't partially apply.

## Migration tracking gotcha (fixed, applies to every future module)

`golang-migrate`'s `schema_migrations` table is shared per-database, not
per-path. Since every module's migrations restart numbering at `000001`,
running a second module's baseline against the plain `DATABASE_URL` gets
silently skipped ("no change") because the tracker already thinks version 1
ran. Fixed by giving each module its own tracking table via
`?x-migrations-table=schema_migrations_<module>` — already wired into the
Makefile, nothing to redo for future modules. See `ARCHITECTURE.md` section 7.

## HTTP API

Mounted at `/service-groups`.

| Method | Path                          | Body                                      | Response |
|--------|-------------------------------|--------------------------------------------|----------|
| GET    | `/service-groups`             | query: `include_deleted`, `is_featured`    | `[]serviceGroupResponse` |
| GET    | `/service-groups/{id}`        | —                                           | `serviceGroupResponse` or `404` |
| GET    | `/service-groups/slug/{slug}` | —                                           | `serviceGroupResponse` or `404` |
| POST   | `/service-groups`             | full create payload                        | `201` |
| PUT    | `/service-groups/{id}`        | partial update payload                     | `200` |
| DELETE | `/service-groups/{id}`        | — (soft delete)                            | `204` |
| POST   | `/service-groups/{id}/restore`| —                                           | `204` |

Not all of these are called from `elc-tem` yet — `getServiceGroupsAction`,
`createServiceGroupAction`, `updateServiceGroupAction`,
`deleteServiceGroupAction` exist in `presentation/actions.ts`; slug lookup
and restore are available on the Go side but have no TS caller today (the
old TS code didn't expose them as Server Actions either).

## Testing

- `go test ./internal/service-group/...` — domain + application unit tests
  (fake in-memory repository).
- `go test -tags=integration ./internal/service-group/infrastructure/...` —
  real DB roundtrip covering create, update, soft delete, and specifically
  the resurrect-on-create rule. Self-cleaning.
