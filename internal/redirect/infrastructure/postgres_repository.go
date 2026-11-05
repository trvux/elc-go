package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const columns = `id, from_path, to_path, is_active, source, note, created_at, updated_at`

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner) (*domain.Redirect, error) {
	var (
		id, from, to, source, note string
		active                     bool
		created, updated           time.Time
	)
	if err := row.Scan(&id, &from, &to, &active, &source, &note, &created, &updated); err != nil {
		return nil, err
	}
	return domain.RehydrateRedirect(id, from, to, active, domain.Source(source), note, created, updated), nil
}

func (r *PostgresRepository) List(ctx context.Context, f domain.Filter) ([]*domain.Redirect, int, error) {
	where := []string{"TRUE"}
	args := []any{}
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+s+"%")
		where = append(where, fmt.Sprintf("(from_path ILIKE $%d OR to_path ILIKE $%d OR note ILIKE $%d)", len(args), len(args), len(args)))
	}
	if f.Source != "" {
		args = append(args, string(f.Source))
		where = append(where, fmt.Sprintf("source = $%d", len(args)))
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM redirects WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("redirect repository list count: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf("SELECT %s FROM redirects WHERE %s ORDER BY updated_at DESC, from_path ASC LIMIT $%d OFFSET $%d",
		columns, clause, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("redirect repository list: %w", err)
	}
	defer rows.Close()

	items := []*domain.Redirect{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("redirect repository list scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("redirect repository list rows: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Redirect, error) {
	return r.getOne(ctx, "SELECT "+columns+" FROM redirects WHERE id = $1", id)
}

func (r *PostgresRepository) GetByFrom(ctx context.Context, fromPath string) (*domain.Redirect, error) {
	return r.getOne(ctx, "SELECT "+columns+" FROM redirects WHERE from_path = $1", fromPath)
}

func (r *PostgresRepository) getOne(ctx context.Context, query string, arg any) (*domain.Redirect, error) {
	item, err := scan(r.pool.QueryRow(ctx, query, arg))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("redirect repository get: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Create(ctx context.Context, d *domain.Redirect) (*domain.Redirect, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO redirects (from_path, to_path, is_active, source, note) VALUES ($1, $2, $3, $4, $5) RETURNING `+columns,
		d.FromPath(), d.ToPath(), d.IsActive(), string(d.Source()), d.Note())
	created, err := scan(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperr.NewConflictError("đã có chuyển hướng cho đường dẫn " + d.FromPath())
		}
		return nil, fmt.Errorf("redirect repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) Update(ctx context.Context, d *domain.Redirect) (*domain.Redirect, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE redirects SET to_path = $1, is_active = $2, note = $3, updated_at = now() WHERE id = $4 RETURNING `+columns,
		d.ToPath(), d.IsActive(), d.Note(), d.ID())
	updated, err := scan(row)
	if err != nil {
		return nil, fmt.Errorf("redirect repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	if _, err := r.pool.Exec(ctx, "DELETE FROM redirects WHERE id = $1", id); err != nil {
		return fmt.Errorf("redirect repository delete: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Repoint(ctx context.Context, oldTarget, newTarget string) error {
	if oldTarget == newTarget {
		return nil
	}
	if _, err := r.pool.Exec(ctx,
		"UPDATE redirects SET to_path = $2, updated_at = now() WHERE to_path = $1 AND from_path <> $2",
		oldTarget, newTarget); err != nil {
		return fmt.Errorf("redirect repository repoint: %w", err)
	}
	return nil
}
