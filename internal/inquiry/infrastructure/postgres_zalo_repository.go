package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// PostgresZaloFollowerRepository implements domain.ZaloFollowerRepository.
type PostgresZaloFollowerRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ZaloFollowerRepository = (*PostgresZaloFollowerRepository)(nil)

func NewPostgresZaloFollowerRepository(pool *pgxpool.Pool) *PostgresZaloFollowerRepository {
	return &PostgresZaloFollowerRepository{pool: pool}
}

func (r *PostgresZaloFollowerRepository) Upsert(ctx context.Context, follower *domain.ZaloOAFollower) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO zalo_oa_followers (zalo_user_id, display_name, is_active)
		VALUES ($1, $2, true)
		ON CONFLICT (zalo_user_id) DO UPDATE
		SET is_active = true, display_name = COALESCE($2, zalo_oa_followers.display_name), updated_at = now()
	`, follower.ZaloUserID(), follower.DisplayName())
	if err != nil {
		return fmt.Errorf("zalo follower repository upsert: %w", err)
	}
	return nil
}

func (r *PostgresZaloFollowerRepository) SetActive(ctx context.Context, zaloUserID string, isActive bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE zalo_oa_followers SET is_active = $1, updated_at = now() WHERE zalo_user_id = $2
	`, isActive, zaloUserID)
	if err != nil {
		return fmt.Errorf("zalo follower repository setActive: %w", err)
	}
	return nil
}

func (r *PostgresZaloFollowerRepository) GetAllActive(ctx context.Context) ([]*domain.ZaloOAFollower, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, zalo_user_id, display_name, is_active, followed_at
		FROM zalo_oa_followers WHERE is_active = true
	`)
	if err != nil {
		return nil, fmt.Errorf("zalo follower repository getAllActive: %w", err)
	}
	defer rows.Close()

	var followers []*domain.ZaloOAFollower
	for rows.Next() {
		var (
			id, zaloUserID string
			displayName    *string
			isActive       bool
			followedAt     time.Time
		)
		if err := rows.Scan(&id, &zaloUserID, &displayName, &isActive, &followedAt); err != nil {
			return nil, fmt.Errorf("zalo follower repository getAllActive scan: %w", err)
		}
		followers = append(followers, domain.RehydrateZaloOAFollower(id, zaloUserID, displayName, isActive, followedAt))
	}
	return followers, rows.Err()
}

// ZaloTokenRepository implementation.

type PostgresZaloTokenRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ZaloTokenRepository = (*PostgresZaloTokenRepository)(nil)

func NewPostgresZaloTokenRepository(pool *pgxpool.Pool) *PostgresZaloTokenRepository {
	return &PostgresZaloTokenRepository{pool: pool}
}

func (r *PostgresZaloTokenRepository) GetCurrent(ctx context.Context) (*domain.ZaloOAToken, error) {
	var accessToken, refreshToken string
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT access_token, refresh_token, expires_at FROM zalo_oa_tokens WHERE id = 1`).
		Scan(&accessToken, &refreshToken, &expiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("zalo token repository getCurrent: %w", err)
	}
	return domain.NewZaloOAToken(accessToken, refreshToken, expiresAt), nil
}

func (r *PostgresZaloTokenRepository) Save(ctx context.Context, token *domain.ZaloOAToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO zalo_oa_tokens (id, access_token, refresh_token, expires_at, updated_at)
		VALUES (1, $1, $2, $3, now())
		ON CONFLICT (id) DO UPDATE
		SET access_token = $1, refresh_token = $2, expires_at = $3, updated_at = now()
	`, token.AccessToken(), token.RefreshToken(), token.ExpiresAt())
	if err != nil {
		return fmt.Errorf("zalo token repository save: %w", err)
	}
	return nil
}
