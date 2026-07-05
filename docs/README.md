# Module Docs

One file per business module migrated from `elc-tem` (Next.js) to this
service. Each file is written for hand-off: someone with no prior context on
this migration should be able to read it and understand what the module
does, how it's built, and how it's wired into the frontend — without having
to read the source first.

For conventions that apply to *every* module (layering rules, error
handling, naming, deploy), see [`../ARCHITECTURE.md`](../ARCHITECTURE.md).
This folder is the "what", `ARCHITECTURE.md` is the "how".

## Status

| Module        | Status      | Doc |
|---------------|-------------|-----|
| contact       | Migrated (fully clean) | [contact.md](contact.md) |
| service-group | Migrated (fully clean) | [service-group.md](service-group.md) |
| service       | Migrated (fully clean) | [service.md](service.md) |
| brand         | Migrated (fully clean) | [brand.md](brand.md) |
| group         | Migrated (fully clean) | [group.md](group.md) |
| category      | Migrated (fully clean) — doc file missing, pre-existing gap not introduced by this migration | — |
| catalog       | Migrated (fully clean) — status corrected here; `elc-tem` cutover (previously listed as "Phase 2, not started") had actually already happened (verified: `application`/`infrastructure` are gone from `modules/catalog` in `elc-tem`) | [catalog.md](catalog.md) |
| project       | Migrated (fully clean) | [project.md](project.md) |
| news          | Migrated (fully clean) | [news.md](news.md) |
| others        | Not started | — |
