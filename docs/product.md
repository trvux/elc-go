# Module: product (renamed from `catalog`)

**Status**: v2 (variants/options/bundles/product lines) built and live on
the self-hosted dev DB as of 2026-07-08 — see the "v2: variants, options,
product lines" section below for the as-built shape and
[product-v2-design.md](product-v2-design.md) for the original design/
rationale. The section further down ("Data model" onward) still describes
the original flat single-SKU shape, which is **unchanged and still what
`elc-tem` reads** — nothing below was removed or altered by v2, it was
purely additive. `elc-tem` cutover to call these endpoints instead of
Supabase directly has NOT happened yet — see "How `elc-tem` calls this"
below.

## v2: variants, options, product lines

Adds the `Product → Option/Variant → VariantComponent` model from
docs/product-v2-design.md on top of the untouched flat schema. New tables:
`product_lines`, `product_options`, `product_option_values`,
`product_variants`, `product_variant_option_values`,
`product_variant_components` (migration
`internal/product/migrations/000005_add_variants.up.sql`). New columns on
`products`: `product_line_id`, `short_description`, `warranty_months`,
`warranty_terms`, plus four **write-time denormalized read-cache columns**
— `default_variant_id`, `display_price`, `display_stock_status`,
`price_min`, `price_max` — recomputed by
`infrastructure.RecomputeDisplayCache` inside the same transaction as any
variant write (same technique as `normalized_specs`: compute once at write
time, read via a plain column, never join/aggregate at list-read time).

**Why this exists**: the flat model couldn't represent a real appliance
retailer's catalog — one Daikin AC model comes in 5 brand tiers
(FTF/FTKB/FTKF/FTKY/FTKZ → `product_lines`), most units are an indoor+
outdoor split system with two separate manufacturer part numbers sold as
one purchasable set (→ `product_variant_components`, `is_standalone=false`
on the parts), and `sku` was being misused as if it were the manufacturer's
own code (a live row stored `"FTKB25ZVMV / RKB25ZVMV"` — two real MPNs
concatenated — in what should have been an internal-only warehouse field).
v2 moves `mpn` (required, the field customers actually search for) and
`sku`/`gtin` down to the variant level.

**Performance**: `GET /products` (list/grid) is **unchanged, still zero
joins to variant tables** — it reads `display_price`/`price_min`/
`price_max`/`display_stock_status` straight off `products`. Only
`GET /products/{id}` / `GET /products/slug/{slug}` (single-product reads)
load the full option/variant/component tree, via separate batched queries
keyed by product id (`fetchOptionsForProduct`/`fetchVariantsForProduct` in
`infrastructure/variant_repository.go`) — never a single big JOIN, for the
same row-multiplication reason `fetchTagsForProducts` is already separate
from the category/brand JOIN.

**Update replaces the variant tree wholesale** (delete options+variants+
components, reinsert), same convention `project-type` already uses for
`project_type_category` — not an ID-matched upsert. This means variant IDs
are not stable across an update that touches the variant tree. Deliberate
simplification for this pass: nothing yet references `product_variants.id`
across requests (no order/pricing module exists yet), so there's no
consumer that would notice. Revisit (ID-matched upsert) once an order or
pricing module is built that holds a `variant_id` reference across
requests.

**The repository itself does not enforce "exactly one default variant"**
— that rule (`resolveDefaultVariant`) lives in the application layer
(`application/create_product.go`/`update_product.go`): a single-variant
product is auto-defaulted, multiple variants with none marked default
default the first one, multiple explicit defaults is a validation error.
The DB only enforces the invariant defensively via a partial unique index
(`product_variants_one_default_active`).

**`ProductLine` is its own repository/handler** (`PostgresProductLineRepository`,
`ProductLineHandler`, mounted at `/product-lines`) but lives inside this
same Go module rather than a separate `internal/product-line` — it
currently has exactly one consumer (`Product`), so a separate bounded
context would be premature.

### Backfill (2026-07-08, run once against the self-hosted dev DB)

