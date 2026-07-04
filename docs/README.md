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
| catalog       | Phase 1 done (Go built & verified) — `elc-tem` cutover is Phase 2, not started | [catalog.md](catalog.md) |
| others        | Not started | — |
