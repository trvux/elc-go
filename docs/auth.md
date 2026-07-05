# Module: auth

**Status**: Backend done in Go. `elc-tem`'s `/admin` login still runs on Supabase Auth — cutover (rewriting `modules/auth` in Next.js to call this API, building accept-invite/forgot-password UI) is not done yet.
**Purpose**: Admin-panel authentication and account management. Deliberately **not** a public registration system — there is no `/register` endpoint. New admin accounts only ever come from an existing admin's invite.

## Design decision: why invite-only, not public registration

The admin panel is reachable at `dienmayelc.com.vn/admin` — a public URL. If registration were public, anyone finding that URL could create an account and log into the admin dashboard. Email OTP verification alone does not fix this: it proves the registrant owns an email address, not that they should have access at all (authentication vs. authorization).

Fix, mirroring how Supabase Auth (`GOTRUE_DISABLE_SIGNUP`) and most admin systems work: there is no public account-creation endpoint. The only way a row is ever inserted into `users` is `POST /auth/accept-invite`, gated by a single-use, short-lived, high-entropy token that an already-authenticated admin generated via `POST /admin/invites`. A stranger hitting `/admin/accept-invite` without a valid token gets nothing.

## Data model

Fresh tables — nothing here existed in Supabase before this module (unlike most other modules). Supabase's built-in `auth.users` is left in place, untouched, still authoritative until `elc-tem` cuts over.

**`users`**

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `username` | text | unique |
| `email` | text | unique |
| `password_hash` | text | bcrypt, cost 10 — same cost Supabase Auth used, so the pre-existing bcrypt hashes from `auth.users` could be migrated as-is without forcing a reset (not yet done — only the bootstrap seed exists today) |
| `name` / `phone` | text, nullable | self-reported at accept-invite, display/contact only — never used for auth |
| `role` | enum `auth_role` | `user` \| `admin` \| `super_admin` (see RBAC below) |
| `status` | enum `auth_user_status` | `active` \| `disabled` |
| `last_login_at` | timestamptz | nullable |

**`verification_tokens`** — backs both the invite flow and forgot-password. Only `token_hash` (SHA-256) is stored, never the raw token.

| Column | Type | Notes |
|---|---|---|
| `purpose` | enum `auth_token_purpose` | `invite` \| `password_reset` |
| `token_hash` | text | unique |
| `email` | text | |
| `role` | enum `auth_role`, nullable | only set for `invite` |
| `invited_by` | uuid FK → users, nullable | only set for `invite` |
| `user_id` | uuid FK → users, nullable | only set for `password_reset` |
| `expires_at` / `consumed_at` | timestamptz | |

**`sessions`** — refresh tokens. Access tokens (JWT) are stateless and never stored; only refresh tokens need a row so logout/password-reset can actually revoke them.

| Column | Type | Notes |
|---|---|---|
| `user_id` | uuid FK → users | |
| `token_hash` | text | unique |
| `user_agent` / `ip_address` | text, nullable | |
| `expires_at` / `revoked_at` | timestamptz | |

## RBAC

Three roles, ranked `user` < `admin` < `super_admin` (`domain.roleRank`). `User.CanInvite(targetRole)` enforces that nobody can grant a role higher than their own, and `user` can't invite at all.

Beyond roles, `internal/auth/domain/permission.go` adds a fine-grained `Permission` layer so call sites check "can this role do X" instead of comparing role strings directly:

| Permission | admin | user | super_admin |
|---|---|---|---|
| `users:manage` (invite/disable/change role) | ✓ | | ✓ (implicit) |
| `content:write` (catalog/news/pages/...) | ✓ | ✓ | ✓ |
| `content:delete` | ✓ | | ✓ |
| `settings:manage` | ✓ | | ✓ |

`super_admin` isn't listed in the map (`rolePermissions`) — `Role.HasPermission` short-circuits to `true` for it, so a new permission constant never needs a matching super_admin entry added by hand.

This is a **static Go map, not a DB-backed policy table** (deliberately — see the doc comment on `Permission`): the role set is small and fixed, nobody has asked for admins to redefine permissions at runtime, and every other part of this codebase avoids infra a real requirement hasn't shown up for yet (no ORM, hand-written SQL, etc.). Checked via `httpserver.RequirePermission(func(role string) bool {...})` middleware — see `/admin/invites`'s route registration for the pattern other modules should copy once they're ready to gate their own write routes.

## HTTP API

