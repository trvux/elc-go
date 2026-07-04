# Module: contact

**Status**: Migrated to Go (pilot module for the whole Next.js → Go migration).
**Purpose**: Manages the contact channels shown across the public site
(phone, Zalo, Messenger, Facebook, email, ...) and the admin CRUD screen
that manages them.

## What it replaced

Before migration, this lived entirely in `elc-tem` (Next.js) as
`modules/contact/{domain,application,infrastructure,presentation}`, querying
Supabase directly via `supabase-js`. That code has been deleted from
`elc-tem` — `application/` and `infrastructure/` no longer exist there.
`elc-tem` now keeps only:
- `modules/contact/domain/` — TS types (`Contact`, `ContactFilter`) and pure
  UI helpers (`getContactHref`, `getDisplayContacts`) still used by
  components for sorting/typing, no DB access.
- `modules/contact/presentation/` — Server Actions that call this Go
  service, plus the React components.

## Data model

Table `contacts` (already existed in Supabase before migration — the Go
migration in `internal/contact/migrations/` is an `IF NOT EXISTS` baseline,
not a fresh creation):

| Column        | Type    | Notes                          |
|---------------|---------|---------------------------------|
| `id`          | uuid    | PK, `gen_random_uuid()` default |
| `type`        | text    | not null — `phone`, `email`, `facebook`, `messenger`, `zalo`, `tiktok`, `youtube`, `website` |
| `label`       | text    | nullable, display label         |
| `value`       | text    | not null — the raw phone/handle/email |
| `is_active`   | boolean | default `true`                  |
| `order_index` | integer | default `0`                     |

Note: Supabase RLS policies exist on this table (`Auth full access`,
`Public read`) but **do not apply** to this service — `pgx` connects
directly, bypassing PostgREST/RLS entirely. Authorization for write
operations must eventually be enforced in this module's application layer
once the `auth` module is migrated; for now, the admin routes are still
gated by Next.js's own Supabase-Auth middleware in `elc-tem`.

## Architecture (see `ARCHITECTURE.md` for the general rules)

```
internal/contact/
  domain/
    types.go        — Contact entity (rich domain model), CreateContactInput,
                       UpdateContactInput, ContactFilter
    types_test.go
    repository.go    — ContactRepository interface
  application/
    create_contact.go, update_contact.go, delete_contact.go, get_contacts.go
    *_test.go         — use a fake in-memory repository (fake_repository_test.go),
                        no real DB needed
  infrastructure/
    postgres_repository.go              — pgx implementation
    postgres_repository_integration_test.go — hits the real Supabase DB,
                        run explicitly with `go test -tags=integration ./internal/contact/infrastructure/...`
  presentation/
    dto.go, handler.go, routes.go
  migrations/
    000001_baseline_contacts.{up,down}.sql
```

`Contact` is a rich domain entity: unexported fields, two constructors
(`NewContact` validates for new data, `RehydrateContact` skips validation for
data loaded from the DB), and methods for mutation (`UpdateValue`,
`UpdateLabel`, ...) and derived values (`Href()`, `IsExternal()` — computed,
never stored, so they can't go stale relative to `type`/`value`).

## HTTP API

Mounted at `/contacts` (see `cmd/server/main.go` for how it's wired into the
shared chi router).

| Method | Path             | Body                                                        | Response |
|--------|------------------|--------------------------------------------------------------|----------|
| GET    | `/contacts`      | — (query params: `type`, `search`)                            | `[]contactResponse` |
| GET    | `/contacts/{id}` | —                                                              | `contactResponse` or `404` |
| POST   | `/contacts`      | `{type, label, value, is_active, order_index}`                | `201 contactResponse` |
| PUT    | `/contacts/{id}` | any subset of the above fields (partial update)                | `200 contactResponse` |
| DELETE | `/contacts/{id}` | —                                                              | `204` |

`contactResponse` (JSON, snake_case):
```json
{
  "id": "uuid",
  "type": "phone",
  "label": "Hotline",
  "value": "0901234567",
  "is_active": true,
  "order_index": 0,
  "href": "tel:0901234567",
  "is_external": false
}
```

Errors are always `{"code", "message", "fields"?}` with the matching HTTP
status (400 validation, 404 not found, 500 internal) — see
`internal/platform/apperr` / `internal/platform/httpserver`.

## How `elc-tem` calls this

`modules/contact/presentation/actions.ts` in `elc-tem` is a thin adapter:
Server Actions `getContactsAction`, `createContactAction`,
`updateContactAction`, `deleteContactAction` call the endpoints above via
`fetch(process.env.GO_API_URL + ...)` and map the snake_case JSON back into
the `Contact` TS type the rest of the frontend already expects — no React
component had to change.

Consumers of `getContactsAction` in `elc-tem`:
- `app/(public)/page.tsx` — homepage (`getCachedHomeData`), feeds
  `HeroContactActions` (call/Zalo buttons) and `CTASection`.
- `modules/contact/presentation/components/ContactManagement.tsx` — admin
  CRUD screen (`/admin/contacts`).
- `modules/dashboard/presentation/actions.ts` — dashboard stats card uses
  `contacts.length` as the count (no dedicated count endpoint on the Go
  side; fetching the full list is cheap enough for this table's size).

Not yet touched (out of scope for this module, belongs to `catalog`/
`service`/`settings` when those migrate): a few pages query the `contacts`
table directly via `supabase-js` for unrelated reasons (they only borrow the
pure `getContactHref` TS helper) — see `getPublicLayoutData.ts`,
`ProductDetailModule.tsx`, `ServiceDetailModule.tsx`.

## Known gotcha (already fixed, keep in mind for future modules)

pgx's default query mode caches prepared statements per connection. Supabase
is reached through a **transaction-mode pooler** (PgBouncer), which doesn't
preserve that state across pooled connections — this caused intermittent
`prepared statement "..." already exists` errors in production. Fixed in
`internal/platform/db/db.go` by setting
`config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol`.
This applies to the shared pool, so every future module already gets this
fix for free — no per-module action needed, just don't revert it.

## Testing

- `go test ./internal/contact/...` — domain + application unit tests (fake
  repository, no network).
- `go test -tags=integration ./internal/contact/infrastructure/...` — real
  DB roundtrip (create/read/update, self-cleaning). Requires `DATABASE_URL`
  in the environment. Not run by default `go test ./...`.

## Deploy / ops notes specific to this module

None beyond the general ones in `ARCHITECTURE.md` — this was the pilot, so
anything module-specific turned out to be either a platform-level fix (the
pgx gotcha above) or a one-time setup cost (Docker on the VPS, the
IPv4-shared-pooler connection string, GitHub Actions secrets) that every
future module benefits from without repeating.
