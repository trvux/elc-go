# Catalog data audit — 2026-07-08

Scope: all 197 products in the migrated `products` table, across all 4 brands and 12 categories. Queried directly against `elc-postgres` (see queries inline). Numbers below are exact counts, not estimates.

## 5. Overall counts (context for everything below)

- **Total products: 197**
- **By category:**

| category | products |
|---|---|
| Máy lạnh treo tường | 60 |
| Máy lạnh giấu trần nối ống gió | 31 |
| Máy lạnh âm trần đa hướng thổi | 28 |
| Máy cấp khí tươi, lọc không khí | 25 |
| Máy lạnh áp trần | 23 |
| Bảng điều khiển | 7 |
| Máy lạnh tủ đứng | 6 |
| Máy lọc nước RO 3 in 1 | 5 |
| Công tắc thông minh | 5 |
| Phụ kiện đồng bộ của hệ thống cấp gió tươi | 3 |
| Cảm biến thông minh | 2 |
| Remote cầm tay | 2 |

- **By brand (top, only 4 brands exist in the catalog):** Daikin 135, Menred 33, Acis 16, LG 13.
- **`mpn` populated (non-null/non-empty): 176 / 197 (89%).** The 21 without: mostly Acis smart-home devices (no mpn recorded at all) and a few Menred water purifiers.
- **`sku` containing `/` or `+` (multi-part signal): 134 / 197 (68%).** Every product whose sku contains `+` also contains `/` (88 contain `+`, all 88 are a subset of the 134 that contain `/`), so `/` alone is the reliable multi-part signal.
- Note: a stray `Throwaway Test Product Phase2 UPDATED` record exists under Máy lạnh treo tường (Daikin) — looks like a leftover dev/test row, not real catalog data. Flagging for cleanup, not counted as a pattern above.

## 1. Near-duplicate product pairs/groups that should be one product with a variant

**Method:** stripped `(1 pha)` / `(3 pha)` from `name` and grouped by (brand, category, stripped name). 36 groups have more than one product sharing a stripped name. Of those, only when the **same `mpn`** also repeats within the group is it a genuine "same indoor unit, different outdoor unit/phase, different price" duplicate — the rest are same-name-different-hardware (see the important caveat below).

**True duplicate pairs (same mpn, differing only by 1-pha/3-pha outdoor unit and price): 16 pairs = 32 products**, all in Daikin's three *ducted/built-in* categories — none in wall-mounted (treo tường) or floor-standing (tủ đứng):

| category | true duplicate pairs |
|---|---|
| Máy lạnh giấu trần nối ống gió (ducted) | 9 |
| Máy lạnh âm trần đa hướng thổi (4-way cassette) | 4 |
| Máy lạnh áp trần (low-static ceiling) | 3 |

