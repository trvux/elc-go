package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/chat-log/domain"
)

type PostgresChatLogRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ChatLogRepository = (*PostgresChatLogRepository)(nil)

func NewPostgresChatLogRepository(pool *pgxpool.Pool) *PostgresChatLogRepository {
	return &PostgresChatLogRepository{pool: pool}
}

const chatLogColumns = `id, visitor_id, message, kind, created_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PostgresChatLogRepository) Create(ctx context.Context, entry *domain.ChatLogEntry) (*domain.ChatLogEntry, error) {
	query := `
		INSERT INTO chat_log_entries (visitor_id, message, kind)
		VALUES ($1, $2, $3)
		RETURNING ` + chatLogColumns

	row := r.pool.QueryRow(ctx, query, entry.VisitorID(), entry.Message(), entry.Kind())
	created, err := scanChatLogEntry(row)
	if err != nil {
		return nil, fmt.Errorf("chat-log repository create: %w", err)
	}
	return created, nil
}

// GetAll defaults to 50 rows, capped at 200 — an admin analysis view, not a
// full unbounded export; Kind/Search (message ILIKE) both narrow further
// when set.
func (r *PostgresChatLogRepository) GetAll(ctx context.Context, filter domain.ChatLogFilter) ([]*domain.ChatLogEntry, error) {
	query := `SELECT ` + chatLogColumns + ` FROM chat_log_entries WHERE 1=1`
	var args []any
	argN := 1

	if filter.Kind != "" {
		query += fmt.Sprintf(" AND kind = $%d", argN)
		args = append(args, filter.Kind)
		argN++
	}
	if filter.Search != "" {
		query += fmt.Sprintf(" AND message ILIKE $%d", argN)
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("chat-log repository get all: %w", err)
	}
	defer rows.Close()

	var entries []*domain.ChatLogEntry
	for rows.Next() {
		entry, err := scanChatLogEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("chat-log repository get all scan: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("chat-log repository get all rows: %w", err)
	}
	return entries, nil
}

func (r *PostgresChatLogRepository) Count(ctx context.Context, filter domain.ChatLogFilter) (int, error) {
	query := `SELECT count(*) FROM chat_log_entries WHERE 1=1`
	var args []any
	argN := 1

	if filter.Kind != "" {
		query += fmt.Sprintf(" AND kind = $%d", argN)
		args = append(args, filter.Kind)
		argN++
	}
	if filter.Search != "" {
		query += fmt.Sprintf(" AND message ILIKE $%d", argN)
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("chat-log repository count: %w", err)
	}
	return count, nil
}

func scanChatLogEntry(row rowScanner) (*domain.ChatLogEntry, error) {
	var (
		id, visitorID, message, kind string
		createdAt                   time.Time
	)
	if err := row.Scan(&id, &visitorID, &message, &kind, &createdAt); err != nil {
		return nil, err
	}
	return domain.RehydrateChatLogEntry(id, visitorID, message, kind, createdAt), nil
}
