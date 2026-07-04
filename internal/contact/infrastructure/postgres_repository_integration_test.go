//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/contact/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/contact/infrastructure/...
// Not part of the normal `go test ./...` run — this hits the real Supabase
// Postgres, so it must never run implicitly (see ARCHITECTURE.md section 10/17).
func TestPostgresContactRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresContactRepository(pool)

	label := "integration-test"
	contact, err := domain.NewContact("phone", &label, "0900000000-test", true, 999)
	if err != nil {
		t.Fatalf("NewContact failed: %v", err)
	}

	created, err := repo.Create(ctx, contact)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// Always clean up, even if a later assertion fails.
	defer func() {
		if err := repo.Delete(ctx, created.ID()); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	if created.ID() == "" {
		t.Error("expected created contact to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Value() != "0900000000-test" {
		t.Errorf("expected fetched contact to match created one, got %+v", fetched)
	}

	newValue := "0911111111-test"
	if err := fetched.UpdateValue(newValue); err != nil {
		t.Fatalf("UpdateValue failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Value() != newValue {
		t.Errorf("expected updated value %s, got %s", newValue, updated.Value())
	}
}