Examples:
- `Máy lạnh giấu trần nối ống gió 4HP Daikin inverter` — mpn `FBA100BVMA9` appears twice: `( 1 pha )` sku `FBA100BVMA9 / RZF100CVM + BRC1E63`, and `( 3 pha )` sku `FBA100BVMA9 / RZF100CYM + BRC1E63`. Same indoor unit, outdoor unit differs by one letter (`CVM`→`CYM`).
- `Máy lạnh áp trần DaiKin inverter 5HP` — mpn `FHA125CVMA` twice, sku differs `RZA125DV1` (1 pha) vs `RZA125DY1` (3 pha).
- `Máy lạnh âm trần đa hướng thổi 4HP Daikin inverter` — mpn `FCF100CVM` twice, prices differ (47,422,320 vs 52,634,120 — note this pair's "outdoor" sku actually differs by more than just the phase digit, `RZF100CVM` vs `RZF100DVM`, worth a manual glance before merging).

Caveat for anyone who checked wall-mounted units expecting this pattern there: **wall-mounted (treo tường) does not have this issue today.** It has a *different, already-handled* issue — see finding 3 below. The remaining 20 of the 36 stripped-name groups (all in treo tường: 10 Daikin groups + 4 LG groups) are cases where a completely different `mpn` (different tier/series) shares one generic name (e.g. 7 different Daikin wall units — FTF, FTKY, FTKM, FTKF, FTKB, FTHB, FTKZ — all literally named "Máy lạnh treo tường Daikin 2HP - một chiều Inverter" with no distinguishing text). Those are genuinely different products with different features and prices; merging them into one product with variants would be wrong. This is already correctly handled in the new schema via `product_lines` (5 tiers defined for Daikin treo tường: FTF-FTC, FTKB-ATKB, FTKF-ATKF, FTKC-FTKA, FTKY-FTKM-FTKZ) — each product keeps its own row and is tagged with a `product_line_id`, not merged into variants. LG's 4 groups follow the identical shape (see finding 3).

No other duplicate-suffix pattern (color, "có/không remote", bundle variants) was found by scanning all 197 product names — the only recurring suffix is the phase marker.

**Recommendation:** the 16 pairs above are safe candidates for merging into one product with a "Điện áp" (1 pha / 3 pha) option + two variants, mirroring the Daikin treo tường fix pattern used elsewhere. Do not touch the treo tường "same name, different mpn" groups the same way — those need a naming/tiering fix (add tier text to `name`, or rely on `product_line_id`), not a merge.

## 2. Multi-part `sku` (bundle candidates for `product_variant_components`)

**134 / 197 products (68%) have a `sku` containing `/` (indoor/outdoor combined code), of which 88 also contain `+` (remote/extra accessory appended, e.g. `+ BRC2E61`).**

By category:

| category | total | multi-part sku | pattern |
|---|---|---|---|
| Máy lạnh giấu trần nối ống gió | 31 | 31 (100%) | `INDOOR / OUTDOOR + REMOTE[+ ACCESSORY]` |
| Máy lạnh âm trần đa hướng thổi | 28 | 28 (100%) | `INDOOR / OUTDOOR + REMOTE + BYCQ...` (extra plenum/diffuser part) |
| Máy lạnh áp trần | 23 | 23 (100%) | `INDOOR / OUTDOOR + REMOTE` |
| Máy lạnh tủ đứng | 6 | 6 (100%) | `INDOOR / OUTDOOR + REMOTE` |
| Máy lạnh treo tường | 60 | 46 (77%) | Daikin: `INDOOR / OUTDOOR`; LG (13) and 1 test row: clean single code, no bundling |
| Máy cấp khí tươi, lọc không khí | 25 | 0 | single clean model code |
| Bảng điều khiển | 7 | 0 | single clean code |
| Máy lọc nước RO 3 in 1 | 5 | 0 | single clean code |
| Công tắc thông minh | 5 | 0 | single clean code |
| Phụ kiện đồng bộ của hệ thống cấp gió tươi | 3 | 0 | single clean code |
| Cảm biến thông minh | 2 | 0 | single clean code |
| Remote cầm tay | 2 | 0 | single clean code |

Example multi-part skus:
- `FBFC100DVM9 / RZFC100EVM + BRC2E61` (ducted: indoor / outdoor + wired remote)
- `FCFC100DVM / RZFC100EY1 + BRC2E61 + BYCQ125EAF8` (4-way cassette: indoor / outdoor + remote + decorative panel — 3-part bundle)
- `FVA100AMVM / RZF100CVM + BRC1E63` (floor-standing: indoor / outdoor + remote)

**Categories where sku is already a single clean code (no bundling needed): every non-air-conditioner category** — fresh-air/ventilation (Menred), all 4 smart-home accessory categories (Acis), and water purifiers (Menred) — plus LG wall-mounted units. Only Daikin's split-system air conditioners (wall-mounted, ducted, cassette, low-static ceiling, floor-standing) use the combined-code pattern, because Daikin's indoor/outdoor unit + remote are genuinely sold and priced as one bundle. This maps directly onto `product_variant_components`.

## 3. MPN-prefix tiering pattern in other brands

**LG (13 products, wall-mounted only) shows the same tiering pattern as Daikin**, confirmed by consistent price ordering at every capacity level:

| capacity | IEC (economy) | IDC (standard) | IDH (higher) | IPC (premium) |
|---|---|---|---|---|
| 1HP | 6,550,000 (IEC09G2) | 9,450,000 (IDC09M1) | 10,550,000 (IDH09M1) | 12,650,000 (IPC09M1) |
| 1.5HP | 8,000,000 (IEC12G2) | 10,750,000 (IDC12M1) | 12,750,000 (IDH12M1) | 15,150,000 (IPC12M1) |
| 2HP | 11,750,000 (IEC18M2) | 16,250,000 (IDC18M1) | 19,500,000 (IDH18M1) | — |
| 2.5HP | 15,100,000 (IEC24M2) | — | 24,950,000 (IDH24M1) | — |

Ordering `IEC < IDC < IDH < IPC` holds at every capacity where multiple prefixes exist — same shape as Daikin's FTF < FTKB < FTKF < FTKC/FTKA < FTKY/FTKM/FTKZ tiering, just with different prefix letters and only 4 tiers observed (vs Daikin's 5) and gaps at higher capacities. Recommend the same `product_lines` treatment for LG treo tường (4 tiers: IEC, IDC, IDH, IPC) once the LG catalog grows.

**Menred (33 products: fresh-air/ventilation + water purifiers) shows no prefix-tiering pattern.** Its mpns (`NET.800`…`NET.8000`, `N5.150A`, `N5.250A`, `P5-`/`P7-`, `R150-`/`R250-`/`R350-`, `S5-`, `G2`/`G5`/`G7`, `O2 S1`/`O2 G3`) are distinct model families differentiated by airflow capacity, not quality tiers of the same physical unit — each is already a genuinely separate product. No tiering fix needed here.

**Acis (16 products: smart-home panels/switches/sensors/remotes) shows no prefix-tiering pattern either** — it has no `mpn` values at all, and each sku (`BDKTHTM`, `CTCATM`, `CBHD`, etc.) names a functionally distinct device type, not a tier of one device.

## 4. Categories where none of the above patterns apply

These categories have **no phase/variant duplication and no multi-part sku** — every product is already a standalone, correctly-modeled item requiring no Options/Bundle work:

- **Máy cấp khí tươi, lọc không khí** (25, Menred) — fresh-air/ventilation units, each a distinct capacity model.
- **Máy lọc nước RO 3 in 1** (5, Menred) — water purifiers, each distinct.
- **Phụ kiện đồng bộ của hệ thống cấp gió tươi** (3, Menred) — accessories, each distinct.
- **Bảng điều khiển** (7, Acis) — control panels, each distinct.
- **Công tắc thông minh** (5, Acis) — smart switches, each distinct.
- **Cảm biến thông minh** (2, Acis) — sensors, each distinct.
- **Remote cầm tay** (2, Acis) — remotes/modules, each distinct.

That's 7 of 12 categories (49 of 197 products, 25%) needing no catalog restructuring at all.

One near-miss: **Máy lạnh tủ đứng** (floor-standing, 6, Daikin) has no duplicate/variant issue (each capacity is one clean product), but its sku *is* the multi-part indoor/outdoor+remote pattern — so it needs bundle-component modeling (finding 2) even though it needs no variant/option work (finding 1).