`cmd/backfill-product-variants` gave all 197 pre-existing `products` rows
exactly one `product_variants` row (`mpn` from the old `mpn` column, or the
old `sku` string as a fallback; `sku`/prices/stock-status copied as-is,
`stock_status` mapped `out_of_stock`/`pre_order` → `order_from_supplier`,
`discontinued` → `discontinued`, everything else → `in_stock`). Ran inside
one transaction with a row-count verification before commit. **Found real
duplicate legacy MPNs** (17 of 197) — e.g. two different products both had
`mpn = "FBFC100DVM9"` already set — which collided with the new
unique-among-active constraint on `product_variants.mpn`. Rather than block
the whole backfill on a legacy data-quality issue, the tool disambiguates:
first claim wins, every subsequent collision gets a `-DUPn` suffix and is
printed to stdout. **These 17 products need manual MPN correction via the
normal admin edit UI** — the tool only prevented data loss, it didn't fix
the underlying duplicate. Re-running the tool is safe (only touches
products with zero existing variants, so already-backfilled rows are
skipped).

### API additions

New nested `options`/`variants` in `POST /products` and `PUT /products/{id}`
bodies (variant option selections reference by `{option_name, value}`, not
ID — those don't exist yet for a tree created in the same request; bundle
components reference by index into the same `variants` array, not ID).
`GET /products/{id}`/`GET /products/slug/{slug}` responses now include
`options[]`/`variants[]` (variant response omits `cost_price` — internal
margin data, never exposed). New: `GET/POST /product-lines`,
`GET/PUT/POST/DELETE /product-lines/{id}`, `POST /product-lines/{id}/restore`.

### Testing

`go test ./internal/product/...` (domain: variant/line validation,
`DisplayPrice` fallback; application: `resolveDefaultVariant`'s four
scenarios). `go test -tags=integration ./internal/product/infrastructure/...`
— real round trip creating a product with 2 option values × a split-system
bundle (2 components + 2 sellable color variants), verifying
`product_variant_option_values`/`product_variant_components` land
correctly and `display_price`/`price_min`/`price_max` recompute correctly
on both create and a full variant-tree replace via update; separate
`ProductLine` CRUD round trip. Both self-clean via `defer`. Run against the
self-hosted dev Postgres — note this container is on OrbStack, where
`localhost:<published-port>` can silently fail to forward (see
`docs/README.md`-linked migration conventions memory / `ARCHITECTURE.md`
for the workaround: run the Go test itself inside a container on the same
Compose network, targeting `postgres:5432` by service name, not the
published host port).
**Purpose**: Products (máy lạnh, máy lọc không khí, etc.) shown on the public
catalog pages and managed in the admin dashboard. The richest module migrated
so far — full-text search, faceted filtering, and HVAC spec normalization,
where brand/service only needed straightforward CRUD + one join.

## What it replaced

Before this migration, product reads/writes lived entirely in `elc-tem`
(Next.js) as `modules/catalog/{domain,application,infrastructure,
presentation}`, querying Supabase directly via `supabase-js`. The most
significant piece being replaced is `modules/catalog/application/
searchProducts.ts` — a ~500-line function that, on every single list/search
request, loaded the *entire* product table into memory, ran a client-side
Fuse.js fuzzy search, derived HVAC facet values via regex/string-matching
scattered inline, filtered, sorted, and paginated, all in the Node process.
That function is now replaced by:

- `internal/catalog/domain/spec_normalizer.go` — the HVAC facet-derivation
  logic (ported verbatim, see below), but run ONCE per product at write time
  instead of on every read.
- Postgres itself doing full-text search (`tsvector`/`unaccent`/`pg_trgm`)
  and faceted aggregation (`GROUP BY`/`unnest`) — see "Search & facets"
  below.

