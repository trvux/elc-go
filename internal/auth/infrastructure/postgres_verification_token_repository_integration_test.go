//go:build integration

package infrastructure

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

func TestPostgresVerificationTokenRepository_MagicLinkRoundTrip(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresVerificationTokenRepository(pool)

	unique := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "itest_" + unique + "@example.com"
	token, _, code, err := domain.NewMagicLinkToken(email, time.Hour)
	if err != nil {
		t.Fatalf("NewMagicLinkToken failed: %v", err)
	}

	created, err := repo.Create(ctx, token)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM verification_tokens WHERE id = $1", created.ID()); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	fetched, err := repo.GetByHash(ctx, domain.TokenPurposeMagicLink, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}
	if fetched == nil || fetched.Email() != email || fetched.Code() != code {
		t.Errorf("expected fetched token to match created one, got %+v", fetched)
	}
	if !fetched.IsUsable() {
		t.Error("expected freshly created token to be usable")
	}

	byEmail, err := repo.GetLatestActiveByEmail(ctx, domain.TokenPurposeMagicLink, email)
	if err != nil {
		t.Fatalf("GetLatestActiveByEmail failed: %v", err)
	}
	if byEmail == nil || byEmail.ID() != created.ID() {
		t.Error("expected GetLatestActiveByEmail to resolve the created token")
	}

	attempts, err := repo.IncrementAttempts(ctx, created.ID())
	if err != nil {
		t.Fatalf("IncrementAttempts failed: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected attempts to be 1 after first increment, got %d", attempts)
	}

	if err := repo.MarkConsumed(ctx, created.ID()); err != nil {
		t.Fatalf("MarkConsumed failed: %v", err)
	}
	afterConsume, err := repo.GetByHash(ctx, domain.TokenPurposeMagicLink, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByHash after consume failed: %v", err)
	}
	if afterConsume.IsUsable() {
		t.Error("expected token to be unusable after MarkConsumed")
	}

	stillActive, err := repo.GetLatestActiveByEmail(ctx, domain.TokenPurposeMagicLink, email)
	if err != nil {
		t.Fatalf("GetLatestActiveByEmail after consume failed: %v", err)
	}
	if stillActive != nil {
		t.Error("expected no active token for this email after consuming the only one")
	}
}

// mustSeedUserID creates a throwaway user row so FK constraints referencing
// users(id) are satisfied (also used by
// postgres_session_repository_integration_test.go), and registers its
// cleanup.
func mustSeedUserID(ctx context.Context, t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	unique := fmt.Sprintf("%d", time.Now().UnixNano())
	var id string
	row := pool.QueryRow(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, 'hash', 'admin') RETURNING id",
		"itest_inviter_"+unique, "itest_inviter_"+unique+"@example.com",
	)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("failed to seed inviter user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}
