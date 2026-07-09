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
| product       | v2 core slice done (renamed from `catalog`; variants/options/bundle-components/product-lines built, migrated, backfilled for all 197 existing rows on the self-hosted dev DB) — flat single-SKU columns kept unchanged alongside it, `elc-tem` cutover NOT done. Pricing-by-customer-group/promotions/generalized attributes/highlights/services/relations are separate future passes, see [product-v2-design.md](product-v2-design.md) | [product.md](product.md) |
| project       | Migrated (fully clean) | [project.md](project.md) |
| news          | Migrated (fully clean) | [news.md](news.md) |
| branch        | Migrated (fully clean) — missing from this table until now, pre-existing gap not introduced by this migration | [branch.md](branch.md) |
| page          | Migrated (fully clean) — missing from this table until now, pre-existing gap not introduced by this migration | [page.md](page.md) |
| settings      | Migrated (fully clean) — missing from this table until now, pre-existing gap not introduced by this migration | [settings.md](settings.md) |
| project-type  | Migrated (fully clean) | [project-type.md](project-type.md) |
| system-page   | Migrated (fully clean) | [system-page.md](system-page.md) |
| auth          | Not started — intentionally deferred (Supabase Cloud stays authoritative for auth/Storage until every other module is migrated) | — |
