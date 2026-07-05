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
	user, err := domain.NewUser("itest_"+unique, "itest_"+unique+"@example.com", "hashed-value", "Test User", "0900000000", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("NewUser failed: %v", err)
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
	if byID == nil || byID.Username() != created.Username() {
		t.Errorf("expected fetched user to match created one, got %+v", byID)
	}
	if byID.Name() != "Test User" || byID.Phone() != "0900000000" {
		t.Errorf("expected name/phone to round-trip, got name=%q phone=%q", byID.Name(), byID.Phone())
	}

	byIdentifier, err := repo.GetByIdentifier(ctx, created.Email())
	if err != nil {
		t.Fatalf("GetByIdentifier(email) failed: %v", err)
	}
	if byIdentifier == nil || byIdentifier.ID() != created.ID() {
		t.Error("expected GetByIdentifier to resolve by email")
	}

	byIdentifier, err = repo.GetByIdentifier(ctx, created.Username())
	if err != nil {
		t.Fatalf("GetByIdentifier(username) failed: %v", err)
	}
	if byIdentifier == nil || byIdentifier.ID() != created.ID() {
		t.Error("expected GetByIdentifier to resolve by username")
	}

	exists, err := repo.ExistsByUsernameOrEmail(ctx, created.Username(), "someone-else@example.com")
	if err != nil {
		t.Fatalf("ExistsByUsernameOrEmail failed: %v", err)
	}
	if !exists {
		t.Error("expected ExistsByUsernameOrEmail to find the existing username")
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
