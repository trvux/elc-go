package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/service/domain"
)

func ptrStr(v string) *string { return &v }

func TestCreateService(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	s, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
		Line: domain.LineThiCong, BriefPrompts: []string{"Dia chi"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.ID() == "" {
		t.Error("expected an ID after create")
	}
	if s.Line() != domain.LineThiCong {
		t.Errorf("expected line thi-cong, got %q", s.Line())
	}
	if len(s.BriefPrompts()) != 1 {
		t.Errorf("expected 1 brief prompt, got %v", s.BriefPrompts())
	}
}

func TestCreateService_TitleAlignDefaultsToLeft(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	s, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.TitleAlign() != domain.TitleAlignLeft {
		t.Errorf("expected titleAlign to default to left, got %s", s.TitleAlign())
	}
}

func TestCreateService_InvalidTitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	_, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong", TitleAlign: "sideways",
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdateService_TitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})

	center := domain.TitleAlignCenter
	updated, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), TitleAlign: &center,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.TitleAlign() != domain.TitleAlignCenter {
		t.Errorf("expected titleAlign updated to center, got %s", updated.TitleAlign())
	}
}

func TestUpdateService_InvalidTitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})

	invalid := "diagonal"
	_, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), TitleAlign: &invalid,
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdateService_NotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	_, err := UpdateService(ctx, repo, domain.UpdateServiceInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateService_PartialUpdate_UnrelatedFieldsUnchanged(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh", Slug: "may-lanh", OrderIndex: 3,
	})

	newTitle := "May lanh cao cap"
	updated, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), Title: ptrStr(newTitle),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title() != newTitle {
		t.Errorf("expected title updated, got %s", updated.Title())
	}
	if updated.OrderIndex() != 3 {
		t.Errorf("expected orderIndex to remain 3, got %d", updated.OrderIndex())
	}
}

func TestDeleteAndRestoreService(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{Title: "May lanh", Slug: "may-lanh"})

	if err := DeleteService(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted service to not be found by GetByID")
	}

	if err := RestoreService(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored service to be found again")
	}
}

func TestCreateService_LinksCategories(t *testing.T) {
	repo := newFakeServiceRepository()
	created, err := CreateService(context.Background(), repo, domain.CreateServiceInput{
		Title: "Lap dat may lanh", Slug: "lap-dat-may-lanh", CategoryIDs: []string{"cat-1", "cat-2"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := repo.categories[created.ID()]; len(got) != 2 || got[0] != "cat-1" {
		t.Errorf("expected both categories linked, got %v", got)
	}
}

func TestCreateService_NoCategoriesMakesNoLinkCall(t *testing.T) {
	repo := newFakeServiceRepository()
	if _, err := CreateService(context.Background(), repo, domain.CreateServiceInput{Title: "X", Slug: "x"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.setCalls != 0 {
		t.Errorf("expected no SetCategories call, got %d", repo.setCalls)
	}
}

func TestUpdateService_CategoriesNilLeavesLinksAlone(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()
	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{Title: "X", Slug: "x", CategoryIDs: []string{"cat-1"}})
	calls := repo.setCalls

	title := "Y"
	if _, err := UpdateService(ctx, repo, domain.UpdateServiceInput{ID: created.ID(), Title: &title}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.setCalls != calls {
		t.Errorf("an update without category_ids must not touch the links")
	}
}

func TestUpdateService_EmptyCategoriesClearsLinks(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()
	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{Title: "X", Slug: "x", CategoryIDs: []string{"cat-1"}})

	if _, err := UpdateService(ctx, repo, domain.UpdateServiceInput{ID: created.ID(), CategoryIDs: []string{}}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := repo.categories[created.ID()]; len(got) != 0 {
		t.Errorf("expected links cleared, got %v", got)
	}
}
