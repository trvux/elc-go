# elc-go — Architecture & Conventions

This document is the source of truth for how this project is built. Any AI or
human contributing to this repo should read this file first. Rules here are
binding unless a future decision explicitly supersedes them (update this file
when that happens — do not let it go stale).

## 1. Purpose & Migration Strategy

This service replaces, incrementally:
- Supabase (Postgres access, Auth, Storage) — currently used directly from
  the Next.js codebase (`elc-tem`).
- The business logic currently living inside Next.js (`modules/*/domain`,
  `application`, `infrastructure` in `elc-tem`).

Once a module is fully migrated, Next.js (`elc-tem`) becomes presentation-only
for that module: its Server Actions call this Go API over HTTP instead of
importing use cases in-process.

**Strategy: Strangler Fig.** Migrate one module at a time, cut over as soon as
that module works, never big-bang. Both systems run in parallel during the
transition — migrated modules call this API; non-migrated modules keep
working exactly as they do today in `elc-tem`.

**Supabase Cloud stays as-is for the entire migration.** Decided explicitly:
no self-hosting of the OSS Supabase stack, no early infra cutover. Go
(`pgx`) connects directly to the same Supabase-hosted Postgres throughout —
this is intentional, not a shortcut (see §7). Supabase Pro keeps running,
and its cost is accepted, until Go has fully replaced every module
**including `auth` and Storage** — only then is Supabase cancelled and cut
over completely. Do not propose partial/early Supabase cutover; this has
already been decided.

**Migration order:**
1. `contact` — pilot. Low risk, few dependencies. Proves the pipeline
   (Go + Postgres + Docker + Next.js calling out over HTTP) before anything
   harder is attempted.
2. Other simple CRUD modules (brand, group, service-group, ...).
3. `auth` — deliberately not first. Every other module depends on it for
   authorization; if the pipeline itself has problems, better to discover
   that on a low-stakes module first.
4. `catalog` — most complex (search, filters, pricing, related products).
   Migrated last among business modules.
5. Supabase Storage fully replaced (MinIO/R2) once the modules that use file
   upload are migrated.

### Module status

| Module    | Status      | Notes |
|-----------|-------------|-------|
| contact   | in progress | pilot module |
| (others)  | not started | |

## 2. Modular Monolith / DDD Layering

Each business module lives under `internal/<module>/` with 4 layers, mirroring
`modules/<name>/{domain,application,infrastructure,presentation}` in the
Next.js codebase 1:1 so the TS source can be used as a spec during translation:

- **`domain/`** — entities (rich domain model, see §3), repository
  interfaces, domain-level errors. No dependency on anything outside
  `internal/platform/apperr`. No framework, no DB driver, no HTTP.
- **`application/`** — use cases. Orchestrates domain + a repository
  *interface* (never a concrete implementation). Depends only on `domain`.
- **`infrastructure/`** — repository implementations (pgx/Postgres). Depends
  on `domain` to satisfy its interfaces.
- **`presentation/`** — HTTP delivery: chi handlers, request/response DTOs,
  route registration. This is the composition root for the module: it is the
  only layer allowed to know about both `infrastructure` (concrete repo) and
  `application` (use cases), and wires them together.

Cross-cutting, non-business code lives in `internal/platform/` — shared by
every module, never module-specific logic:

- **`apperr/`** — shared error taxonomy, used by every module's domain layer.
- **`logger/`** — zap setup.
- **`db/`** — pgx pool setup.
- **`httpserver/`** — chi router bootstrap, shared middleware (request ID,
  recoverer, centralized error-to-HTTP mapping).

## 3. Rich Domain Model

Entities are not plain data bags. Rules:

- Fields are **unexported** (lowercase). No public setters for raw fields.
- Two constructors per entity:
  - `New<Entity>(...) (*Entity, error)` — validates, used when creating from
    user input. Returns an `apperr` validation error on failure.
  - `Rehydrate<Entity>(...) *Entity` — no validation, used **only** by the
    infrastructure layer to reconstruct an entity from a trusted DB row.
- Behavior lives on the entity as methods. Anything that mutates state uses a
  **pointer receiver** (`func (c *Contact) UpdateValue(v string) error`) —
  a value receiver silently mutates a copy, not the real entity.
