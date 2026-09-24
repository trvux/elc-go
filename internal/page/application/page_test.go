package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestPageUseCases(t *testing.T) {
	ctx := context.Background()
	repo := newFakePageRepository()

	content := json.RawMessage(`{"text":"test"}`)

	// 1. Create page
	input := domain.CreatePageInput{
		Title:       "Test Page",
		Slug:        "test-page",
		Content:     content,
		IsPublished: true,
	}

	created, err := CreatePage(ctx, repo, input)
	if err != nil {
		t.Fatalf("unexpected error on CreatePage: %v", err)
	}

	if created.ID() == "" || created.Title() != "Test Page" || created.Slug() != "test-page" {
		t.Errorf("unexpected created page state: %+v", created)
	}
	if created.TitleAlign() != domain.TitleAlignLeft {
		t.Errorf("expected titleAlign to default to left, got %s", created.TitleAlign())
	}

	// 2. Get pages
	filter := domain.PageFilter{
		Search: "Test",
	}
	list, err := GetPages(ctx, repo, filter)
	if err != nil {
		t.Fatalf("unexpected error on GetPages: %v", err)
	}
	if len(list) != 1 || list[0].ID() != created.ID() {
		t.Errorf("unexpected pages list: %+v", list)
	}

	// 3. Count pages
	count, err := CountPages(ctx, repo, filter)
	if err != nil {
		t.Fatalf("unexpected error on CountPages: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// 4. Update page
	updateInput := domain.UpdatePageInput{
		ID:          created.ID(),
		Title:       "Updated Page Title",
		TitleAlign:  domain.TitleAlignCenter,
		Slug:        "updated-page-slug",
		Content:     content,
		IsPublished: false,
	}

	updated, err := UpdatePage(ctx, repo, updateInput)
	if err != nil {
		t.Fatalf("unexpected error on UpdatePage: %v", err)
	}
	if updated.Title() != "Updated Page Title" || updated.Slug() != "updated-page-slug" || updated.IsPublished() != false {
		t.Errorf("unexpected updated page state: %+v", updated)
	}
	if updated.TitleAlign() != domain.TitleAlignCenter {
		t.Errorf("expected titleAlign updated to center, got %s", updated.TitleAlign())
	}

	// 5. Delete page
	err = DeletePage(ctx, repo, created.ID())
	if err != nil {
		t.Fatalf("unexpected error on DeletePage: %v", err)
	}

	// Verify no longer found in standard active query
	found, err := GetPageByID(ctx, repo, created.ID())
	if err != nil {
		t.Fatalf("unexpected error on GetPageByID: %v", err)
	}
	if found != nil {
		t.Error("expected page to be soft-deleted and not found")
	}

	// 6. Restore page
	err = RestorePage(ctx, repo, created.ID())
	if err != nil {
		t.Fatalf("unexpected error on RestorePage: %v", err)
	}

	found, err = GetPageByID(ctx, repo, created.ID())
	if err != nil {
		t.Fatalf("unexpected error on GetPageByID post-restore: %v", err)
	}
	if found == nil || found.DeletedAt() != nil {
		t.Error("expected page to be restored and active")
	}
}

func TestCreatePage_InvalidTitleAlign(t *testing.T) {
	ctx := context.Background()
	repo := newFakePageRepository()

	_, err := CreatePage(ctx, repo, domain.CreatePageInput{
		Title: "Test Page", Slug: "test-page", TitleAlign: "sideways",
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdatePage_InvalidTitleAlign(t *testing.T) {
	ctx := context.Background()
	repo := newFakePageRepository()

	created, _ := CreatePage(ctx, repo, domain.CreatePageInput{Title: "Test Page", Slug: "test-page"})

	_, err := UpdatePage(ctx, repo, domain.UpdatePageInput{
		ID: created.ID(), Title: "Test Page", Slug: "test-page", TitleAlign: "sideways",
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}
