# Module: brand

**Status**: Migrated to Go. Admin CRUD and every real consumer in `elc-tem`
go through Go now.
**Purpose**: Manufacturer/brand records shown on the public site (product
filters, the homepage brand showcase, `/san-pham/all/{slug}` pages) and
managed in the admin dashboard.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as
`modules/brand/{domain,application,infrastructure,presentation}`, querying
Supabase directly via `supabase-js`. `application/` and `infrastructure/`
have been deleted from `elc-tem` — `modules/brand/domain/` (TS types +
zod validators, still used by `useBrandForm.ts`'s client-side form
validation) and `modules/brand/presentation/` (Server Actions + admin
components) are all that remain.

## Data model

Table `brands` (already existed in Supabase before migration — the Go
migration in `internal/brand/migrations/` is an `IF NOT EXISTS` baseline,
not a fresh creation):

| Column              | Type        | Notes |
|---------------------|-------------|-------|
| `id`                | uuid        | PK, `gen_random_uuid()` default |
| `name`              | text        | not null, **globally UNIQUE — including soft-deleted rows** (see gotcha below) |
| `slug`              | text        | not null, unique only among **non-deleted** rows (partial index `brands_slug_unique_active`) |
| `logo_url`          | text        | not null, default `''` |
| `meta_title`        | text        | nullable |
| `meta_description`  | text        | nullable |
| `is_featured`       | boolean     | default `false` |
| `order_index`       | integer     | default `0` |
| `content`           | jsonb       | nullable — Tiptap rich text, passed through opaquely as `json.RawMessage`, Go never parses its structure |
| `faq`               | jsonb       | default `'[]'::jsonb`, nullable — `[]FAQItem{Question, Answer}` |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete |

**Referenced by**: `products.brand_id` — FK `ON DELETE SET NULL`, but
`products.brand_id` is itself **`NOT NULL`**. These two constraints
contradict each other: a real hard `DELETE` on a referenced brand would
never actually reach the "set null" step, because Postgres would first
reject the would-be `NULL` write against the `NOT NULL` column. This was
confirmed via `psql \d products`/`\d brands` before writing any Go code —
the TS types (`database.types.ts`) don't surface FK actions or NOT NULL
combinations like this at all. Practical consequence: **`SoftDelete` here
does not, and cannot, clean up `products.brand_id`** — unlike
`service-group`'s `SoftDelete`, which nulls out `services.group_id` because
that column *is* nullable. Every product keeps pointing at its brand's row
even after the brand is soft-deleted; this matches the old TS `delete()`
behavior exactly (it never touched `products` either). Confirmed by
`TestPostgresBrandRepository_SoftDeleteLeavesProductBrandIDIntact`.

**Trigger**: `trg_brand_slug_registry` keeps the shared `slug_registry`
table in sync on INSERT/UPDATE/DELETE, same pattern as
`trigger_sync_service_group_slug`/`trigger_sync_service_slug` — runs at the
DB level regardless of client, no Go code needed for it.

## Business rules that are easy to miss

1. **No "resurrect on create" — unlike `service-group`.** `service_groups.slug`
   is a plain column-level `UNIQUE` (blocks even soft-deleted rows), which is
   why that module's `Create` resurrects a soft-deleted row instead of
   inserting. `brands.slug` is only unique among **non-deleted** rows (a
   partial index), so a brand's `Create` here is a plain `INSERT` — creating
   a new brand that reuses a soft-deleted brand's slug just works, no
   conflict, no resurrect logic. Verified in
   `TestPostgresBrandRepository_CRUD`.
2. **`brands.name`, however, IS a full unique constraint** (`brands_name_key`,
   no partial `WHERE` clause) — it blocks reuse even against soft-deleted
   rows. This is a pre-existing landmine carried over unchanged from the old
   TS behavior (the old `SupabaseBrandRepository` had no handling for it
   either): re-creating a brand with the same `name` as a soft-deleted one
   fails with a raw Postgres unique-violation, surfaced to the client as a
   generic 500 (no repository in this codebase currently maps unique
   violations to `apperr.NewConflictError` — see `internal/contact`,
   `internal/service-group`, `internal/service`, none of them do this
   either). Deliberately **not** fixed here — that would be a behavior
   change beyond faithfully migrating what already existed, and no admin
   workflow in `elc-tem` currently soft-deletes and re-creates same-named
   brands. Flagging so it isn't rediscovered as a "new" Go bug later.
3. **Soft delete does not cascade anywhere** — see the FK section above.
   This is the one place this module's `SoftDelete` deliberately does
   *less* than `service-group`'s, and it's because the schema (NOT NULL
   FK column) makes the "more" version physically impossible, not an
   oversight.

## Migration tracking gotcha (already fixed at the platform level)

