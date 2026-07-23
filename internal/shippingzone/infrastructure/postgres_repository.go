package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

type PostgresShippingZoneRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ShippingZoneRepository = (*PostgresShippingZoneRepository)(nil)

func NewPostgresShippingZoneRepository(pool *pgxpool.Pool) *PostgresShippingZoneRepository {
	return &PostgresShippingZoneRepository{pool: pool}
}

const zoneColumns = `id, name, fee_vnd, min_days, max_days, is_default, created_at, updated_at, deleted_at`

func (r *PostgresShippingZoneRepository) GetAll(ctx context.Context, filter domain.ZoneFilter) ([]*domain.ShippingZone, error) {
	query := `SELECT ` + zoneColumns + ` FROM shipping_zones`
	if !filter.IncludeDeleted {
		query += ` WHERE deleted_at IS NULL`
	}
	query += ` ORDER BY is_default DESC, name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository getAll: %w", err)
	}
	defer rows.Close()

	var zones []*domain.ShippingZone
	for rows.Next() {
		z, err := scanZone(rows)
		if err != nil {
			return nil, fmt.Errorf("shippingzone repository getAll scan: %w", err)
		}
		zones = append(zones, z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("shippingzone repository getAll rows: %w", err)
	}

	if err := r.hydrateRelations(ctx, zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (r *PostgresShippingZoneRepository) GetByID(ctx context.Context, id string) (*domain.ShippingZone, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+zoneColumns+` FROM shipping_zones WHERE id = $1 AND deleted_at IS NULL`, id)
	z, err := scanZone(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("shippingzone repository getById: %w", err)
	}
	if err := r.hydrateRelations(ctx, []*domain.ShippingZone{z}); err != nil {
		return nil, err
	}
	return z, nil
}

func (r *PostgresShippingZoneRepository) GetDefault(ctx context.Context) (*domain.ShippingZone, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+zoneColumns+` FROM shipping_zones WHERE is_default = true AND deleted_at IS NULL LIMIT 1`)
	z, err := scanZone(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("shippingzone repository getDefault: %w", err)
	}
	if err := r.hydrateRelations(ctx, []*domain.ShippingZone{z}); err != nil {
		return nil, err
	}
	return z, nil
}

func (r *PostgresShippingZoneRepository) FindByProvince(ctx context.Context, provinceCode string) ([]*domain.ShippingZone, error) {
	query := `SELECT z.` + strings.ReplaceAll(zoneColumns, ", ", ", z.") + `
		FROM shipping_zones z
		JOIN shipping_zone_provinces zp ON zp.zone_id = z.id
		WHERE zp.province_code = $1 AND z.deleted_at IS NULL
		ORDER BY z.name ASC`

	rows, err := r.pool.Query(ctx, query, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository findByProvince: %w", err)
	}
	defer rows.Close()

	var zones []*domain.ShippingZone
	for rows.Next() {
		z, err := scanZone(rows)
		if err != nil {
			return nil, fmt.Errorf("shippingzone repository findByProvince scan: %w", err)
		}
		zones = append(zones, z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("shippingzone repository findByProvince rows: %w", err)
	}

	if err := r.hydrateRelations(ctx, zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (r *PostgresShippingZoneRepository) Create(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository create (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	if zone.IsDefault() {
		if _, err := tx.Exec(ctx, `UPDATE shipping_zones SET is_default = false WHERE is_default = true`); err != nil {
			return nil, fmt.Errorf("shippingzone repository create (clear default): %w", err)
		}
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO shipping_zones (name, fee_vnd, min_days, max_days, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+zoneColumns,
		zone.Name(), zone.FeeVND(), zone.MinDays(), zone.MaxDays(), zone.IsDefault(),
	)
	created, err := scanZone(row)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository create: %w", err)
	}

	if err := replaceRelationsTx(ctx, tx, created.ID(), zone.ProvinceCodes(), zone.WardCodes()); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("shippingzone repository create (commit tx): %w", err)
	}

	return r.GetByID(ctx, created.ID())
}

func (r *PostgresShippingZoneRepository) Update(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository update (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	if zone.IsDefault() {
		if _, err := tx.Exec(ctx, `UPDATE shipping_zones SET is_default = false WHERE is_default = true AND id != $1`, zone.ID()); err != nil {
			return nil, fmt.Errorf("shippingzone repository update (clear default): %w", err)
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE shipping_zones
		SET name = $1, fee_vnd = $2, min_days = $3, max_days = $4, is_default = $5, updated_at = $6
		WHERE id = $7
		RETURNING `+zoneColumns,
		zone.Name(), zone.FeeVND(), zone.MinDays(), zone.MaxDays(), zone.IsDefault(), zone.UpdatedAt(), zone.ID(),
	)
	updated, err := scanZone(row)
	if err != nil {
		return nil, fmt.Errorf("shippingzone repository update: %w", err)
	}

	if err := replaceRelationsTx(ctx, tx, updated.ID(), zone.ProvinceCodes(), zone.WardCodes()); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("shippingzone repository update (commit tx): %w", err)
	}

	return r.GetByID(ctx, updated.ID())
}