`modules/catalog/domain/price.ts`'s `normalizeProductPrice` is ported to
`internal/catalog/domain/price.go` — same logic, moved from read-time
(applied to every product in `searchProducts.ts`'s final `.map()`) to
write-time (applied once in `application/create_product.go`/
`update_product.go`). Phase 2 (TS cutover) is a separate step; this Go module
is not wired into `elc-tem` yet.

## Data model

Table `products` (already existed in Supabase — the Go migration in
`internal/catalog/migrations/` is mostly an `IF NOT EXISTS` no-op against the
real DB; the two genuinely new pieces are `normalized_specs` and
`search_vector`, added via `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`):

| Column              | Type                | Notes |
|---------------------|---------------------|-------|
| `id`                | uuid                | PK, `gen_random_uuid()` default |
| `category_id`       | uuid                | not null, FK → `categories(id)` ON DELETE RESTRICT |
| `brand_id`          | uuid                | not null, FK → `brands(id)` ON DELETE SET NULL — same contradictory NOT NULL + SET NULL combination as documented in `docs/brand.md`; practically unreachable |
| `name`              | text                | not null |
| `sku`               | text                | not null, **globally UNIQUE** (`products_sku_key`) — even across soft-deleted rows |
| `slug`              | text                | not null, unique only among **non-deleted** rows (partial index `products_slug_unique_active`) |
| `description`       | jsonb               | not null, default `'{}'` — Tiptap rich text, passed through opaquely as `json.RawMessage` |
| `specs`             | jsonb               | not null, default `'{}'` — semantically an array of `SpecItem`, see below |
| `images`, `labels`  | text[]              | default `'{}'` |
| `original_price`    | numeric(15,0)       | default `0` |
| `sale_price`        | numeric(15,0)       | nullable, no default |
| `discount_percent`  | numeric(5,2)        | default `0` |
| `is_featured`, `is_published` | boolean   | default `false`/`true` |
| `order_index`       | integer             | default `0` |
| `stock_status`      | text                | default `'in_stock'` |
| `condition`         | `product_condition` (Postgres enum: `new`/`used`) | not null, default `'new'` |
| `meta_title`, `meta_description`, `mpn`, `gtin` | text | nullable |
| `created_at`/`updated_at`/`deleted_at` | timestamptz | soft delete; `updated_at` is set automatically by the pre-existing `update_products_modtime` BEFORE UPDATE trigger — Go's `Update` deliberately does NOT set it (unlike brand's `Update`, written before this trigger's presence on this table was confirmed) |
| `normalized_specs` **(new)** | text[]      | default `'{}'` — write-time-derived facet strings, see below |
| `search_vector` **(new)**    | tsvector (generated, stored) | `to_tsvector('simple', immutable_unaccent(name \|\| ' ' \|\| sku))` — see "Search & facets" |

`specs` shape (`internal/catalog/domain/types.go`, ported verbatim from
`modules/catalog/domain/types.ts`):
```go
type SpecSubItem struct { Label string; Value string; Unit *string }
type SpecItem struct { Label string; Value *string; Unit *string; Items []SpecSubItem }
```
A `SpecItem` either carries a single `Value` or a nested `Items` list, never
both in real data (confirmed against live rows, e.g. `{"label": "Công suất
làm lạnh", "items": [{"value": "2 Hp"}, {"value": "18.000 Btu/h"}]}`).

**Trigger**: `trg_product_slug_registry` keeps the shared `slug_registry`
table in sync on INSERT/UPDATE/DELETE — same shared platform infrastructure
as `trg_brand_slug_registry`/`trigger_sync_service_slug`. **Fully automatic —
Go code never touches `slug_registry` directly**, same as every other
migrated module.

**Joined refs**: `internal/catalog/domain/types.go` defines `CategoryRef`
(`id, name, slug, meta_title, meta_description`) and `BrandRef` (`id, name,
slug, logo_url, meta_title, meta_description, is_featured, order_index`) —
lightweight, read-only structs populated by `LEFT JOIN categories`/`LEFT JOIN
brands` directly in this module's own SQL (`internal/catalog/infrastructure`),
same reasoning as service's `GroupRef`/`CategoryRef`: avoids a cross-module Go
dependency on `internal/brand`/`internal/category` for the handful of fields
any UI actually reads from the join.

## Design decision: write-time spec normalization (replaces read-time Fuse.js + regex)

The old TS `searchProducts.ts` derived HVAC facet values (capacity, cooling
direction, gas type, inverter, filtration, filter efficiency) via regex/
substring matching **on every single search request**, against the entire
in-memory product list. `internal/catalog/domain/spec_normalizer.go` ports
that exact logic (`SPEC_WHITELIST`, `REVERSE_SPEC_MAPPING`,
`getCoolingDirection`, `getCapacityFromText`, `getGasType`,
`normalizeSpecValue`) but runs it **once, at write time** (`create_product.go`/
`update_product.go`, via `domain.NormalizeProductSpecs(name, specs)`), storing
the flat result (`"UILabel::Value"` strings, e.g. `"Công suất::1.5 HP"`) in
the new `normalized_specs text[]` column. Read/list/search queries then just
do a plain array-overlap (`&&`) match against that column — no regex, no
Fuse.js, no per-request recomputation. This is an intentional simplification
of the old Fuse.js "specs" search key too: that key was already low-weight
(0.1 of the fuzzy match score) in the old config, and `normalized_specs` is
explicitly an exact-facet filtering mechanism, not part of full-text search
(`search_vector` only covers `name`+`sku` — see below).

**A pre-existing bug in the old TS `normalizeSpecValue` was ported verbatim,
not fixed**: inside the "Lọc không khí" branch, three of six substring checks
search for a *multi-word* phrase (`"bụi mịn"`, `"khử mùi"`, `"tiêu chuẩn"`,
`"lọc thô"`) against a string that already had **all whitespace stripped**
one line earlier (`lower = val.toLowerCase().replace(/\s/g, "")`). A
whitespace-free string can never contain a substring that itself has a space
in it, so those three conditions are dead code — only the single-word checks
(`hepa`, `pm2.5`, `ion`, `nanoe`, `mesh`) ever actually match, in both the
original and this port. Documented in `spec_normalizer.go` and covered by
`TestNormalizeSpecValue_AirFilterSingleWordChecksOnly`. Not fixed here —
faithfully porting existing behavior was the goal, not silently changing
product-facing filter results as a side effect of an unrelated migration.

**Backfill was performed for the 196 pre-existing rows.** Since normalization
only runs at write time, every row that predated this migration had
`normalized_specs = '{}'` until it went through `Update` at least once —
confirmed live immediately after the migration (`SELECT count(*) FROM
products WHERE normalized_specs != '{}'` returned `0`). A one-off script
called `PUT /products/{id}` with just `{"name": <unchanged existing name>}`
for all 196 products — a genuine no-op on every other field, but enough to
trigger `update_product.go`'s "name changed → recompute" branch. Re-verified
after: 164/196 rows now have non-empty `normalized_specs` (the remaining 32
simply have no HVAC-recognizable spec/name pattern — e.g. accessories or
non-AC categories — expected, not a bug), and `GET /products` facets now
return real populated values (capacity, inverter, gas type, filter class,
cooling direction) instead of an empty list.

## Design decision: search & facets (why Postgres FTS, not a dedicated search engine)

`search_vector` (`GENERATED ALWAYS AS ... STORED`) covers `name` + `sku` only
— using the `'simple'` text search config (no stemming; Vietnamese isn't a
supported PostgreSQL stemming language) combined with `unaccent()` so
"dieu hoa" and "điều hòa" both match the same rows.

**Full-text and the fuzzy/typo-tolerance fallback are two mutually exclusive
strategies, never OR'd together in the same query** (`resolveSearchMode`
picks one before the real queries run): try plain full-text first; only if
it finds *zero* rows (within every other currently-active filter) does the
per-token `pg_trgm` `word_similarity()` fallback kick in, with **AND across
tokens**, not a single whole-query-vs-whole-name comparison. Both of these
were tuned after finding real, empirically-verified false positives during
manual QA against the live catalog with a plain `similarity()` OR-blend
(the first attempt):
- Whole-string `similarity()` diluted a genuine typo match to noise: the
  query `"inveter"` (missing the middle `r` of `"inverter"`) scored only
  ~0.19 against the full name `"Máy lạnh LG Inverter 1Hp ( 1 pha )"` because
  the short query gets penalized against a long multi-word name — below any
  workable threshold. `word_similarity()` finds the best-matching
  word-bounded substring instead, scoring the same pair 0.545, well above
  threshold.