- Derived values (e.g. `Href()`, `IsExternal()` on `Contact`) are computed
  methods, never stored fields — they can never go out of sync with the data
  they're derived from.
- Input/filter DTOs (`CreateContactInput`, `ContactFilter`, ...) are plain
  structs at the application/presentation boundary. They carry data only —
  they are not validated themselves; validation happens inside the entity
  constructor/mutator methods they feed into.

## 4. SOLID

- **SRP** — handlers only decode request → call use case → encode response.
  Use cases only orchestrate. Repositories only do I/O. No layer does more
  than one job.
- **OCP / LSP** — repository interfaces are defined in `domain`, satisfied by
  `infrastructure` implementations. Swap the implementation without touching
  `domain` or `application`.
- **ISP** — one repository interface per aggregate. No monolithic
  "god repository" covering unrelated entities.
- **DIP** — `application` depends on `domain` interfaces, never imports
  `infrastructure` directly. `presentation` (composition root) is the only
  place that constructs a concrete infrastructure implementation and injects
  it into a use case.

## 5. Error Handling

`internal/platform/apperr.AppError`:

```go
type AppError struct {
    Code    string
    Message string
    Status  int
    Fields  map[string][]string // validation errors only
    Err     error                // wrapped internal error — never serialized to the client
}
```

Constructors: `NewValidationError`, `NewNotFoundError`, `NewUnauthorizedError`,
`NewConflictError`, `NewInternalError`.

Rules:
- All messages in English.
- Handlers never set HTTP status codes for domain errors by hand. A single
  centralized mapper in `internal/platform/httpserver` uses `errors.As` to
  detect `*apperr.AppError` and writes the right status + JSON body. Anything
  that isn't an `*apperr.AppError` is treated as an unexpected internal error
  (logged with full detail via zap, client gets a generic 500).

## 6. Logging

- `*zap.Logger` (structured, **not** `SugaredLogger`) via
  `internal/platform/logger.New(env)`.
- One base logger is created once, in `cmd/server/main.go` (the process
  composition root), and passed down explicitly. Not accessed as a package
  global — matches the DIP stance in §4.
- Every log line in English. Request-scoped fields (request ID) are attached
  by middleware, not manually re-typed in every handler.

## 7. Database

- Driver: `pgx/v5` + `pgxpool`. No ORM. Hand-written SQL in each module's
  `infrastructure` layer.
- Connection target: Supabase Postgres, but through the **pooler connection
  string (port 6543, PgBouncer transaction mode)**, not the direct 5432
  connection — this service handles many concurrent requests and must not
  exhaust Supabase's direct-connection limit.
- **Use the "Shared Pooler" (IPv4) variant of the connection string, not the
  dedicated pooler.** Supabase's default/dedicated pooler resolves to an
  IPv6 address; Docker containers (and most VPS networks) commonly have no
  IPv6 egress, which fails with `network is unreachable`. In the Supabase
  dashboard: Connect → Direct → Transaction pooler → toggle "Use IPv4
  connection (Shared Pooler)" — free, no paid add-on needed. Host changes
  from `db.<ref>.supabase.co` to `aws-<n>-<region>.pooler.supabase.com`,
  and the user becomes `postgres.<project-ref>` instead of plain `postgres`.
- One shared `*pgxpool.Pool` per process, injected into every repository.
  Never one pool per module.
- **Migrations**: `golang-migrate` CLI. SQL up/down files, sequential naming
  (`-ext sql -seq`). One migrations folder per module:
  `internal/<module>/migrations/`.
  - **Tables that already exist in Supabase** (e.g. `contacts`): the first
    migration for that module is a no-op baseline (`IF NOT EXISTS` guards, or
    an intentionally empty up/down pair). Do not let golang-migrate try to
    recreate a table Supabase already owns. If needed, baseline
    `schema_migrations` with `migrate force <version>` so the tool's state
    matches reality without re-running DDL against a table that's already there.
  - New tables/columns introduced *after* a module has been cut over: real
    migrations, owned by Go from that point on.

Makefile convention (parameterized by module, since every module owns its own
migrations folder):

```makefile
include .env

migrate-up:
	migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir internal/$(module)/migrations -seq $(name)

migrate-force:
	migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" force $(version)
```