func (r *PostgresShippingZoneRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE shipping_zones SET deleted_at = $1, updated_at = $1 WHERE id = $2`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("shippingzone repository softDelete: %w", err)
	}
	return nil
}

// replaceRelationsTx always keeps ProvinceCodes()/WardCodes() nil-safe — a
// zone's provinces/wards are always submitted in full on create/update
// (there's no partial-patch endpoint for them), so replacing the join rows
// wholesale is simpler and avoids a diff.
func replaceRelationsTx(ctx context.Context, tx pgx.Tx, zoneID string, provinceCodes, wardCodes []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM shipping_zone_provinces WHERE zone_id = $1`, zoneID); err != nil {
		return fmt.Errorf("shippingzone repository replaceRelations (delete provinces): %w", err)
	}
	for _, code := range provinceCodes {
		if _, err := tx.Exec(ctx, `INSERT INTO shipping_zone_provinces (zone_id, province_code) VALUES ($1, $2)`, zoneID, code); err != nil {
			return fmt.Errorf("shippingzone repository replaceRelations (insert province %s): %w", code, err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM shipping_zone_wards WHERE zone_id = $1`, zoneID); err != nil {
		return fmt.Errorf("shippingzone repository replaceRelations (delete wards): %w", err)
	}
	for _, wc := range wardCodes {
		if wc == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO shipping_zone_wards (zone_id, ward_code) VALUES ($1, $2)`, zoneID, wc); err != nil {
			return fmt.Errorf("shippingzone repository replaceRelations (insert ward %s): %w", wc, err)
		}
	}
	return nil
}

// hydrateRelations batch-loads province codes + ward codes for the given
// zones and sets them via a package-private rehydrate helper — kept as a
// second pass (rather than a single joined query) because a zone can have
// many provinces and many wards, and joining both at once would duplicate
// rows on the cartesian product.
func (r *PostgresShippingZoneRepository) hydrateRelations(ctx context.Context, zones []*domain.ShippingZone) error {
	if len(zones) == 0 {
		return nil
	}
	ids := make([]string, len(zones))
	byID := make(map[string]*domain.ShippingZone, len(zones))
	for i, z := range zones {
		ids[i] = z.ID()
		byID[z.ID()] = z
	}

	provinceRows, err := r.pool.Query(ctx, `SELECT zone_id, province_code FROM shipping_zone_provinces WHERE zone_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("shippingzone repository hydrateRelations (provinces): %w", err)
	}
	provinceCodes := map[string][]string{}
	for provinceRows.Next() {
		var zoneID, code string
		if err := provinceRows.Scan(&zoneID, &code); err != nil {
			provinceRows.Close()
			return fmt.Errorf("shippingzone repository hydrateRelations (provinces scan): %w", err)
		}
		provinceCodes[zoneID] = append(provinceCodes[zoneID], code)
	}
	provinceRows.Close()
	if err := provinceRows.Err(); err != nil {
		return fmt.Errorf("shippingzone repository hydrateRelations (provinces rows): %w", err)
	}

	wardRows, err := r.pool.Query(ctx, `SELECT zone_id, ward_code FROM shipping_zone_wards WHERE zone_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("shippingzone repository hydrateRelations (wards): %w", err)
	}
	wardCodes := map[string][]string{}
	for wardRows.Next() {
		var zoneID, wc string
		if err := wardRows.Scan(&zoneID, &wc); err != nil {
			wardRows.Close()
			return fmt.Errorf("shippingzone repository hydrateRelations (wards scan): %w", err)
		}
		wardCodes[zoneID] = append(wardCodes[zoneID], wc)
	}
	wardRows.Close()
	if err := wardRows.Err(); err != nil {
		return fmt.Errorf("shippingzone repository hydrateRelations (wards rows): %w", err)
	}

	for id, z := range byID {
		z.SetRelations(provinceCodes[id], wardCodes[id])
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanZone(row rowScanner) (*domain.ShippingZone, error) {
	var (
		id, name             string
		feeVND               int64
		minDays, maxDays     int
		isDefault            bool
		createdAt, updatedAt time.Time
		deletedAt            *time.Time
	)
	if err := row.Scan(&id, &name, &feeVND, &minDays, &maxDays, &isDefault, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, err
	}
	return domain.RehydrateShippingZone(id, name, feeVND, minDays, maxDays, isDefault, nil, nil, createdAt, updatedAt, deletedAt), nil
}
