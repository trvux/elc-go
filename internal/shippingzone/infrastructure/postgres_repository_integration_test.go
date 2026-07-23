//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

// Run explicitly with: go test -tags=integration ./internal/shippingzone/infrastructure/...
// Requires the shippingzone migrations to already be applied (which seed the
// "an-giang" province used below).
func TestPostgresShippingZoneRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresShippingZoneRepository(pool)

	zone, err := domain.NewShippingZone(
		"Integration Test Zone", 25000, 2, 4, false,
		[]string{"an-giang"}, []string{"30292"}, // Phường Bình Đức, An Giang
	)
	if err != nil {
		t.Fatalf("NewShippingZone failed: %v", err)
	}

	created, err := repo.Create(ctx, zone)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM shipping_zones WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created zone to have an ID")
	}
	if len(created.ProvinceCodes()) != 1 || created.ProvinceCodes()[0] != "an-giang" {
		t.Errorf("expected province codes to round-trip, got %+v", created.ProvinceCodes())
	}
	if len(created.WardCodes()) != 1 || created.WardCodes()[0] != "30292" {
		t.Errorf("expected ward codes to round-trip, got %+v", created.WardCodes())
	}

	found, err := repo.FindByProvince(ctx, "an-giang")
	if err != nil {
		t.Fatalf("FindByProvince failed: %v", err)
	}
	var matched bool
	for _, z := range found {
		if z.ID() == created.ID() {
			matched = true
		}
	}
	if !matched {
		t.Errorf("expected FindByProvince(an-giang) to include the created zone, got %+v", found)
	}

	if err := created.UpdateFeeVND(30000); err != nil {
		t.Fatalf("UpdateFeeVND failed: %v", err)
	}
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.FeeVND() != 30000 {
		t.Errorf("expected fee to update, got %d", updated.FeeVND())
	}

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted zone to not be found by GetByID")
	}
}
