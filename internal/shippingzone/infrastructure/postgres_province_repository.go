package infrastructure

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

// vietnameseCollator sorts by proper Vietnamese alphabet order (Đ right
// after D, correct tone-mark order, ...) — Postgres' own ORDER BY sorted by
// this table's DB collation (en_US.utf8, since Alpine's musl libc has no
// vi_VN locale), which put every "Đ..." name after Z instead of after D.
var vietnameseCollator = collate.New(language.Vietnamese)

type PostgresProvinceRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProvinceRepository = (*PostgresProvinceRepository)(nil)

func NewPostgresProvinceRepository(pool *pgxpool.Pool) *PostgresProvinceRepository {
	return &PostgresProvinceRepository{pool: pool}
}

func (r *PostgresProvinceRepository) GetAll(ctx context.Context) ([]*domain.Province, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM provinces`)
	if err != nil {
		return nil, fmt.Errorf("province repository getAll: %w", err)
	}
	defer rows.Close()

	var provinces []*domain.Province
	for rows.Next() {
		var p domain.Province
		if err := rows.Scan(&p.Code, &p.Name); err != nil {
			return nil, fmt.Errorf("province repository getAll scan: %w", err)
		}
		provinces = append(provinces, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("province repository getAll rows: %w", err)
	}

	sort.Slice(provinces, func(i, j int) bool {
		return vietnameseCollator.CompareString(provinces[i].Name, provinces[j].Name) < 0
	})
	return provinces, nil
}

func (r *PostgresProvinceRepository) Create(ctx context.Context, province *domain.Province) (*domain.Province, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO provinces (code, name) VALUES ($1, $2)`, province.Code, province.Name)
	if err != nil {
		return nil, fmt.Errorf("province repository create: %w", err)
	}
	return province, nil
}

func (r *PostgresProvinceRepository) Update(ctx context.Context, code, name string) (*domain.Province, error) {
	_, err := r.pool.Exec(ctx, `UPDATE provinces SET name = $1, updated_at = now() WHERE code = $2`, name, code)
	if err != nil {
		return nil, fmt.Errorf("province repository update: %w", err)
	}
	return &domain.Province{Code: code, Name: name}, nil
}

func (r *PostgresProvinceRepository) Delete(ctx context.Context, code string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM provinces WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("province repository delete: %w", err)
	}
	return nil
}
