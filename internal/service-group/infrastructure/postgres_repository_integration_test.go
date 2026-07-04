//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/service-group/domain"
)

// Run explicitly with: go test -tags=integration ./internal/service-group/infrastructure/...
func TestPostgresServiceGroupRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresServiceGroupRepository(pool)

	sg, err := domain.NewServiceGroup(
		"Integration Test Group", "integration-test-group-xyz",
		nil, nil, nil, false, 999, []string{},
	)
	if err != nil {
		t.Fatalf("NewServiceGroup failed: %v", err)
	}

	created, err := repo.Create(ctx, sg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM service_groups WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created group to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-group-xyz" {
		t.Errorf("expected fetched group to match created one, got %+v", fetched)
	}

	if err := fetched.UpdateName("Integration Test Group Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Group Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	// Soft delete then verify it's excluded from GetByID.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted group to not be found by GetByID")
	}

	// Resurrect rule: creating again with the same slug should reuse the
	// soft-deleted row (same ID), not fail on the unique constraint.
	resurrectInput, err := domain.NewServiceGroup(
		"Resurrected Name", "integration-test-group-xyz",
		nil, nil, nil, false, 1, []string{},
	)
	if err != nil {
		t.Fatalf("NewServiceGroup (resurrect) failed: %v", err)
	}
	resurrected, err := repo.Create(ctx, resurrectInput)
	if err != nil {
		t.Fatalf("Create (resurrect) failed: %v", err)
	}
	if resurrected.ID() != created.ID() {
		t.Errorf("expected resurrect to reuse id %s, got %s", created.ID(), resurrected.ID())
	}
	if resurrected.Name() != "Resurrected Name" {
		t.Errorf("expected resurrected name, got %s", resurrected.Name())
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}
