//go:build integration

package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/inquiry/infrastructure/...
// Not part of the normal `go test ./...` run — this hits the real Supabase
// Postgres, so it must never run implicitly (see ARCHITECTURE.md section 10/17).
func TestPostgresInquiryRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresInquiryRepository(pool)

	inquiry, err := domain.NewInquiry(
		"Integration Test", "0900000000", nil, nil, nil, nil, nil, "", nil, nil, nil,
		"", nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil,
	)
	if err != nil {
		t.Fatalf("NewInquiry failed: %v", err)
	}

	created, err := repo.Create(ctx, inquiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM inquiries WHERE id = $1", created.ID()); err != nil {
			t.Errorf("cleanup delete failed: %v", err)
		}
	}()

	if created.ID() == "" {
		t.Error("expected created inquiry to have an ID")
	}
	if created.Status() != domain.InquiryStatusNew {
		t.Errorf("expected status new, got %s", created.Status())
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Name() != "Integration Test" {
		t.Errorf("expected fetched inquiry to match created one, got %+v", fetched)
	}

	fetched.MarkContacted()
	note := "Called back"
	fetched.SetInternalNote(&note)
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Status() != domain.InquiryStatusContacted {
		t.Errorf("expected status contacted, got %s", updated.Status())
	}
	if updated.InternalNote() == nil || *updated.InternalNote() != note {
		t.Errorf("expected internal note %q, got %v", note, updated.InternalNote())
	}

	count, err := repo.Count(ctx, domain.InquiryFilter{Status: "contacted"})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected count >= 1 for status=contacted, got %d", count)
	}
}

// attachments is jsonb holding two shapes: a bare URL string (leads from before private
// files) and an object. Both must survive a write and a read, and a legacy row must stay
// exactly the string it was.
func TestPostgresInquiryRepository_AttachmentsKeepBothShapes(t *testing.T) {
	ctx := context.Background()
	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()
	repo := NewPostgresInquiryRepository(pool)

	key := "inquiries/202611/" + "0123456789abcdef0123456789abcdef" + ".dwg"
	inquiry, err := domain.NewInquiry("Attachment Shapes", "0900000000", nil, nil, nil, nil, nil, "", nil, nil,
		[]domain.Attachment{
			{URL: "https://pub.example/inquiries/old.webp"},
			{Key: key, Name: "mat bang.dwg", Size: 2048, Type: domain.TypeDWG},
		},
		"", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewInquiry failed: %v", err)
	}
	created, err := repo.Create(ctx, inquiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM inquiries WHERE id = $1", created.ID()) }()

	var raw string
	if err := pool.QueryRow(ctx, "SELECT attachments::text FROM inquiries WHERE id = $1", created.ID()).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if want := `"https://pub.example/inquiries/old.webp"`; !strings.Contains(raw, want) {
		t.Errorf("the legacy entry must stay a bare string in the database, got %s", raw)
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil || fetched == nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	got := fetched.Attachments()
	if len(got) != 2 || got[0].URL != "https://pub.example/inquiries/old.webp" || got[0].IsPrivate() {
		t.Fatalf("legacy entry read back as %#v", got)
	}
	if !got[1].IsPrivate() || got[1].Key != key || got[1].Size != 2048 || got[1].Type != domain.TypeDWG || got[1].Name != "mat bang.dwg" {
		t.Fatalf("private entry read back as %#v", got[1])
	}
}