- Blending full-text OR whole-string-fuzzy let unrelated products leak in:
  searching `"am tran"` (unaccented "âm trần", ceiling-cassette AC) also
  matched `"Máy lạnh giấu trần..."` ("giấu trần" = concealed duct-in-ceiling
  — a different, unrelated product line) at word_similarity 0.625 — *higher*
  than the legitimate inverter typo match above — because "trần" alone
  overlaps strongly even though "âm" and "giấu" share nothing. No single
  scalar threshold separates these two cases. Per-token AND does:
  `"giấu trần"` scores 0 on the token `"am"` (no word in it resembles "am"
  at all) and is excluded; `"âm trần"` scores 1.0 on both tokens and is
  included. Same semantics the old Fuse.js search used
  (`queryTokens.every(set => set.has(id))`), reimplemented in SQL.
  A related case reproduced the same way: `"treo tuong"` (wall-mounted)
  spuriously matched `"âm trần đa hướng thổi"` products under the naive
  whole-string/OR design (`"tuong"` vs `"hướng"` scored 0.667 — Vietnamese's
  small syllable inventory means short unaccented tokens collide easily);
  the two-phase resolve-then-query design fixes this the same way, since
  `"treo tuong"` already finds correct full-text matches and the fuzzy path
  is never reached for it at all.