Every module gets its own `schema_migrations_<module>` tracking table via
`?x-migrations-table=...` in the Makefile — already wired in, nothing
module-specific needed here. See `ARCHITECTURE.md` §7 and
`docs/service-group.md`.

## pgx simple-protocol gotcha for `jsonb` struct slices (new, specific to this module)

The shared pool runs with `pgx.QueryExecModeSimpleProtocol` (the
`contact`/PgBouncer fix — see `docs/contact.md`). That mode can infer how to
encode/decode a `jsonb` column into `json.RawMessage`/`[]byte` (used for
`content`, and for `service`'s `content` column) without help, but it
**cannot** infer an OID for an arbitrary Go struct slice like
`[]domain.FAQItem`. Two symptoms hit while writing the integration test,
before any code was assumed to be correct:

- Encoding `[]domain.FAQItem` directly as a query arg failed with
  `unable to encode ... into text format for unknown type (OID 0)`.
- Encoding a plain `[]byte` (from a hand-rolled `json.Marshal`) *also*
  failed, with `invalid input syntax for type json` — pgx encodes a bare
  `[]byte` as a bytea literal, not raw JSON text, and Postgres then rejects
  casting bytea to jsonb.

Fix: marshal/unmarshal `faq` by hand in
`internal/brand/infrastructure/postgres_repository.go`
(`marshalFAQ`/`unmarshalFAQ`), and make sure the marshal function's return
type is the named `json.RawMessage` type specifically — not a plain
`[]byte` — since pgx special-cases `json.RawMessage` for jsonb columns.
Any future module storing a typed struct (not `json.RawMessage`/`[]string`)
in a jsonb column will hit the same thing.

## HTTP API

Mounted at `/brands`.

| Method | Path                  | Body                                                                    | Response |
|--------|-----------------------|--------------------------------------------------------------------------|----------|
| GET    | `/brands`             | query: `search`, `limit`, `offset`, `include_deleted`                      | `[]brandResponse` |
| GET    | `/brands/{id}`        | —                                                                          | `brandResponse` or `404` |
| GET    | `/brands/slug/{slug}` | —                                                                          | `brandResponse` or `404` |
| POST   | `/brands`             | `{name, slug, logo_url, meta_title, meta_description, is_featured, order_index, content, faq}` | `201 brandResponse` |
| PUT    | `/brands/{id}`        | any subset of the above fields (partial update)                            | `200 brandResponse` |
| DELETE | `/brands/{id}`        | — (soft delete)                                                            | `204` |
| POST   | `/brands/{id}/restore`| —                                                                          | `204` |

`brandResponse` (JSON, snake_case):
```json
{
  "id": "uuid",
  "name": "Daikin",
  "slug": "daikin",
  "logo_url": "https://.../logo.webp",
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

`slug` lookup and `restore` are available on the Go side but have no TS
caller today — same situation as `service-group` (see `docs/service-group.md`):
the old TS code never exposed a `getBrandBySlugAction`/restore Server
Action either, so nothing regresses by not adding one now.

## How `elc-tem` calls this

`modules/brand/presentation/actions.ts` is a thin adapter (same shape as
`contact`/`service-group`): `getBrandsAction`, `getBrandByIdAction`,
`createBrandAction`, `updateBrandAction`, `deleteBrandAction` call the
endpoints above via `fetch(process.env.GO_API_URL + ...)`, using
`toSnakeCaseBody` for request bodies and a `mapGoBrand` function for the
response. Error messages stay **Vietnamese** (unlike `contact`/
`service-group`'s actions, which use English strings) — per
`ARCHITECTURE.md` §9, user-facing strings belong to the Next.js side and
this module's admin UI already showed Vietnamese toasts before migration;
no reason to regress that.

Consumers fixed during this cutover (found by grepping the whole of
`elc-tem` for `modules/brand/application`, `modules/brand/infrastructure`,
`brandRepo`, and `modules/brand/domain`/`Brand` type imports, before
deleting anything):

- `app/(public)/page.tsx` — homepage (`getCachedHomeData`) called
  `getBrands(brandRepo, {limit: 100})` directly instead of going through
  the module's own Server Action. Switched to
  `getBrandsAction({ limit: 100 })`, same pattern already used there for
  `getContactsAction()`.
- `modules/dashboard/{application,presentation}` — `getDashboardStats`
  took a DIP-injected `BrandRepository` for `.count()` and `.getAll()`.
  Since the full list was needed anyway (for `brandDistribution`), removed
  `brandRepo` from `DashboardRepositories` entirely and changed
  `getDashboardStats` to take a pre-fetched `allBrands: Brand[]` parameter,
  same pattern as the existing `contactsCount`/`servicesCount` parameters
  for the other Go-migrated modules. `presentation/actions.ts` now calls
  `getBrandsAction()` alongside `getContactsAction()`/the services-count
  fetch. **Side effect, not a regression**: the old `brandRepo.count()`
  never filtered `deleted_at` (inconsistent with `getAll()`, which did) —
  the dashboard brand count silently included soft-deleted brands before.
  Using `allBrands.length` from `getBrandsAction()` (active-only by
  default) fixes that inconsistency, aligning brand's dashboard count with
  every other entity's (which all count active rows only).
- `modules/catalog/presentation/actions.ts` had its own **duplicate**
  `getBrandsAction`/`createBrandAction`, importing `brandRepo`/`application`
  directly instead of reusing the brand module's own actions. Audit found
  `createBrandAction` there had **zero callers anywhere** (pure dead code —
  deleted outright); `getBrandsAction` there was called only from
  `ProductManagement.tsx` (brand dropdown in the product form) — switched
  that one call site to import `getBrandsAction` from
  `@/modules/brand/presentation/actions` instead, then deleted both
  duplicate functions and their now-unused imports from catalog's
  `actions.ts`.
- `modules/catalog/domain/types.ts`, `shared/components/sections/
  brand-showcase.tsx`, `app/(admin)/admin/(dashboard)/brands/page.tsx` —
  type-only imports of `Brand`/`CreateBrandInput`/`UpdateBrandInput` from
  `modules/brand/domain`, or the `BrandManagement` presentation component.
  Left untouched — same as `contact`'s `getContactHref` helper staying in
  `elc-tem`, per `ARCHITECTURE.md` §2, `domain`/pure-presentation code isn't
  something that needs a Go equivalent.

## Dead code found during audit (documented, not silently dropped)

- `getBrandBySlug` (old `modules/brand/application/getBrandBySlug.ts`) had
  **zero callers** — not even wired into a Server Action in the old
  `actions.ts`. Confirmed via grep before deleting. No public "brand detail
  page by slug" exists in `elc-tem` today; if one is added later, it should
  call a new `getBrandBySlugAction` backed by the already-implemented
  `GET /brands/slug/{slug}` endpoint above.
- `BrandRepository.getByIds(ids)` (old TS repository interface) had zero
  callers anywhere in `elc-tem` — not implemented on the Go side, for the
  same "no premature abstraction" reason `contact` skipped a dedicated
  `/count` endpoint (see `docs/contact.md`). Add it if/when `catalog`
  (product↔brand batch lookups) actually migrates and needs it.

## Not migrated (out of scope for this module)

`catalog` itself is not migrated yet (still last in the migration order —
see `ARCHITECTURE.md` §1). Its own infrastructure queries the `brands`
table directly via `supabase-js` as part of building product↔brand joins —
this is `catalog`'s own scope, not brand CRUD, same precedent as
`contact.md`'s note about `ProductDetailModule`/`ServiceDetailModule`:
- `modules/catalog/infrastructure/SupabaseProductRepository.ts`,
  `modules/catalog/infrastructure/resolveProductPath.ts` — join `products`
  with `brands` to build `ProductWithRelations`/resolve canonical URLs.
- `modules/catalog/presentation/components/{ProductFilterMobile,
  ProductFilters}.tsx` — read `brands` directly to build filter dropdown
  options.
- `app/llms.txt/route.ts`, `app/llms-full.txt/route.ts`,
  `app/api/products/route.ts`,
  `shared/components/layout/user/infinite-product-grid.tsx`,
  `modules/settings/application/getPublicLayoutData.ts` — all read
  products joined with brands for unrelated (SEO feed / infinite scroll /
  layout data) purposes.
- `scratch/*.ts` — one-off maintenance scripts, not part of the running
  application.

These will move to Go's `/brands` (and `catalog`'s own future endpoints)
when `catalog` itself migrates, not before.

## Testing

- `go test ./internal/brand/...` — domain + application unit tests (fake
  repository, no network).
- `go test -tags=integration ./internal/brand/infrastructure/...` — real
  DB roundtrip: create/read/update/soft-delete/restore, the FAQ jsonb
  round-trip, the slug-reuse-after-soft-delete behavior (see gotcha #1
  above), and a dedicated test proving `products.brand_id` is left intact
  after a brand soft delete (see FK section above). Self-cleaning.
- TS side: `modules/brand/__tests__/domain/validators.test.ts` is the only
  surviving test (client-side zod validation used by `useBrandForm.ts`);
  the old `application`/`infrastructure`/`presentation` test files were
  deleted along with their source, matching the `contact` precedent.

## Deploy / ops notes specific to this module

None beyond the general ones in `ARCHITECTURE.md` — no new platform-level
fix was needed (the pgx jsonb-struct-slice handling above is module-local,
in `internal/brand/infrastructure`, not a shared `platform/` change).
