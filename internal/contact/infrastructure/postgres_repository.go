package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/contact/domain"
)

// PostgresContactRepository implements domain.ContactRepository against the
// contacts table, whether that table lives on Supabase's Postgres (today) or
// a self-hosted one later — nothing here is Supabase-specific.
type PostgresContactRepository struct {
	pool *pgxpool.Pool
}

// Compile-time check: fails the build if this type ever stops satisfying the
// interface, instead of failing much later at the composition root.
var _ domain.ContactRepository = (*PostgresContactRepository)(nil)

func NewPostgresContactRepository(pool *pgxpool.Pool) *PostgresContactRepository {
	return &PostgresContactRepository{pool: pool}
}

const contactColumns = "id, type, label, value, is_active, order_index"

func (r *PostgresContactRepository) GetAll(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error) {
	query := "SELECT " + contactColumns + " FROM contacts"
	conditions := []string{}
	args := []any{}
	argN := 1

	if filter.Type != "" && filter.Type != "all" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argN))
		args = append(args, filter.Type)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(label ILIKE $%d OR value ILIKE $%d)", argN, argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY order_index ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("contact repository getAll: %w", err)
	}
	defer rows.Close()

	var contacts []*domain.Contact
	for rows.Next() {
		contact, err := scanContact(rows)
		if err != nil {
			return nil, fmt.Errorf("contact repository getAll scan: %w", err)
		}
		contacts = append(contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("contact repository getAll rows: %w", err)
	}

	return contacts, nil
}

func (r *PostgresContactRepository) Count(ctx context.Context, filter domain.ContactFilter) (int, error) {
	query := "SELECT COUNT(*) FROM contacts"
	args := []any{}

	if filter.Type != "" && filter.Type != "all" {
		query += " WHERE type = $1"
		args = append(args, filter.Type)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("contact repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresContactRepository) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	query := "SELECT " + contactColumns + " FROM contacts WHERE id = $1"

	row := r.pool.QueryRow(ctx, query, id)
	contact, err := scanContact(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("contact repository getById: %w", err)
	}
	return contact, nil
}

func (r *PostgresContactRepository) Create(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	query := `
		INSERT INTO contacts (type, label, value, is_active, order_index)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + contactColumns

	row := r.pool.QueryRow(ctx, query, contact.Type(), contact.Label(), contact.Value(), contact.IsActive(), contact.OrderIndex())
	created, err := scanContact(row)
	if err != nil {
		return nil, fmt.Errorf("contact repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresContactRepository) Update(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	query := `
		UPDATE contacts
		SET type = $1, label = $2, value = $3, is_active = $4, order_index = $5
		WHERE id = $6
		RETURNING ` + contactColumns

	row := r.pool.QueryRow(ctx, query, contact.Type(), contact.Label(), contact.Value(), contact.IsActive(), contact.OrderIndex(), contact.ID())
	updated, err := scanContact(row)
	if err != nil {
		return nil, fmt.Errorf("contact repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresContactRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM contacts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("contact repository delete: %w", err)
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query),
// so scanContact works for both single-row and multi-row callers.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanContact(row rowScanner) (*domain.Contact, error) {
	var (
		id         string
		typ        string
		label      *string
		value      string
		isActive   bool
		orderIndex int
	)

	if err := row.Scan(&id, &typ, &label, &value, &isActive, &orderIndex); err != nil {
		return nil, err
	}

	return domain.RehydrateContact(id, typ, label, value, isActive, orderIndex), nil
}