When full-text mode is used, results rank by `ts_rank(...)` alone; when the
fuzzy fallback is used, results rank by the average of each token's
`word_similarity(...)` (never both in the same query — see above). Either
way, the requested/default sort applies as a tiebreaker after rank.

At 196 rows today, this is comfortably fast with the GIN indexes in place;
Postgres full-text search scales to 100k+ rows with the same indexing
approach. This was a deliberate, scale-appropriate choice for this catalog's
actual size — not a stopgap or a compromise pending "a real search engine"
later.

**`unaccent()` is STABLE, not IMMUTABLE**, and Postgres rejects STABLE
functions inside a `GENERATED ALWAYS AS` expression ("generation expression
is not immutable" — hit this for real running the migration against the live
DB before adding the fix). Fixed with the standard workaround: a thin SQL
wrapper `immutable_unaccent(text)` explicitly marked `IMMUTABLE` (safe in
practice — the `unaccent` dictionary this project uses is never altered at
runtime). Used both in the generated column and in the equivalent
query-time `unaccent(...)` calls in `postgres_repository.go`, for
consistency between what's indexed and what's queried.

**Facets are computed with 3 additional aggregate queries, each ignoring the
one filter dimension it facets on** (standard technique — e.g. the brand
facet should reflect every OTHER active filter but not the currently-selected
brand, so a user can still see and switch to sibling brands):
- Brand facet: `SELECT DISTINCT br.id, br.name, br.slug ... WHERE <all filters except brand>`
- Spec facet: `SELECT unnest(normalized_specs) ... WHERE <all filters except specs> GROUP BY facet`, split `"Label::Value"` in Go and grouped by label
- Price facet: `MIN/MAX(COALESCE(sale_price, original_price)) WHERE <all filters except price>`

All three, plus the main list query and the total count, run **concurrently**
via `golang.org/x/sync/errgroup` against the shared `pgxpool` (5 independent
queries, each on its own pooled connection) — turning what would be 5
sequential round trips into 1.

## Business rules / gotchas

