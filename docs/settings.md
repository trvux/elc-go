# Module: settings

**Status**: Migrated to Go. Admin settings dashboard and public layout queries in `elc-tem` go through Go now.
**Purpose**: Site-wide key-value configuration storage (e.g. `company_name`, `company_short_desc`, logo, footer settings, etc.).

## What it replaced

Before migration, this lived in `elc-tem` (Next.js) as `modules/settings/{domain,application,infrastructure,presentation}`, querying Supabase directly via `supabase-js`. The frontend `application/getSiteSettings.ts`, `application/updateSettings.ts` and `infrastructure/` folder have been deleted. `modules/settings/domain` and `modules/settings/presentation/actions.ts` are all that remain.

## Data model

Table `site_settings` (already existed in Supabase before migration — the Go migration in `internal/settings/migrations/000001_baseline_settings.up.sql` is an `IF NOT EXISTS` baseline, not a fresh creation):

| Column  | Type         | Notes |
|---------|--------------|-------|
| `key`   | varchar(255) | PK |
| `value` | text         | nullable |

## HTTP API

Mounted at `/settings`.

| Method | Path         | Body                     | Response |
|--------|--------------|--------------------------|----------|
| GET    | `/settings`  | —                        | `[]SiteSettingDTO` |
| PUT    | `/settings`  | `[]SiteSettingDTO` (bulk) | `204` |

`SiteSettingDTO` (JSON):
```json
{
  "key": "company_name",
  "value": "Cong ty TNHH Dien May ABC"
}
```

## How `elc-tem` calls this

`modules/settings/presentation/actions.ts` has been rewritten:
- `getSiteSettingsAction` performs an HTTP `GET` to retrieve all settings.
- `updateSettingsAction` performs an HTTP `PUT` to bulk upsert settings keys.
- Integrates `isPrerenderError` to cleanly handle Next.js static prerender abort signals.

The following consumers were refactored:
- **Admin Settings Page** (`app/(admin)/admin/(dashboard)/settings/page.tsx`): Updated to fetch and save data using the new Server Actions instead of direct browser-side Supabase client invocations.
- **Public Layout Cache** (`modules/settings/application/getPublicLayoutData.ts`): Updated to resolve site settings from `getSiteSettingsAction()` rather than issuing raw Supabase client queries.

## Testing

- `go test ./internal/settings/...` — domain + application use case unit tests.
- `go test -tags=integration ./internal/settings/infrastructure/...` — integration tests verifying bulk transaction insertion/upserting and retrieval.
