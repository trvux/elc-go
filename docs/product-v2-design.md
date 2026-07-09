# Design: `catalog` → `product` v2 (B2C + B2B multi-brand appliance retail)

**Status**: Approved by the user 2026-07-08. Implementation in progress —
Phase 1 (rename `catalog` → `product`) is done; the core variant/option/
bundle slice (Phase 2) is being built now, see `docs/product.md` for the
as-built shape and `.claude/plans/fancy-snacking-turing.md` for the
implementation plan this phase follows. Pricing-by-customer-group,
promotions, the generalized attribute system, highlights, attached
services, and product relations remain future passes per the "Explicitly
out of scope" section below (unchanged from the original proposal).

**Why this exists**: the `catalog` module was originally built for a
marketing/showcase site — one flat `Product` row per SKU, one flat price, a
free-text stock label, HVAC-only spec normalization hard-coded in
`spec_normalizer.go`. The business has since shifted to a real online +
offline appliance retailer (máy lạnh, tủ lạnh, máy giặt, TV, etc.) selling
80% B2C / 20% B2B. That flat model can no longer represent the domain:
Daikin alone sells 5 different tiers (FTF/FTKB/FTKF/FTKY/FTKZ) each with
multiple capacities, most models come as an indoor+outdoor split system
(2 physical parts, 2 manufacturer part numbers, 1 sellable unit), B2B
customers need a different price than walk-in customers, and none of
highlights/attached-services/promotions/accessory cross-sell/review
aggregation exist today. This doc is the target design, researched against
how Shopify and Medusa (both audited via Context7) model the same problem,
adapted to this business's actual shape (a project-install/reseller
business with ~1 real showroom, not a multi-warehouse big-box retailer —
see "Explicitly out of scope" for why full multi-branch inventory was
rejected).

## Terminology fix: SKU vs MPN vs GTIN