1. **`condition` enum validated in Go before it ever reaches Postgres.**
   `validateCondition` rejects anything but `"new"`/`""`/`"used"` with a clean
   `400`, instead of a raw "invalid input value for enum product_condition"
   `500`. Empty string defaults to `"new"` in `domain.NewProduct` (matching
   the column's own default).
2. **`category_id`/`brand_id` validated as required in Go**, beyond what was
   explicitly called out for brand/service (whose equivalent FK columns are
   nullable) — because these two are `NOT NULL` on `products`. A bad partial
   update now fails with a `400` instead of a raw NOT NULL constraint
   violation surfacing as a `500`.
3. **Pricing is reconciled at write time via `NormalizeProductPrice`**,
   ported verbatim from `modules/catalog/domain/price.ts`. `UpdateProduct`
   resolves `OriginalPrice`/`SalePrice`/`DiscountPercent` together — a
   partial update touching only one of the three never leaves the other two
   stale, same invariant `service`'s `UpdatePricing` established (see
   `docs/service.md`), widened here to all three fields since `products`
   persists `sale_price`/`discount_percent` directly (service only persists
   two fields and derives `SalePrice()` fresh on every read instead).
   **Caveat**: pre-existing rows that predate this migration were never
   re-normalized — their stored `sale_price`/`discount_percent` may still
   reflect whatever inconsistency the old read-time `normalizeProductPrice`
   was papering over on every request. They'll self-correct the next time
   each row is updated through this API.
4. **`UpdateSpecs`/`UpdatePricing` each take multiple fields together on
   purpose** — mirrors `service.Service.UpdatePricing`'s "resolve the
   unchanged half to its current value before calling" pattern, extended
   here because `normalizedSpecs` depends on *both* `specs` and `name`:
   renaming a product (e.g. adding "2HP" to the name) with no `specs` in the
   same request still triggers a `normalized_specs` recompute in
   `application/update_product.go`. Covered by
   `TestUpdateProduct_RenamingRecomputesNormalizedSpecs`.
5. **Repository interface returns `*ProductWithRelations` for reads, plain
   `*Product` for writes** — a deliberate deviation from the task's initial
   sketch (which showed `GetByID`/`GetBySlug`/`GetByIDs` returning bare
   `*Product`), made to match the join pattern `service` already established
   (`ServiceWithRelations` for `GetAll`/`GetByID`/`GetBySlug`, plain
   `*Service` for `Create`/`Update`) — necessary here since the HTTP response
   needs nested `category`/`brand` objects without a second round trip.
6. **`normalized_specs` is never exposed in the API response** — it's an
   internal write-time facet-indexing detail (see spec normalization design
   above), not something any UI reads directly, same reasoning brand/service
   never expose derived/computed state on their entities. `specs` in the
   response is always the client's original `[]SpecItem`.

## pgx simple-protocol gotchas

Same root cause as `docs/brand.md`'s FAQ gotcha (the shared pool runs
`pgx.QueryExecModeSimpleProtocol` for the PgBouncer transaction-pooler fix —
see `docs/contact.md`):

- **`specs` (`[]domain.SpecItem`)**: hand-marshaled via `marshalSpecs`/
  `unmarshalSpecs` in `infrastructure/postgres_repository.go`, returning the
  named `json.RawMessage` type specifically (not a plain `[]byte`) — pgx
  encodes `json.RawMessage` as raw JSON text but a bare `[]byte` as a bytea
  literal, which Postgres then rejects casting to `jsonb`. Unlike brand's
  `faq` column (nullable), `products.specs` is `NOT NULL DEFAULT '{}'`
  jsonb but semantically holds a JSON **array** — `marshalSpecs(nil)` returns
  `"[]"`, never `nil`/`"{}"`, to avoid either a NOT NULL violation or storing
  the wrong JSON shape.
- **`description` (`json.RawMessage`)**: same NOT NULL DEFAULT '{}' situation
  — `orEmptyJSON` defaults a nil/empty value to `"{}"` before binding, since
  an explicit `NULL` bind would violate the NOT NULL constraint (the column
  default only applies when a column is omitted from the INSERT list
  entirely, not when NULL is given explicitly).
