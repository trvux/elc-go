//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/service/domain"
)

// Run explicitly with: go test -tags=integration ./internal/service/infrastructure/...
func TestPostgresServiceRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresServiceRepository(pool)

	originalPrice := int64(200000)
	discountPercent := 10
	content := json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)

	svc, err := domain.NewService(
		"Integration Test Service", "integration-test-service-xyz",
		nil, nil,
		&originalPrice, &discountPercent, nil,
		[]string{"moi", "hot"}, nil, content,
		nil, nil, nil, false, true, 0,
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	created, err := repo.Create(ctx, svc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM services WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created service to have an ID")
	}
	// jsonb reformats whitespace on storage (canonicalizes), so compare
	// decoded values, not raw bytes.
	var gotContent, wantContent any
	if err := json.Unmarshal(created.Content(), &gotContent); err != nil {
		t.Fatalf("failed to decode returned content: %v", err)
	}
	if err := json.Unmarshal(content, &wantContent); err != nil {
		t.Fatalf("failed to decode expected content: %v", err)
	}
	gotJSON, _ := json.Marshal(gotContent)
	wantJSON, _ := json.Marshal(wantContent)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("expected content roundtrip, got %s want %s", gotJSON, wantJSON)
	}
	if len(created.Labels()) != 2 || created.Labels()[0] != "moi" {
		t.Errorf("expected labels roundtrip, got %v", created.Labels())
	}

	// GetByID goes through the LEFT JOIN path — group/category should both
	// be nil since none was set.
	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected to find the created service")
	}
	if fetched.Group != nil {
		t.Errorf("expected nil Group, got %+v", fetched.Group)
	}
	if fetched.Category != nil {
		t.Errorf("expected nil Category, got %+v", fetched.Category)
	}
	if *fetched.SalePrice() != 180000 {
		t.Errorf("expected sale price 180000, got %d", *fetched.SalePrice())
	}

	if err := fetched.UpdateTitle("Updated Title"); err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched.Service)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title() != "Updated Title" {
		t.Errorf("expected updated title, got %s", updated.Title())
	}

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted service to not be found by GetByID")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}

// TestPostgresServiceRepository_JoinsGroupAndCategory verifies the LEFT JOIN
// actually resolves real group/category names when the FKs are set.
func TestPostgresServiceRepository_JoinsGroupAndCategory(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var groupID, categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM service_groups LIMIT 1").Scan(&groupID); err != nil {
		t.Skipf("no service_groups row available to test join: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT id FROM categories LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no categories row available to test join: %v", err)
	}

	repo := NewPostgresServiceRepository(pool)

	svc, err := domain.NewService(
		"Join Test Service", "integration-test-service-join-xyz",
		&groupID, &categoryID,
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, false, true, 0,
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	created, err := repo.Create(ctx, svc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM services WHERE id = $1", created.ID())
	}()

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Group == nil || fetched.Group.ID != groupID {
		t.Errorf("expected Group to be joined with id %s, got %+v", groupID, fetched.Group)
	}
	if fetched.Category == nil || fetched.Category.ID != categoryID {
		t.Errorf("expected Category to be joined with id %s, got %+v", categoryID, fetched.Category)
	}
}
