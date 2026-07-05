package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/auth/domain"
)

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

var _ domain.SessionRepository = (*PostgresSessionRepository)(nil)

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

const sessionColumns = "id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, created_at"

func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.Session) (*domain.Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)
		RETURNING ` + sessionColumns

	row := r.pool.QueryRow(ctx, query, session.UserID(), session.TokenHash(), session.UserAgent(), session.IPAddress(), session.ExpiresAt())
	created, err := scanSession(row)
	if err != nil {
		return nil, fmt.Errorf("session repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+sessionColumns+" FROM sessions WHERE token_hash = $1", tokenHash)
	session, err := scanSession(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("session repository getByTokenHash: %w", err)
	}
	return session, nil
}

func (r *PostgresSessionRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("session repository revoke: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL", userID)
	if err != nil {
		return fmt.Errorf("session repository revokeAllByUserID: %w", err)
	}
	return nil
}

func scanSession(row rowScanner) (*domain.Session, error) {
	var (
		id, userID, tokenHash string
		userAgent, ipAddress  *string
		expiresAt, createdAt  time.Time
		revokedAt             *time.Time
	)

	if err := row.Scan(&id, &userID, &tokenHash, &userAgent, &ipAddress, &expiresAt, &revokedAt, &createdAt); err != nil {
		return nil, err
	}

	return domain.RehydrateSession(id, userID, tokenHash, deref(userAgent), deref(ipAddress), expiresAt, revokedAt, createdAt), nil
}