The current `products.sku` column is misused as if it were the
manufacturer's own product code — e.g. a real row stores
`"FTKB25ZVMV / RKB25ZVMV"` (Daikin's indoor+outdoor part numbers, concatenated
into one string) as `sku`. Confirmed against Shopify's own field
definitions (`ProductVariant.sku`: *"the Stock Keeping Unit identifier for
inventory tracking"* vs `ProductVariant.barcode`: *"the barcode, UPC, ISBN,
or GTIN associated with the variant"*) and Google Merchant Center's product
data spec (requires GTIN, or Brand+MPN, to identify a product — never SKU):

| Field | Meaning | Who sets it | Customer-facing? |
|---|---|---|---|
| `sku` | Merchant's own internal inventory code | This business, any scheme | No |
| `mpn` | Manufacturer's part number for that exact model | Daikin/LG/Samsung/etc. | **Yes — this is what customers Google-search** |
| `gtin` | Global barcode (UPC/EAN) | GS1 registry | Scanned at POS / fed to Google Shopping |

v2 moves `mpn` to the primary, required, searchable identifier at the
**variant** level (each variant = one real manufacturer model, so each has
its own MPN). `sku` moves to variant level too but stays internal-only,
optional, auto-generated if left blank. `gtin` stays optional (barcode,
useful once POS/Google Shopping integration exists). Full-text search moves
from `name + sku` to `name + mpn`.

## Bounded contexts (Go modules)

| Module | Owns | Notes |
|---|---|---|
| `internal/product` (renamed from `internal/catalog`) | `Product`, `ProductOption`/`ProductOptionValue`, `ProductVariant`, `ProductVariantOptionValue`, `ProductVariantComponent`, `ProductHighlight`, `ProductService` (join to `service`), `ProductRelation`, `Attribute`/`ProductAttributeValue` | Aggregate root stays `Product`; everything below it is owned/loaded through it |
| `internal/pricing` (new) | `CustomerGroup`, `PriceList`, `PriceListItem` | Separate module — referenced by `variant_id` only, no cross-module FK, same pattern as existing modules linking by ID |
| `internal/promotion` (new) | `Promotion`, `PromotionTarget` | Time-bound campaigns, decoupled from base pricing |
| `internal/review` (existing, extended) | + aggregate rating computation | No new module; `product_id` FK already exists |
| `category`, `brand`, `group`, `tag`, `project-type` (existing) | Unchanged | Taxonomy layer is solid, reused as-is |

Explicitly **not** built as a module: `internal/inventory`/multi-branch
warehouse tracking — see "Explicitly out of scope".

## ERD

### `products` (trimmed — marketing-level fields only)

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `category_id`, `brand_id` | uuid | unchanged, NOT NULL |
| `product_line_id` | uuid, nullable | **new** — FK → `product_lines`, see below |
| `name` | text | unchanged |
| `slug` | text | unchanged |
| `short_description` | text, nullable | **new** — currently only exists as a TS-only, non-persisted field (`elc-tem/modules/catalog/domain/types.ts`); formalized here as a real column |
| `description` | jsonb | unchanged (Tiptap) |
| `images` | jsonb | unchanged |
| `labels` | text[] | unchanged |
| `is_featured`, `is_published`, `order_index` | unchanged | |
| `condition` | enum | unchanged |
| `warranty_months` | int, nullable | **new** |
| `warranty_terms` | text, nullable | **new** |
| `meta_title`/`meta_description`/`seo` | unchanged | |
| `created_at`/`updated_at`/`deleted_at` | unchanged | |

**Removed from this table** (moved to `product_variants`): `sku`,
`original_price`, `sale_price`, `discount_percent`, `stock_status`, `mpn`,
`gtin`. **Removed** (replaced by the attribute system below): `specs`,
`normalized_specs`.

### `product_lines` (new — "dòng sản phẩm", e.g. Daikin FTF/FTKB/FTKF/FTKY/FTKZ)

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `brand_id` | uuid | FK → `brands` |
| `category_id` | uuid, nullable | scope to a category if the line only applies there |
| `code` | text | e.g. `"FTKZ"` |
| `name` | text | e.g. `"Dòng Inverter siêu cao cấp"` |
| `tier_rank` | int | sort order, standard → premium |
| `description` | text, nullable | |

### `product_options` / `product_option_values` (new)

```
product_options: id, product_id, name (e.g. "Màu sắc", "Dung tích"), order_index
product_option_values: id, option_id, value (e.g. "Đỏ", "256GB"), order_index
```

### `product_variants` (new — where SKU/MPN/GTIN/price/stock actually live)

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `product_id` | uuid | FK → `products` |
| `mpn` | text | **required**, primary searchable identifier |
| `sku` | text, nullable | internal only, auto-generated if blank, never in public API response |
| `gtin` | text, nullable | barcode |
| `is_default` | bool | which variant a bare product-level view falls back to |
| `is_standalone` | bool, default `true` | **new** — `false` marks a variant that only exists as a component inside a bundle (see `product_variant_components`) and must never appear standalone in listing/search |
| `stock_status` | enum(`in_stock`/`order_from_supplier`/`discontinued`) | replaces the old free-text label; `order_from_supplier` reflects the real fulfillment model (order from đại lý on demand) instead of a fake "out of stock" |
| `lead_time_days` | int, nullable | expected wait when `order_from_supplier` |
| `cost_price` | numeric, nullable | **internal only**, never exposed in public API DTO — for margin visibility in admin |
| `weight`/`dimensions` | nullable | |
| `is_active`, `order_index` | | |
| `created_at`/`updated_at`/`deleted_at` | soft delete | |

### `product_variant_option_values` (new, join)

`(variant_id, option_value_id)` — defines which option-value combination a variant represents.

### `product_variant_components` (new — split systems, e.g. dàn lạnh/dàn nóng)

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `parent_variant_id` | uuid | the sellable bundle, e.g. "Set FTKB25" |
| `component_variant_id` | uuid | a part variant, e.g. "Dàn lạnh FTKB25" (own MPN/cost) |
| `quantity` | int, default `1` | |
| `role` | text, nullable | free text, e.g. `"Dàn lạnh"`/`"Dàn nóng"` — generic, not AC-specific, reusable for any multi-part bundle |

Component variants get `is_standalone = false` unless the business later
sells that exact part on its own (e.g. replacing a broken outdoor unit).

### Attribute system (new — replaces hard-coded `spec_normalizer.go`)

```
attributes: id, category_id (nullable = applies to every category), name, unit (nullable), data_type enum(text/number/boolean/enum), order_index
product_attribute_values: product_id, attribute_id, value_text, value_number
```

Each category defines its own attribute set (máy lạnh → Công suất/Loại
gas/Inverter; tủ lạnh → Dung tích/Số cửa; TV → Kích thước màn hình/Độ phân
giải) instead of one hard-coded 6-label HVAC whitelist. Enables real
indexed filtering instead of jsonb-array-of-freeform-objects.

### `product_highlights` (new — "Đặc điểm nổi bật")

`id, product_id, text, icon (nullable), order_index`

### `product_services` (new, join — "Dịch vụ đi kèm")

`product_id, service_id, is_included (bool), order_index` — reuses the
existing `service` module, no new service entity needed.

### `product_relations` (new — accessories / frequently-bought-together)

`id, product_id, related_product_id, relation_type enum(accessory/frequently_bought_together/replacement_part), order_index`

One self-referencing table covers accessory upsell (remote, ống đồng, giá
treo) and cross-sell, rather than a separate table per relation type.

### Pricing (new module `internal/pricing`)

```
customer_groups: id, name, slug (retail/b2b), is_default
price_lists: id, customer_group_id, name, currency default 'VND', priority, is_active
price_list_items: id, price_list_id, variant_id, min_qty default 1, list_price, tax_included bool default true, valid_from (nullable), valid_until (nullable)
```

`list_price` is the standing catalog price per customer group — **not**
where time-bound discounts live (see Promotion below). A variant can have
a "Bán lẻ" (retail) row and a "B2B" row simultaneously; effective price for
a request is resolved by customer group at query time. Quantity breaks
(`min_qty`) are modeled now even though not used day 1, so adding real
tiered pricing later doesn't require a schema change.

### Promotions (new module `internal/promotion`)

```
promotions: id, name, slug, description, type enum(percentage/fixed_amount/gift), value (nullable), gift_product_id (nullable), starts_at, ends_at, is_active
promotion_targets: promotion_id, target_type enum(product/variant/category/brand), target_id
```

Polymorphic target so one promotion can span many products/a whole brand at
once (e.g. "Sale tháng 7 toàn bộ Daikin"). Effective displayed price =
`price_list_items.list_price` minus any currently-active matching
promotion, computed in the application layer at read time — never
persisted as a mutated price, so an expired promotion can't leave stale
discounted prices behind.

### Reviews (existing module, extended)

No new tables. Add aggregate rating computation (avg + count) surfaced on
the product read model for `schema.org AggregateRating` (Google rich
snippets) — the actual driver behind this ask. Consider adding
`seller_reply`/`is_verified_purchase` to `reviews` for a more complete
Google-compliant review schema, tracked as a follow-up, not blocking.

## Explicitly out of scope (and why)

- **Multi-branch inventory / warehouse ledger.** Rejected after discussion:
  this business has 4 locations in HCMC but functionally one real
  showroom/office — it's a project-install/reseller model (order from
  khách → order from đại lý trung gian), not a big-box retailer holding
  stock at N stores. Building `inventory_levels`/`stock_locations` now
  would be solving a problem this business doesn't have. `stock_status` +
  `lead_time_days` on `product_variants` covers the real fulfillment model
  (in stock at the one showroom, or ordered on demand from a supplier).
  If the business model changes (opens real stocking branches), a proper
  multi-location inventory module can be added later without touching
  `product_variants`' shape — it would just gain a new referencing table.