- **`images`/`labels`/`normalized_specs` (`[]string`)**: pgx encodes/decodes
  these natively to/from Postgres `text[]` with no special handling, same as
  `service.labels`. `orEmptyStrings` still defaults a nil Go slice to an
  empty one before binding, matching each column's own `DEFAULT '{}'` intent
  rather than writing an explicit `NULL`.
- **Array filter params** (`category_ids`, `brand_ids`, `brand_slugs`,
  `normalized_specs &&`): bound with an explicit `::uuid[]`/`::text[]` cast in
  the SQL text (e.g. `p.category_id = ANY($1::uuid[])`) — under simple
  protocol, an untyped array literal compared via `ANY()`/`&&` needs an
  explicit cast to avoid relying on Postgres's implicit-unknown-type
  coercion working the same way it does for a single-value comparison.

## HTTP API

Mounted at `/products`.

| Method | Path                      | Notes |
|--------|---------------------------|-------|
| GET    | `/products`               | query: `category_id`, `category_ids` (comma-separated), `brand_id`, `brand_ids`, `brand_slugs`, `is_featured`, `is_published`, `search`, `min_price`, `max_price`, `sort_by` (`price_asc`\|`price_desc`\|`newest`\|`popularity`\|`discount_desc`), `condition`, `spec_<label>` (repeatable), `limit`, `offset`, `include_deleted`. Response: `{data, total_count, facets}` |
| GET    | `/products/count`         | same filters, `{"count": N}` |
| GET    | `/products/slug/{slug}`   | joined (`category`, `brand`) |
| GET    | `/products/{id}`          | joined |
| GET    | `/products/{id}/adjacent` | `{prev: {name,slug}\|null, next: {name,slug}\|null}` |
| POST   | `/products/by-ids`        | body `{"ids": [...]}`, bulk fetch (e.g. "related products") |
| POST   | `/products`               | create, plain (no join) response |
| PUT    | `/products/{id}`          | partial update, plain response |
| DELETE | `/products/{id}`          | soft delete |
| POST   | `/products/{id}/restore`  | |

Example `GET /products?limit=1` response shape (abbreviated):
```json
{
  "data": [
    {
      "id": "47782a9e-2c46-456c-b6f9-2a9c51f22ba1",
      "category_id": "db74c68a-3e74-4cb8-8ed9-8ab439876df5",
      "brand_id": "c596457f-52bd-4284-8e6d-929e7a978716",
      "name": "Máy lạnh treo tường Daikin 1HP - một chiều Inverter",
      "sku": "FTKB25ZVMV / RKB25ZVMV",
      "slug": "may-lanh-treo-tuong-daikin-1hp-mot-chieu-inverter-ftkb25zvmv",
      "description": { "type": "doc", "content": [] },
      "specs": [{ "label": "Công nghệ Inverter", "value": "Có" }],
      "images": [],
      "labels": [],
      "original_price": 9342593,
      "sale_price": 9342593,
      "discount_percent": 0,
      "is_featured": false,
      "is_published": true,
      "order_index": 0,
      "stock_status": "in_stock",
      "condition": "new",
      "meta_title": null,
      "meta_description": null,
      "mpn": null,
      "gtin": null,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z",
      "deleted_at": null,
      "category": { "id": "...", "name": "Máy lạnh treo tường", "slug": "may-lanh-treo-tuong", "meta_title": null, "meta_description": null },
      "brand": { "id": "...", "name": "Daikin", "slug": "daikin", "logo_url": "...", "meta_title": null, "meta_description": null, "is_featured": true, "order_index": 0 }
    }
  ],
  "total_count": 196,
  "facets": {
    "brands": [{ "id": "...", "name": "Daikin", "slug": "daikin" }],
    "specs": [{ "label": "Công suất", "values": ["1 HP", "1.5 HP"] }],
    "min_price": 0,
    "max_price": 67510520
  }
}
```

