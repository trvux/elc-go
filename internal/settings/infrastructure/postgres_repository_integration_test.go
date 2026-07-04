//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/settings/domain"
)

func TestPostgresSettingsRepository_CRUD(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresSettingsRepository(pool)

	// Clean up initial test variables to prevent residue
	_, _ = pool.Exec(ctx, "DELETE FROM site_settings WHERE key IN ('integration_test_key_1', 'integration_test_key_2')")
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM site_settings WHERE key IN ('integration_test_key_1', 'integration_test_key_2')")
	}()

	s1, err := domain.NewSiteSetting("integration_test_key_1", "value_1")
	if err != nil {
		t.Fatalf("unexpected NewSiteSetting error: %v", err)
	}
	s2, _ := domain.NewSiteSetting("integration_test_key_2", "value_2")

	err = repo.UpdateMany(ctx, []*domain.SiteSetting{s1, s2})
	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	list, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	var found1, found2 bool
	for _, item := range list {
		if item.Key() == "integration_test_key_1" && item.Value() == "value_1" {
			found1 = true
		}
		if item.Key() == "integration_test_key_2" && item.Value() == "value_2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("could not find expected saved settings: %+v", list)
	}

	// Update existing keys (upsert check)
	s1.UpdateValue("value_1_updated")
	err = repo.UpdateMany(ctx, []*domain.SiteSetting{s1})
	if err != nil {
		t.Fatalf("UpdateMany for upsert failed: %v", err)
	}

	list, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll post-upsert failed: %v", err)
	}

	foundUpdated := false
	for _, item := range list {
		if item.Key() == "integration_test_key_1" && item.Value() == "value_1_updated" {
			foundUpdated = true
			break
		}
	}

	if !foundUpdated {
		t.Errorf("expected upserted key to have updated value, got: %+v", list)
	}
}
