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
		nil,
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

	// GetByID goes through the LEFT JOIN path — group should be nil since
	// none was set.
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

// TestPostgresServiceRepository_JoinsGroup verifies the LEFT JOIN actually
// resolves the real group name when the FK is set.
func TestPostgresServiceRepository_JoinsGroup(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var groupID string
	if err := pool.QueryRow(ctx, "SELECT id FROM service_groups LIMIT 1").Scan(&groupID); err != nil {
		t.Skipf("no service_groups row available to test join: %v", err)
	}

	repo := NewPostgresServiceRepository(pool)

	svc, err := domain.NewService(
		"Join Test Service", "integration-test-service-join-xyz",
		"",
		&groupID,
		nil, "", nil, nil, nil,
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

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Group == nil || fetched.Group.ID != groupID {
		t.Errorf("expected Group to be joined with id %s, got %+v", groupID, fetched.Group)
	}
}
