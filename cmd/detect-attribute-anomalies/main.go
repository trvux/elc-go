// Command detect-attribute-anomalies flags product_attribute_values rows
// that are statistical outliers vs. every other product's value for the
// same (number-type) attribute — IQR (interquartile range, Tukey's fences),
// not an admin-defined per-attribute range: the user explicitly rejected
// hand-entering a valid min/max for dozens of attributes as impractical.
// See docs/rfc/2026-08-18-product-data-anomaly-detection.md.
//
// internal/ai's search_products tool skips any flagged value when grounding
// an answer — a value this tool has never seen no admin-configured range
// for isn't "wrong", it's just excluded from what the model can quote.
//
// Idempotent: every run recomputes flagged_anomaly from scratch for every
// attribute, so fixing a bad value un-flags it on the next run rather than
// needing a manual reset. A plain one-shot CLI like every other cmd/ here,
// meant to run on a schedule (see .github/workflows for the cron wiring) —
// `go run ./cmd/detect-attribute-anomalies` runs it once.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// minSampleSize is the fewest products a number-type attribute must have a
// recorded value for before an IQR bound is computed for it — below this,
// "typical range" isn't statistically meaningful. Every row for that
// attribute is left (or reset) unflagged rather than guessing.
const minSampleSize = 10

// iqrMultiplier is the standard Tukey's-fences constant for a "mild"
// outlier (3.0 would be the stricter "extreme outlier" fence).
const iqrMultiplier = 1.5

type attributeDefRow struct {
	id, code string
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	defs, err := listNumberAttributeDefinitions(ctx, pool)
	if err != nil {
		log.Fatalf("list number attribute definitions: %v", err)
	}

	totalFlagged, totalSkipped := 0, 0
	for _, def := range defs {
		n, flagged, err := detectAnomaliesForAttribute(ctx, pool, def.id)
		if err != nil {
			log.Fatalf("attribute %s: %v", def.code, err)
		}
		if n < minSampleSize {
			fmt.Printf("detect-attribute-anomalies: %s: only %d value(s), skipping (need >= %d)\n", def.code, n, minSampleSize)
			totalSkipped++
			continue
		}
		fmt.Printf("detect-attribute-anomalies: %s: %d values, %d flagged as outliers\n", def.code, n, flagged)
		totalFlagged += flagged
	}

	fmt.Printf("detect-attribute-anomalies: done — %d attribute(s) skipped (too few values), %d value(s) flagged\n", totalSkipped, totalFlagged)
}

func listNumberAttributeDefinitions(ctx context.Context, pool *pgxpool.Pool) ([]attributeDefRow, error) {
	rows, err := pool.Query(ctx, `SELECT id, code FROM attribute_definitions WHERE data_type = 'number' AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []attributeDefRow
	for rows.Next() {
		var d attributeDefRow
		if err := rows.Scan(&d.id, &d.code); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		defs = append(defs, d)
	}
	return defs, rows.Err()
}

// detectAnomaliesForAttribute recomputes flagged_anomaly for every row of
// one attribute definition from scratch — idempotent, always reflects the
// current data.
func detectAnomaliesForAttribute(ctx context.Context, pool *pgxpool.Pool, attributeDefinitionID string) (n, flaggedCount int, err error) {
	rows, err := pool.Query(ctx, `
		SELECT id, value_number FROM product_attribute_values
		WHERE attribute_definition_id = $1 AND value_number IS NOT NULL AND deleted_at IS NULL`,
		attributeDefinitionID,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("query values: %w", err)
	}
	type valueRow struct {
		id    string
		value float64
	}
	var values []valueRow
	for rows.Next() {
		var r valueRow
		if err := rows.Scan(&r.id, &r.value); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan value: %w", err)
		}
		values = append(values, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("rows: %w", err)
	}

	n = len(values)
	if n < minSampleSize {
		// Not enough data to judge — reset any stale flag from a previous
		// run when the sample was larger (e.g. products got deleted since).
		if _, err := pool.Exec(ctx, `UPDATE product_attribute_values SET flagged_anomaly = false WHERE attribute_definition_id = $1`, attributeDefinitionID); err != nil {
			return n, 0, fmt.Errorf("reset flags (small sample): %w", err)
		}
		return n, 0, nil
	}

	sorted := make([]float64, n)
	for i, r := range values {
		sorted[i] = r.value
	}
	sort.Float64s(sorted)

	q1 := percentile(sorted, 0.25)
	q3 := percentile(sorted, 0.75)
	iqr := q3 - q1
	lower := q1 - iqrMultiplier*iqr
	upper := q3 + iqrMultiplier*iqr

	tx, err := pool.Begin(ctx)
	if err != nil {
		return n, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, r := range values {
		flagged := r.value < lower || r.value > upper
		if flagged {
			flaggedCount++
		}
		if _, err := tx.Exec(ctx, `UPDATE product_attribute_values SET flagged_anomaly = $1 WHERE id = $2`, flagged, r.id); err != nil {
			return n, flaggedCount, fmt.Errorf("update flag for %s: %w", r.id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return n, flaggedCount, fmt.Errorf("commit: %w", err)
	}
	return n, flaggedCount, nil
}

// percentile is linear-interpolation ("type 7", the common default —
// matches numpy/R) over an already-sorted slice.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := p * float64(n-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}