## How `elc-tem` calls this

**TBD — Phase 2, not yet migrated.** This task built and verified the Go
service in isolation; `elc-tem` continues to read/write `products` directly
via `supabase-js` (`modules/catalog/infrastructure/SupabaseProductRepository.ts`,
`modules/catalog/application/{getProducts,searchProducts,getAdjacentProducts}.ts`,
etc.) until a separate follow-up cutover wires `presentation/actions.ts` to
call the endpoints above instead — same phased pattern already used for
`brand`/`service`/`service-group`. Do not modify `elc-tem` as part of this
task; that repo was explicitly out of scope here.

## Dead code found during audit (documented, not silently dropped)

- The `normalizeSpecValue` "Lọc không khí" whitespace-stripping bug described
  above (three dead multi-word substring checks) — found while porting, kept
  as-is rather than "fixed" as a drive-by change.
- `modules/catalog/domain/repository.ts`'s `ProductRepository` interface
  historically included methods with no callers anywhere in `elc-tem`
  (not independently re-audited exhaustively here since Phase 2 is out of
  scope for this task) — worth a fresh grep-based audit at Phase 2 cutover
  time, same as brand/service's precedent, rather than assuming this doc's
  Go-side interface is a 1:1 necessity match for every old TS method.

## Not migrated / out of scope for this module (Phase 1)

- Anything in `/Users/tranvux/Documents/elc-tem` — untouched, per explicit
  instruction. Read-only references were consulted (`modules/catalog/
  application/searchProducts.ts`, `domain/{types,price}.ts`,
  `application/getAdjacentProducts.ts`) purely to port their logic faithfully.
- Backfilling `normalized_specs` for the 196 pre-existing rows (see design
  decision above) — schema and write path both support it, but running a
  bulk `UPDATE` against every existing row wasn't part of this task's ask.
- Any admin-only bulk import/export tooling that may exist in `elc-tem` for
  products — not inventoried here, would be part of Phase 2's consumer audit.

## Testing

- `go test ./internal/catalog/...` — domain (`NormalizeProductPrice`,
  `NormalizeProductSpecs` including the HP/inverter/gas/sub-item/unrecognized-
  label/dedup cases and the documented whitespace-stripping quirk) +
  application unit tests (fake in-memory repository, including the
  write-time normalization and "rename recomputes normalized_specs"
  invariants).
- `go test -tags=integration ./internal/catalog/infrastructure/...` — real DB
  roundtrip (create/read-by-id/read-by-slug/read-by-ids/update/soft-delete/
  restore, specs+images jsonb/array round-trip, category/brand join
  resolution), an unaccent search test (`"dieu hoa"` and `"điều hòa"` both
  match), a facets sanity test, and a `GetAdjacent` test against real
  category data. Self-cleaning (every inserted row is deleted in a
  `defer`) — confirmed row count back to the pre-test baseline (196) after
  a full run.
- Manual verification: started the server, ran `GET /products?limit=5`,
  `GET /products?search=daikin`, `GET /products?search=dieu+hoa` /
  `GET /products?search=điều+hòa` (both matched the same 6 "điều hòa" rows —
  see PR/task report for the actual curl output), a `min_price`/`max_price`
  request, `GET /products/{id}/adjacent`, `POST /products/by-ids`, and a
  full create → spec-facet-filter → delete round trip through the live API
  to prove `normalized_specs` populates correctly for new writes (since none
  of the 196 pre-existing rows have been backfilled). Server process killed
  afterward; verified no leftover rows in the shared `products` table.

## Deploy / ops notes specific to this module

None beyond the general ones in `ARCHITECTURE.md`. `immutable_unaccent` is a
new database-level function (created by this module's migration, dropped by
its `.down.sql`) — it's schema-scoped, not a Go platform change, but worth
knowing it exists if another module ever wants unaccented search/indexing
too (it can be reused as-is rather than redefined).