Usage: `make migrate-up module=contact`, `make migrate-create module=contact name=baseline`.

## 8. HTTP Layer

- Router: `chi`. Not `gin`/`echo` — keeps handlers as plain `net/http`,
  avoids the temptation to bind/validate inside the delivery layer (that
  belongs in `domain`, per §3).
- Shared middleware (mounted once, in `cmd/server/main.go`):
  `middleware.RequestID`, a recoverer that logs panics via zap and converts
  them to `apperr.NewInternalError`, and the centralized error-to-JSON mapper
  from §5.
- One route-registration file per module (`presentation/routes.go`); the
  composition root in `cmd/server/main.go` mounts each module's routes onto
  the shared chi router.

## 9. File & Naming Conventions

- All lowercase filenames. No camelCase/PascalCase. `_` is reserved for Go's
  special suffixes (`_test.go`, `_linux.go`), not general word separation.
- Single-file packages: filename = package name (`apperr.go`, `logger.go`,
  `db.go`).
- Multi-file packages (e.g. `domain`): filename = content, matching the
  equivalent TS filename in `elc-tem` (`types.go`, `repository.go`,
  `utils.go`) for easy cross-reference during migration.
- All identifiers, messages, log lines, and this document: English. (The
  Next.js codebase keeps Vietnamese for user-facing strings — that stays on
  the Next.js/presentation side.)

## 10. Testing

- Skip tests for pure data structs/interfaces — nothing to assert.
- Required for: entity constructors/methods, use cases, and (later) HTTP
  handlers via `httptest`.
- Use the existing Next.js tests as a spec reference when writing the Go
  equivalent: `modules/<name>/__tests__/{domain,application}/*.test.ts` in
  `elc-tem` describes the behavior the Go code must match.
- Test files: `<name>_test.go`, same package (white-box), unless a test is
  explicitly only exercising the public API.

## 11. Manual API Testing

- `.http` files (GoLand/IntelliJ HTTP Client format), one per module:
  `requests/<module>.http`. Manual smoke-testing during development only —
  not a substitute for automated tests.

## 12. Deployment

**VPS constraints (only server available — size every decision around this):**
2 vCPU, 4GB RAM, 80GB disk, **no backup add-on purchased**. Next.js (PM2),
Nginx, and — eventually — self-hosted Postgres + the Go service all compete
for this same 4GB RAM. Do not assume headroom that isn't there; see §17
before Postgres ever runs on this box.

- Next.js (`elc-tem`): unchanged. Standalone build, rsync, PM2 on the VPS.
- Go (`elc-go`): its own Docker image, its own container, its own port
  (e.g. `:8080`) on the same VPS. **Never** merged into one container with
  Next.js — that breaks the one-process-per-container model and couples the
  restart/rebuild of two unrelated services.
- If Next.js is later containerized too: `docker-compose.yml` with
  **separate services**, still not merged processes.
- CI: extend `elc-tem/.github/workflows` with a new job/workflow that builds
  the Go Docker image and deploys it to the VPS using the existing SSH
  secrets, independent of the Next.js deploy job.

## 13. Decided Stack

| Concern     | Choice                        | Explicitly not used |
|-------------|--------------------------------|----------------------|
| Router      | chi                            | gin, echo |
| DB driver   | pgx/v5 (raw SQL)                | GORM / any ORM |
| Migrations  | golang-migrate                  | — |
| Logging     | zap (`*zap.Logger`, not Sugared)| — |
| Container   | Docker, one service per container | 2 services in 1 container |

## 14. Final Data Migration & RLS

This applies only at the very end (§1) — the one-time cutover off Supabase
Cloud, after every module including `auth`/Storage is already on Go.

