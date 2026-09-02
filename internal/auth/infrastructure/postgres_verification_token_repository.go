package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/auth/domain"
)

type PostgresVerificationTokenRepository struct {
	pool *pgxpool.Pool
}

var _ domain.VerificationTokenRepository = (*PostgresVerificationTokenRepository)(nil)

func NewPostgresVerificationTokenRepository(pool *pgxpool.Pool) *PostgresVerificationTokenRepository {
	return &PostgresVerificationTokenRepository{pool: pool}
}

const verificationTokenColumns = "id, purpose, token_hash, email, code, attempts, expires_at, consumed_at, created_at"

func (r *PostgresVerificationTokenRepository) Create(ctx context.Context, token *domain.VerificationToken) (*domain.VerificationToken, error) {
	query := `
		INSERT INTO verification_tokens (purpose, token_hash, email, code, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + verificationTokenColumns

	row := r.pool.QueryRow(ctx, query,
		string(token.Purpose()), token.TokenHash(), token.Email(), token.Code(), token.ExpiresAt(),
	)
	created, err := scanVerificationToken(row)
	if err != nil {
		return nil, fmt.Errorf("verification token repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresVerificationTokenRepository) GetByHash(ctx context.Context, purpose domain.TokenPurpose, tokenHash string) (*domain.VerificationToken, error) {
	query := "SELECT " + verificationTokenColumns + " FROM verification_tokens WHERE purpose = $1 AND token_hash = $2"
	row := r.pool.QueryRow(ctx, query, string(purpose), tokenHash)
	token, err := scanVerificationToken(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("verification token repository getByHash: %w", err)
	}
	return token, nil
}

// GetLatestActiveByEmail returns the newest not-consumed, not-expired token
// for this email — NOT matched by code, so the caller (VerifyMagicLink) can
// compare the code itself and count wrong guesses instead of every wrong
// guess looking identical to "no such token".
func (r *PostgresVerificationTokenRepository) GetLatestActiveByEmail(ctx context.Context, purpose domain.TokenPurpose, email string) (*domain.VerificationToken, error) {
	query := `
		SELECT ` + verificationTokenColumns + `
		FROM verification_tokens
		WHERE purpose = $1 AND email = lower($2) AND consumed_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, string(purpose), email)
	token, err := scanVerificationToken(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("verification token repository getLatestActiveByEmail: %w", err)
	}
	return token, nil
}

func (r *PostgresVerificationTokenRepository) MarkConsumed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE verification_tokens SET consumed_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("verification token repository markConsumed: %w", err)
	}
	return nil
}

// IncrementAttempts returns the new count so the caller can decide whether to
// lock the token out. The increment and RETURNING happen in one statement so
// concurrent wrong guesses on the same row still serialize correctly instead
// of racing on a read-then-write.
func (r *PostgresVerificationTokenRepository) IncrementAttempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := r.pool.QueryRow(ctx, "UPDATE verification_tokens SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts", id).Scan(&attempts)
	if err != nil {
		return 0, fmt.Errorf("verification token repository incrementAttempts: %w", err)
	}
	return attempts, nil
}

func scanVerificationToken(row rowScanner) (*domain.VerificationToken, error) {
	var (
		id, purpose, tokenHash, email string
		code                          *string
		attempts                      int
		expiresAt, createdAt          time.Time
		consumedAt                    *time.Time
	)

	if err := row.Scan(&id, &purpose, &tokenHash, &email, &code, &attempts, &expiresAt, &consumedAt, &createdAt); err != nil {
		return nil, err
	}

	return domain.RehydrateVerificationToken(
		id,
		domain.TokenPurpose(purpose),
		tokenHash,
		email,
		deref(code),
		attempts,
		expiresAt,
		consumedAt,
		createdAt,
	), nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
