//go:build integration

package infrastructure

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/auth/infrastructure/...
// Not part of the normal `go test ./...` run — this hits the real Supabase
// Postgres, so it must never run implicitly (see ARCHITECTURE.md section 10/17).
func TestPostgresUserRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresUserRepository(pool)

	unique := fmt.Sprintf("%d", time.Now().UnixNano())
	sub := "itest-sub-" + unique
	user, err := domain.NewOAuthUser("itest_"+unique+"@example.com", "Test User", "", &sub, domain.RoleMember)
	if err != nil {
		t.Fatalf("NewOAuthUser failed: %v", err)
	}

	created, err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", created.ID()); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	if created.ID() == "" {
		t.Error("expected created user to have an ID")
	}

	byID, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if byID == nil || byID.Username() != "" {
		t.Errorf("expected fetched user to have no username, got %+v", byID)
	}
	if byID.Name() != "Test User" {
		t.Errorf("expected name to round-trip, got name=%q", byID.Name())
	}

	byEmail, err := repo.GetByEmail(ctx, created.Email())
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if byEmail == nil || byEmail.ID() != created.ID() {
		t.Error("expected GetByEmail to resolve the created user")
	}

	byGoogleSub, err := repo.GetByGoogleSub(ctx, *user.GoogleSub())
	if err != nil {
		t.Fatalf("GetByGoogleSub failed: %v", err)
	}
	if byGoogleSub == nil || byGoogleSub.ID() != created.ID() {
		t.Error("expected GetByGoogleSub to resolve the created user")
	}

	created.RecordLogin(time.Now())
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.LastLoginAt() == nil {
		t.Error("expected last_login_at to be persisted")
	}
}
