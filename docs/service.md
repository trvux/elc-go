# Module: service

**Status**: Fully migrated. `modules/service/{application,infrastructure}`
in `elc-tem` are deleted — every consumer goes through Go now.
**Purpose**: Services (dịch vụ) shown on `/dich-vu` and managed in the admin
dashboard. Sibling of `service-group` (each service optionally belongs to a
group and a category).

## History: this took two passes to fully clean up

The first pass only cut over `modules/service/presentation/actions.ts`
(admin CRUD) and the dashboard count. Several other consumers kept using
`serviceRepo`/`modules/service/application` directly, because they needed
things the admin actions didn't expose yet:

- `app/(public)/dich-vu/page.tsx`, `dich-vu/[slug]/page.tsx`,
  `dich-vu/[slug]/[location]/page.tsx` called
  `getPublishedServicesGrouped(serviceRepo, serviceGroupRepo)` — a cross-repo
  read (service + service-group) with no Go equivalent yet.
- `modules/project/presentation/components/public/ProjectListModule.tsx` and
  `app/(public)/du-an/[slug]/page.tsx` (the `project` module, itself not
  migrated) read services directly for project↔service cross-filtering.
- `shared/components/layout/user/header/search-actions.ts` (header search)
  read the published service list directly.
- `modules/service/presentation/components/public/ServiceDetailModule.tsx`
  called `getAdjacentServices(serviceRepo, ...)` for prev/next navigation.

**Second pass (this is now done):** added the missing pieces to
`presentation/actions.ts` — `getServiceBySlugAction`,
`getAdjacentServicesAction` (same "prefer same-group siblings, fall back to
full list, sort featured-first" logic as before), and
`getPublishedServicesGroupedAction` (composes `getServicesAction()` +
`service-group`'s `getServiceGroupsAction()` — two already-Go-backed
actions, no new Go endpoint needed). Updated all five call sites above to
use these instead of importing `serviceRepo`/`application` directly. Once
nothing did anymore (verified by grep), deleted
`modules/service/{application,infrastructure}` and
`modules/service-group/{application,infrastructure}` outright — the latter
had no callers left either, for the same reason (see `docs/service-group.md`).

`project` itself is still not migrated — it now calls `service`'s Go-backed
Server Actions instead of Supabase directly, the same pattern
`dashboard`/`contact` already used. That's an ordinary cross-module Server
Action call, not a lingering TS/Go split like before.

## Data model

Table `services` (pre-existing in Supabase):

| Column               | Type        | Notes |
|----------------------|-------------|-------|
| `id`                 | uuid        | PK |
| `title`              | text        | not null |
| `slug`               | text        | not null, UNIQUE |
| `group_id`           | uuid        | FK → `service_groups(id)` ON DELETE SET NULL |
| `category_id`        | uuid        | FK → `categories(id)` ON DELETE SET NULL |
| `original_price`     | bigint      | nullable |
| `sale_price`         | bigint      | nullable — **written for compatibility, never read back**; see below |
| `discount_percent`   | integer     | nullable |
| `price_display_text` | text        | nullable |
| `labels`             | text[]      | nullable |
| `description`        | text        | nullable |
| `content`            | jsonb       | nullable — Tiptap rich text, passed through opaquely as `json.RawMessage`, Go never parses its structure |
| `image`, `meta_title`, `meta_description` | text | nullable |
| `is_featured`, `is_published` | boolean | |
| `order_index`        | integer     | |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete, same pattern as `service-group` |

**Referenced by**: `project_service.service_id` (FK `ON DELETE CASCADE`,
`project` module not migrated — be aware before changing this table's shape
or doing a hard delete).

**Trigger**: `trigger_sync_service_slug` keeps `slug_registry` in sync,
runs at the DB level regardless of client — no Go code needed for it.

## Design decisions worth knowing

**Only a 2-level join, not 3.** The old TS code's `ServiceWithRelations`
joined `service → category → category.group` (3 tables). An audit before
writing the Go version confirmed `category.group` is constructed by the old
repository but **never read by any component**. Go joins `service → group`
and `service → category` (2 tables, `GroupRef`/`CategoryRef` with just
`{id, name}` — the only fields any UI reads) and stops there. If something
later needs `category.group`, add it back deliberately, with a real caller
in hand.

**`SalePrice()` is a computed method, never stored/read as independent
state — a deliberate bug fix, not just a port.** The old TS code stored
`sale_price` in the DB and had **two different call sites compute the
"final price" two different ways**: `ServiceDetailModule.tsx` read the
stored `sale_price` column; `ServiceColumns.tsx`/`mappers.ts` (admin table +
public cards) recomputed it client-side from `original_price`/
`discount_percent`, ignoring the stored value. On top of that, updating only
`discountPercent` without `originalPrice` in the same call left the stored
`sale_price` stale (never recalculated). The Go entity now computes
`SalePrice()` fresh every time from `OriginalPrice`/`DiscountPercent` — it
is still **written** to the `sale_price` column on create/update (for any
other consumer that might read the raw column — none found, but harmless to
keep populated) but Go itself never reads that column back. This removes
the whole bug class permanently. `UpdatePricing(originalPrice,
discountPercent)` takes both fields together on purpose: the application
layer resolves whichever one *isn't* in a partial update request to its
current value before calling it, so `SalePrice()` is always computed from a
consistent pair — see `application/update_service.go`.

**Clients cannot set `sale_price` at all** — it's not a field on
`CreateServiceInput`/`UpdateServiceInput` on either the Go or TS side.

## HTTP API

Mounted at `/services`.

| Method | Path                    | Notes |
|--------|--------------------------|-------|
| GET    | `/services`              | query: `group_id`, `category_id`, `is_featured`, `is_published`, `search`, `include_deleted` |
| GET    | `/services/count`        | same query filters, `{"count": N}` — used by dashboard |
| GET    | `/services/{id}`         | joined (`group`, `category`) |
| GET    | `/services/slug/{slug}`  | joined |
| POST   | `/services`              | plain (no join) response |
| PUT    | `/services/{id}`         | partial update, plain response |
| DELETE | `/services/{id}`         | soft delete |
| POST   | `/services/{id}/restore` | |

## Testing

- `go test ./internal/service/...` — domain (including `SalePrice()` and the
  partial-update-never-stale invariant) + application unit tests.
- `go test -tags=integration ./internal/service/infrastructure/...` — two
  tests: full CRUD roundtrip (including jsonb `content` and `labels`
  array), and a dedicated join test that verifies `Group`/`Category` resolve
  correctly against real `service_groups`/`categories` rows.