| Method | Path | Auth | Body | Notes |
|---|---|---|---|---|
| POST | `/auth/login` | — | `{identifier, password}` | `identifier` = username or email |
| POST | `/auth/logout` | refresh cookie | — | idempotent |
| POST | `/auth/refresh` | refresh cookie | — | rotates the refresh token |
| POST | `/auth/forgot-password` | — | `{email}` | always returns the same generic message |
| POST | `/auth/reset-password` | — | `{token, password}` | revokes all sessions on success |
| POST | `/auth/accept-invite` | — | `{token, username, password, name?, phone?}` | the only way a user row is created |
| GET | `/auth/me` | bearer | — | |
| POST | `/admin/invites` | bearer, permission `users:manage` | `{email, role}` | target role capped by the caller's own rank — see RBAC |
| GET | `/admin/users` | bearer, permission `users:manage` | — | backs the admin-panel user management screen |
| PATCH | `/admin/users/{id}` | bearer, permission `users:manage` | `{role?, status?}` | can't act on your own account; can't touch/grant a rank above your own (`User.CanManageUser`/`CanInvite`); disabling revokes all of that user's sessions immediately |

`elc-tem`'s `/admin/users` page (module `admin-users`) is the UI for the last two — invite dialog, per-row role `Select`, lock/unlock button. Sidebar only shows the "Quản lý người dùng" link for admin/super_admin (`shared/components/layout/admin/sidebar.tsx`); the page itself also redirects a `user`-role visitor server-side, so hiding the link is UX only, not the real gate.

Login/refresh set `refresh_token` as an httpOnly, `SameSite=Lax` cookie (`Secure` when `ENV=production`), scoped to `/auth`. Access token (15 min TTL) comes back in the JSON body — client holds it in memory, not localStorage.

Password policy (`domain.ValidatePassword`): more than 8 characters, at least one uppercase, one lowercase, one digit, one special character.

Rate limiting: in-memory fixed-window (`internal/platform/ratelimit`), scoped per-process since this runs as a single container — login (10/15min by identifier+IP), forgot-password (5/15min by email+IP, silent), reset-password/accept-invite (10/15min by IP).

## Bootstrapping the first account

There is no public registration and invites require an existing admin — so something has to create the very first account. That's `cmd/seed-admin`, **not** a migration: a real person's name/email/phone/password must never be written into a committed SQL file, since migrations live in git forever, readable by anyone with repo access, long after that person leaves and the DB row is gone or rotated.

```
ADMIN_USERNAME=... ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... ADMIN_PHONE=... ADMIN_ROLE=super_admin \
  make seed-admin
```

Values only ever exist in the shell/CI secret store at run time. If the email already exists, this only resets its password — same command doubles as the break-glass path if the bootstrap account is ever locked out. `internal/auth/migrations/000002_seed_bootstrap_admin.{up,down}.sql` are deliberately empty placeholders left in the sequence (renaming/renumbering migrate files after the fact is worse than an empty file with a comment explaining why).

## What's NOT done yet

- The old Supabase `auth.users` accounts are being retired, not migrated — this system only has the one seeded `super_admin` account (`tranvux` / `huynhdong1115@gmail.com`) by design; nobody else gets an account except via a new invite. Deletion is deferred until elc-tem's cutover is confirmed working, so the old Supabase-based `/admin` login isn't broken mid-flight.

## elc-tem cutover (done)

`elc-tem` is a BFF: its Server Actions call elc-go's `/auth/*` endpoints server-to-server and store the resulting tokens as its own httpOnly cookies (`go_access_token`, `go_refresh_token`) scoped to its own domain — the browser never talks to elc-go directly, and elc-go's own `refresh_token` cookie never needs to leave the server-to-server call.

- `modules/auth/infrastructure/authRepo.ts` — `GoAuthRepository`, replaces the old `SupabaseAuthRepository`.
- `shared/lib/auth/session.ts` — replaces `shared/lib/supabase/session.ts` (deleted) as the middleware (`shared/proxy.ts`) session check: decodes (doesn't verify) the access token's `exp` claim to decide if a refresh is worth attempting before falling back to redirecting to `/admin/login`.
- New pages: `app/(admin)/admin/{accept-invite,forgot-password,reset-password}/page.tsx`, all public (excluded from the middleware's auth guard alongside `/admin/login`).
- `app/(admin)/admin/(dashboard)/layout.tsx` still does the same `getCurrentUser(authRepo)` + `redirect` guard as before — just backed by Go's `/auth/me` now instead of Supabase.

## RBAC wiring on other modules (done)

Every other module's write routes (`contact`, `brand`, `catalog`, `project`, `news`, `branch`, `page`, `settings`, `project-type`, `system-page`, `service`, `service-group`, `group`, `category`) now go through `httpserver.RequireAuth` + `httpserver.RequirePermission` — reads stay public. Create/update/restore/reorder routes require `authdomain.CanWriteContent`, delete routes require `authdomain.CanDeleteContent`, `settings`'s single `PUT /` requires `authdomain.CanManageSettings`. `cmd/server/main.go` passes the same `tokenIssuer` (already implementing `httpserver.TokenVerifier`) into every module's `RegisterRoutes` call.

## Testing

- `go test ./internal/auth/...` — domain + application unit tests, fake repos/services, no network.
- `go test -tags=integration ./internal/auth/infrastructure/...` — real DB round-trip for all three repositories.
