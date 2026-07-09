//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/product/domain"
)

// Run explicitly with: go test -tags=integration ./internal/product/infrastructure/...
func TestPostgresProductLineRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var brandID string
	if err := pool.QueryRow(ctx, "SELECT id FROM brands LIMIT 1").Scan(&brandID); err != nil {
		t.Skipf("no brand row available to test against: %v", err)
	}

	repo := NewPostgresProductLineRepository(pool)

	line, err := domain.NewProductLine(&brandID, nil, "FTKZ-TEST", "Dòng Inverter siêu cao cấp", 4, nil, []string{"FTKZ"})
	if err != nil {
		t.Fatalf("NewProductLine failed: %v", err)
	}

	created, err := repo.Create(ctx, line)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM product_lines WHERE id = $1", created.ID())
	}()

	if created.Code() != "FTKZ-TEST" || created.TierRank() != 4 {
		t.Errorf("unexpected created line: %+v", created)
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Name() != "Dòng Inverter siêu cao cấp" {
		t.Fatalf("expected fetched line to match, got %+v", fetched)
	}

	if err := fetched.Update(nil, "Dòng Inverter siêu cao cấp (updated)", 5, nil, []string{"FTKZ", "FTKM"}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("repo Update failed: %v", err)
	}
	if updated.TierRank() != 5 {
		t.Errorf("expected tier_rank 5, got %d", updated.TierRank())
	}

	lines, err := repo.List(ctx, &brandID, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	found := false
	for _, l := range lines {
		if l.ID() == created.ID() {
			found = true
		}
	}
	if !found {
		t.Error("expected List(brandID) to include the created line")
	}

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted line to not be found by GetByID")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored line to be found again")
	}
}
