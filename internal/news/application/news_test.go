package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/news/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestCreateNews(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	n, err := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateNews_ValidationError(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	_, err := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "", Slug: "Invalid Slug"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["title"]) == 0 || len(appErr.Fields["slug"]) == 0 {
		t.Errorf("expected title/slug field errors, got %v", appErr.Fields)
	}
}

func TestUpdateNews_NotFound(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	_, err := UpdateNews(ctx, repo, domain.UpdateNewsInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateNews_PartialUpdate(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	created, _ := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a", OrderIndex: 5})

	newTitle := "Tin tuc A (sua)"
	updated, err := UpdateNews(ctx, repo, domain.UpdateNewsInput{ID: created.ID(), Title: &newTitle})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title() != newTitle {
		t.Errorf("expected title updated, got %s", updated.Title())
	}
	// OrderIndex wasn't in the input, must stay unchanged.
	if updated.OrderIndex() != 5 {
		t.Errorf("expected orderIndex to remain 5, got %d", updated.OrderIndex())
	}
}

func TestDeleteAndRestoreNews(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	created, _ := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a"})

	if err := DeleteNews(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetNewsByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted news to not be found by GetByID")
	}

	if err := RestoreNews(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetNewsByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored news to be found again")
	}
}

func TestCreateNews_TitleAlignDefaultsToCenter(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	n, err := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.TitleAlign() != domain.TitleAlignCenter {
		t.Errorf("expected titleAlign to default to %q, got %q", domain.TitleAlignCenter, n.TitleAlign())
	}
}

func TestCreateNews_InvalidTitleAlign(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	_, err := CreateNews(ctx, repo, domain.CreateNewsInput{
		Title: "Tin tuc A", Slug: "tin-tuc-a", TitleAlign: "diagonal",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdateNews_TitleAlign(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	created, _ := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a"})

	center := domain.TitleAlignCenter
	updated, err := UpdateNews(ctx, repo, domain.UpdateNewsInput{ID: created.ID(), TitleAlign: &center})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.TitleAlign() != domain.TitleAlignCenter {
		t.Errorf("expected titleAlign %q, got %q", domain.TitleAlignCenter, updated.TitleAlign())
	}
}

func TestUpdateNews_InvalidTitleAlign(t *testing.T) {
	repo := newFakeNewsRepository()
	ctx := context.Background()

	created, _ := CreateNews(ctx, repo, domain.CreateNewsInput{Title: "Tin tuc A", Slug: "tin-tuc-a"})

	invalid := "diagonal"
	_, err := UpdateNews(ctx, repo, domain.UpdateNewsInput{ID: created.ID(), TitleAlign: &invalid})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}