- **RLS policies will NOT carry over as-is, and that's fine.** Supabase's RLS
  policies rely on Supabase-specific roles/functions (`auth.uid()`,
  `auth.role()`, the `anon`/`authenticated`/`service_role` roles) that only
  exist because PostgREST/GoTrue set session claims per request. Go connects
  via `pgx` with its own role, not through PostgREST — so:
  - **Right now, during the migration**, Go's queries already bypass RLS
    entirely in practice (same as Supabase's `service_role` key does).
    Any authorization Supabase used to give you for free via RLS **must be
    re-implemented explicitly in the Go application/domain layer**
    (permission checks in use cases, not assumed from the DB connection).
    Do not treat this as "still protected" just because RLS rows are still
    defined in Supabase.
  - **At final cutover**, RLS policies referencing `auth.*` will fail to
    restore/execute on a stock self-hosted Postgres (no GoTrue schema there).
    Since authorization is fully owned by Go's application layer by then,
    the policies are dead weight — plan to `DROP POLICY`/skip them rather
    than force them to survive the dump.
- **`pg_dump`/`pg_restore` will not "kill" the source Supabase DB** — a dump
  is read-only against the source. The real risk is a *bad restore* (schema
  errors from Supabase-specific extensions/roles), not damage to the
  original. De-risk by:
  1. Restoring into a **separate staging Postgres** first (not the VPS,
     not prod) and diffing row counts per table against Supabase.
  2. Excluding Supabase-internal schemas (`auth`, `storage`, `realtime`,
     `supabase_functions`) from the dump — only the application schema
     (`public`, plus any custom ones) is needed once Go owns auth/storage.
  3. Only after a clean staging restore, do the real cutover with a short
     write-freeze window (see §1).

## 15. Performance Checklist

- **N+1 queries**: never loop single-row queries per item. Use
  `WHERE id = ANY($1)` with a slice param, or a `JOIN`, to fetch related rows
  in one round trip. This applies especially once `catalog` (products,
  brands, categories, related products) is migrated.
- **Connection pool sizing is constrained by the VPS, not just Postgres
  defaults.** On a 2 vCPU / 4GB box that will eventually also run Postgres
  itself, an unbounded or default-sized `pgxpool` can starve the DB or the
  Go process. Explicitly set `MaxConns` on `pgxpool.Config` (start low,
  e.g. 10, and measure — do not leave it at the library default) once the
  pool moves past the bare `pgxpool.New` in `internal/platform/db`.
- Prefer `SELECT` explicit columns over `SELECT *` in repositories once
  tables grow wider (avoids pulling unused columns over the wire).

## 16. Security Checklist

- **Parameterized queries only, always.** Every repository must pass values
  as `pgx` args (`$1`, `$2`, ...), never string-concatenate user input into
  SQL — this is already the pattern in `PostgresContactRepository`, keep it
  that way for every future repository. No exceptions for "just this one
  dynamic filter."
- **Authorization moves from RLS to application code (see §14).** Every use
  case that touches user-owned or permission-gated data must check the
  caller's permission explicitly — do not rely on the DB connection's role
  to enforce this once Go owns the query.
- Validation errors (`apperr.NewValidationError`) must never leak internal
  details (raw DB error text, stack traces) to the client — only
  `apperr.NewInternalError` wraps the real error, and that wrapped `Err` is
  logged via zap, never serialized in the HTTP response body.
- When the `auth` module is migrated: secure cookie flags
  (`HttpOnly`, `Secure`, `SameSite`), short-lived access tokens + refresh
  rotation, and rate-limiting the login/reset-password endpoints — carry
  this list forward when that module's turn comes.

## 17. Backup Strategy (blocking prerequisite before self-hosting Postgres)

The VPS has **no backup add-on**. Supabase Cloud, whatever its cost, has been
providing some baseline durability for free until now. Moving Postgres onto
this VPS with zero backup would make a single VPS incident (disk failure,
bad `docker compose down -v`, botched migration) unrecoverable — strictly
worse than staying on Supabase from a data-safety standpoint.

**Before Postgres ever runs in Docker on this VPS for real data (not just
local dev), set up, at minimum:**
- A scheduled (daily) `pg_dump`, compressed, shipped **off the VPS** — a
  cheap object storage bucket (Cloudflare R2/Backblaze B2/S3) or even `scp`
  to a different machine. Anything off-box beats what exists today (nothing).
- A tested restore — an untested backup is not a backup. Restore the dump
  into a scratch Postgres at least once before relying on it.
- If budget allows later: WAL archiving (`pgBackRest`/`WAL-G`) for
  point-in-time recovery instead of only daily snapshots — not required to
  start, but worth revisiting once the final cutover date approaches.
