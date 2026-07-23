package infrastructure

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

type PostgresWardRepository struct {
	pool *pgxpool.Pool
}

var _ domain.WardRepository = (*PostgresWardRepository)(nil)

func NewPostgresWardRepository(pool *pgxpool.Pool) *PostgresWardRepository {
	return &PostgresWardRepository{pool: pool}
}

func scanWards(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*domain.Ward, error) {
	var wards []*domain.Ward
	for rows.Next() {
		var w domain.Ward
		if err := rows.Scan(&w.Code, &w.Name, &w.ProvinceCode); err != nil {
			return nil, fmt.Errorf("ward repository scan: %w", err)
		}
		wards = append(wards, &w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ward repository rows: %w", err)
	}
	// Postgres' own ORDER BY sorts by this DB's collation (en_US.utf8, no
	// vi_VN locale on Alpine's musl libc) which puts every "Đ..." name after
	// Z instead of after D — same issue fixed for provinces in
	// postgres_province_repository.go, see vietnameseCollator there.
	sort.Slice(wards, func(i, j int) bool {
		return vietnameseCollator.CompareString(wards[i].Name, wards[j].Name) < 0
	})
	return wards, nil
}

func (r *PostgresWardRepository) GetAll(ctx context.Context) ([]*domain.Ward, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name, province_code FROM wards ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("ward repository getAll: %w", err)
	}
	defer rows.Close()
	return scanWards(rows)
}

func (r *PostgresWardRepository) GetByProvince(ctx context.Context, provinceCode string) ([]*domain.Ward, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name, province_code FROM wards WHERE province_code = $1 ORDER BY name ASC`, provinceCode)
	if err != nil {
		return nil, fmt.Errorf("ward repository getByProvince: %w", err)
	}
	defer rows.Close()
	return scanWards(rows)
}
