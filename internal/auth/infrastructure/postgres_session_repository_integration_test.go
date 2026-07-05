//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

func TestPostgresSessionRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	// t.Cleanup, not defer — see comment in
	// postgres_verification_token_repository_integration_test.go's
	// mustSeedUserID caller for why this matters.
	t.Cleanup(pool.Close)

	repo := NewPostgresSessionRepository(pool)
	userID := mustSeedUserID(ctx, t, pool)

	session, _, err := domain.NewSession(userID, "integration-test-agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	created, err := repo.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", created.ID()); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	fetched, err := repo.GetByTokenHash(ctx, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByTokenHash failed: %v", err)
	}
	if fetched == nil || fetched.UserID() != userID || !fetched.IsValid() {
		t.Errorf("expected fetched session to match created one, got %+v", fetched)
	}

	if err := repo.Revoke(ctx, created.ID()); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	afterRevoke, err := repo.GetByTokenHash(ctx, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByTokenHash after revoke failed: %v", err)
	}
	if afterRevoke.IsValid() {
		t.Error("expected session to be invalid after revoke")
	}
}

func TestPostgresSessionRepository_RevokeAllByUserID(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	// t.Cleanup, not defer — see comment in
	// postgres_verification_token_repository_integration_test.go's
	// mustSeedUserID caller for why this matters.
	t.Cleanup(pool.Close)

	repo := NewPostgresSessionRepository(pool)
	userID := mustSeedUserID(ctx, t, pool)

	var ids []string
	for i := 0; i < 2; i++ {
		session, _, err := domain.NewSession(userID, "", "", time.Hour)
		if err != nil {
			t.Fatalf("NewSession failed: %v", err)
		}
		created, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		ids = append(ids, created.ID())
	}
	defer func() {
		for _, id := range ids {
			_, _ = pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
		}
	}()

	if err := repo.RevokeAllByUserID(ctx, userID); err != nil {
		t.Fatalf("RevokeAllByUserID failed: %v", err)
	}

	for _, id := range ids {
		var revokedAt *time.Time
		if err := pool.QueryRow(ctx, "SELECT revoked_at FROM sessions WHERE id = $1", id).Scan(&revokedAt); err != nil {
			t.Fatalf("failed to verify revocation: %v", err)
		}
		if revokedAt == nil {
			t.Errorf("expected session %s to be revoked", id)
		}
	}
}