- **B2B credit terms / quote approval workflow.** Deferred by explicit
  choice — v2 only adds a different price list per customer group. Credit
  limits, order approval, and formal quotes need an `order`/`customer`
  module that doesn't exist yet at all; building that workflow now would be
  designing ahead of a dependency that isn't there.
- **Order/cart/checkout.** Not audited or designed here — noted only so
  that `variant_id` and `price_list_item.id` are legitimate future FK
  targets, not a name collision with something to be redesigned later.

## Rollout plan (safe data migration, no destructive step until the very end)

1. **Rename** `internal/catalog` → `internal/product` (package, folder,
   `docs/catalog.md` → `docs/product.md`, `cmd/server/main.go` import alias,
   the module's own `schema_migrations_catalog` tracking table →
   `schema_migrations_product`), and `elc-tem/modules/catalog` →
   `elc-tem/modules/product`. Pure rename, no schema change, done first so
   every subsequent migration file lives under the right module name.
2. **Additive schema migration**: create every new table above via
   `golang-migrate`. Add `product_line_id`/`short_description`/
   `warranty_months`/`warranty_terms` to `products`. **Do not drop**
   `sku`/`original_price`/`sale_price`/`discount_percent`/`stock_status`/
   `specs`/`normalized_specs`/`mpn`/`gtin` yet — kept for dual-read safety
   during the transition.
3. **Backfill script**, run inside one transaction with a row-count
   verification before commit (`COUNT(product_variants) == COUNT(products)`,
   `COUNT(price_list_items) == COUNT(products)`; rollback on mismatch):
   - Each existing `products` row → one `product_variants` row
     (`is_default = true`, `mpn` seeded from the old `mpn` column if
     present else parsed out of the old combined `sku` string, `sku`
     seeded from the old `sku` column as-is, `stock_status` copied).
   - One `price_list_items` row per product against a seeded default
     "Bán lẻ" price list (`list_price = COALESCE(sale_price, original_price)`).
   - `specs` jsonb → `product_attribute_values`, mapped by hand per label
     (not fully automatable — old `specs` labels are freeform strings).
4. **Application/infrastructure/presentation** for the new modules, built
   per the existing layering conventions. The old `GET /products` response
   shape is preserved during the transition (default variant's
   price/stock mapped back into the legacy flat fields) so `elc-tem` isn't
   forced to cut over atomically.
5. **`elc-tem` cutover in two waves**: admin forms first (variant/option/
   price-list management UI — internal, lower risk), then storefront PDP
   (variant switcher, customer-group-aware pricing) — never both at once.
6. **Cleanup migration**: drop the deprecated flat columns from `products`
   only after grep confirms no remaining reader in `elc-tem` and production
   has run clean on the new shape for a period. `pg_dump` immediately
   before, per this project's existing dump-before-destructive-change habit.
