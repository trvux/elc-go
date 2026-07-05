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

func TestPostgresVerificationTokenRepository_InviteRoundTrip(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	// t.Cleanup, not defer: mustSeedUserID below also registers a t.Cleanup
	// to delete its row, and all t.Cleanup funcs run (LIFO) only after every
	// defer in this function has already run — a deferred pool.Close() here
	// would close the pool before that delete ever executes.
	t.Cleanup(pool.Close)

	repo := NewPostgresVerificationTokenRepository(pool)

	unique := fmt.Sprintf("%d", time.Now().UnixNano())
	token, _, err := domain.NewInviteToken("itest_"+unique+"@example.com", domain.RoleAdmin, mustSeedUserID(ctx, t, pool), time.Hour)
	if err != nil {
		t.Fatalf("NewInviteToken failed: %v", err)
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

	fetched, err := repo.GetByHash(ctx, domain.TokenPurposeInvite, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}
	if fetched == nil || fetched.Email() != created.Email() || fetched.Role() != domain.RoleAdmin {
		t.Errorf("expected fetched token to match created one, got %+v", fetched)
	}
	if !fetched.IsUsable() {
		t.Error("expected freshly created token to be usable")
	}

	if err := repo.MarkConsumed(ctx, created.ID()); err != nil {
		t.Fatalf("MarkConsumed failed: %v", err)
	}
	afterConsume, err := repo.GetByHash(ctx, domain.TokenPurposeInvite, created.TokenHash())
	if err != nil {
		t.Fatalf("GetByHash after consume failed: %v", err)
	}
	if afterConsume.IsUsable() {
		t.Error("expected token to be unusable after MarkConsumed")
	}
}

// mustSeedUserID creates a throwaway user row so invited_by's FK constraint
// is satisfied, and registers its cleanup.
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
