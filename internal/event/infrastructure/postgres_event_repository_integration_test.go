//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/event/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/event/infrastructure/...
// Not part of the normal `go test ./...` run — hits the real Supabase
// Postgres (see ARCHITECTURE.md section 10/17).
func TestPostgresEventRepository_CreateAndTopViewed(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresEventRepository(pool)

	entityType := domain.EntityTypeProduct
	entityID := "integration-test-product-id"
	pagePath := "/san-pham/integration-test"
	sessionID := "integration-test-session"

	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM tracking_events WHERE event_label = $1", entityID); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	event, err := domain.NewEvent(domain.EventViewItem, &entityType, &entityID, &pagePath, &sessionID)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	result, err := repo.TopViewed(ctx, domain.TopViewedFilter{
		EntityType: domain.EntityTypeProduct,
		Since:      time.Now().Add(-time.Hour),
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("TopViewed failed: %v", err)
	}

	found := false
	for _, row := range result {
		if row.EntityID == entityID {
			found = true
			if row.Count < 1 {
				t.Errorf("expected count >= 1 for %s, got %d", entityID, row.Count)
			}
		}
	}
	if !found {
		t.Errorf("expected %s to appear in TopViewed results, got %+v", entityID, result)
	}
}
