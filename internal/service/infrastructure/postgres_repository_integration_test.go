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

	content := json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)

	svc, err := domain.NewService(
		"Integration Test Service", "integration-test-service-xyz",
		"",
		[]string{"moi", "hot"}, "", nil, nil, content,
		nil, nil, nil, true, 0,
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

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected to find the created service")
	}
	if fetched.Line() != domain.LineDichVu {
		t.Errorf("expected default line %q, got %q", domain.LineDichVu, fetched.Line())
	}

	newTitle := "Updated Title"
	if err := fetched.Update(domain.UpdateServiceInput{Title: &newTitle}); err != nil {
		t.Fatalf("Update failed: %v", err)
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

// TestPostgresServiceRepository_Categories verifies SetCategories replaces the links
// and that the read queries return them.
func TestPostgresServiceRepository_Categories(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 2")
	if err != nil {
		t.Fatalf("query categories: %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < 2 {
		t.Skip("need two categories to test linking")
	}

	repo := NewPostgresServiceRepository(pool)
	svc, err := domain.NewService("Categories Test Service", "integration-test-service-cats-xyz", "", nil, "", nil, nil, nil, nil, nil, nil, true, 0)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	created, err := repo.Create(ctx, svc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM services WHERE id = $1", created.ID()) }()

	if err := repo.SetCategories(ctx, created.ID(), ids); err != nil {
		t.Fatalf("SetCategories failed: %v", err)
	}
	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil || fetched == nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(fetched.CategoryIDs) != 2 {
		t.Fatalf("expected 2 linked categories, got %v", fetched.CategoryIDs)
	}

	// Replace with one, then clear.
	if err := repo.SetCategories(ctx, created.ID(), ids[:1]); err != nil {
		t.Fatalf("SetCategories (replace) failed: %v", err)
	}
	fetched, _ = repo.GetByID(ctx, created.ID())
	if len(fetched.CategoryIDs) != 1 || fetched.CategoryIDs[0] != ids[0] {
		t.Fatalf("expected the single replacement link, got %v", fetched.CategoryIDs)
	}
	if err := repo.SetCategories(ctx, created.ID(), nil); err != nil {
		t.Fatalf("SetCategories (clear) failed: %v", err)
	}
	fetched, _ = repo.GetByID(ctx, created.ID())
	if len(fetched.CategoryIDs) != 0 {
		t.Fatalf("expected links cleared, got %v", fetched.CategoryIDs)
	}
}
