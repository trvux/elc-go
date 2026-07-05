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

const verificationTokenColumns = "id, purpose, token_hash, email, role, invited_by, user_id, expires_at, consumed_at, created_at"

func (r *PostgresVerificationTokenRepository) Create(ctx context.Context, token *domain.VerificationToken) (*domain.VerificationToken, error) {
	// NULLIF forces early type resolution of its arguments to text (a
	// Postgres quirk — unlike a bare VALUES literal, which can stay
	// "unknown" and coerce to the column's type), so the result needs an
	// explicit cast back to the actual column type before it can be
	// assigned to an enum/uuid column.
	query := `
		INSERT INTO verification_tokens (purpose, token_hash, email, role, invited_by, user_id, expires_at)
		VALUES ($1, $2, $3, NULLIF($4, '')::auth_role, NULLIF($5, '')::uuid, NULLIF($6, '')::uuid, $7)
		RETURNING ` + verificationTokenColumns

	row := r.pool.QueryRow(ctx, query,
		string(token.Purpose()), token.TokenHash(), token.Email(), string(token.Role()),
		token.InvitedBy(), token.UserID(), token.ExpiresAt(),
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

func (r *PostgresVerificationTokenRepository) MarkConsumed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE verification_tokens SET consumed_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("verification token repository markConsumed: %w", err)
	}
	return nil
}

func scanVerificationToken(row rowScanner) (*domain.VerificationToken, error) {
	var (
		id, purpose, tokenHash, email string
		role, invitedBy, userID       *string
		expiresAt, createdAt          time.Time
		consumedAt                    *time.Time
	)

	if err := row.Scan(&id, &purpose, &tokenHash, &email, &role, &invitedBy, &userID, &expiresAt, &consumedAt, &createdAt); err != nil {
		return nil, err
	}

	return domain.RehydrateVerificationToken(
		id,
		domain.TokenPurpose(purpose),
		tokenHash,
		email,
		domain.Role(deref(role)),
		deref(invitedBy),
		deref(userID),
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
