//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Exercises the export filters (date range, selected ids) against real
// Postgres: the ::uuid[] cast and timestamptz comparisons are the parts a
// unit test cannot cover. Run with -tags=integration and DATABASE_URL pointing
// at a scratch database, never production.
func TestPostgresInquiryRepository_ExportFilters(t *testing.T) {
	ctx := context.Background()
	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()
	repo := NewPostgresInquiryRepository(pool)

	all, err := repo.GetAll(ctx, domain.InquiryFilter{})
	if err != nil || len(all) < 3 {
		t.Fatalf("need at least 3 rows in the scratch DB, got %d (err %v)", len(all), err)
	}

	// Selected ids: exactly those rows, nothing else.
	picked := []string{all[0].ID(), all[2].ID()}
	got, err := repo.GetAll(ctx, domain.InquiryFilter{IDs: picked})
	if err != nil {
		t.Fatalf("ids filter: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ids filter returned %d rows, want 2", len(got))
	}

	// Date range: [From, To) must keep exactly the rows created inside it.
	from := all[1].CreatedAt()
	to := all[0].CreatedAt()
	want := 0
	for _, i := range all {
		if !i.CreatedAt().Before(from) && i.CreatedAt().Before(to) {
			want++
		}
	}
	got, err = repo.GetAll(ctx, domain.InquiryFilter{From: from, To: to})
	if err != nil {
		t.Fatalf("date filter: %v", err)
	}
	if len(got) != want {
		t.Errorf("date filter returned %d rows, want %d", len(got), want)
	}
	n, err := repo.Count(ctx, domain.InquiryFilter{From: from, To: to})
	if err != nil || n != want {
		t.Errorf("Count = %d (err %v), want %d", n, err, want)
	}

	// An empty window returns nothing, not everything.
	got, err = repo.GetAll(ctx, domain.InquiryFilter{From: time.Now().Add(time.Hour)})
	if err != nil || len(got) != 0 {
		t.Errorf("future window returned %d rows (err %v), want 0", len(got), err)
	}
}
