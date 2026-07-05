package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project-type/domain"
)

func TestCreateProjectType(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	pt, err := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{
		Name: "Nha xuong", Slug: "nha-xuong",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pt.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateProjectType_ValidationError(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	_, err := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{Name: "", Slug: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["name"]) == 0 || len(appErr.Fields["slug"]) == 0 {
		t.Errorf("expected name/slug field errors, got %v", appErr.Fields)
	}
}

func TestCreateProjectType_StoresCategories(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	pt, err := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{
		Name: "Nha xuong", Slug: "nha-xuong", CategoryIDs: []string{"cat-1", "cat-2"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, _ := GetProjectTypeByID(ctx, repo, pt.ID())
	if len(got.Categories) != 2 {
		t.Errorf("expected 2 categories, got %+v", got.Categories)
	}
}

func TestUpdateProjectType_NotFound(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	_, err := UpdateProjectType(ctx, repo, domain.UpdateProjectTypeInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateProjectType_PartialUpdate(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	created, _ := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{
		Name: "Nha xuong", Slug: "nha-xuong", OrderIndex: 5,
	})

	newName := "Nha xuong (sua)"
	updated, err := UpdateProjectType(ctx, repo, domain.UpdateProjectTypeInput{
		ID: created.ID(), Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name() != newName {
		t.Errorf("expected name updated, got %s", updated.Name())
	}
	// OrderIndex wasn't in the input, must stay unchanged.
	if updated.OrderIndex() != 5 {
		t.Errorf("expected orderIndex to remain 5, got %d", updated.OrderIndex())
	}
}

// TestUpdateProjectType_NilVsEmptyCategories exercises the exact distinction
// the old TS repository made via `input.categoryIds !== undefined`: omitting
// the field (nil pointer) must leave existing relations untouched, while
// explicitly sending an empty set (non-nil pointer to an empty slice) must
// clear them. See domain.UpdateProjectTypeInput's doc comment.
func TestUpdateProjectType_NilVsEmptyCategories(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	created, _ := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{
		Name: "Nha xuong", Slug: "nha-xuong", CategoryIDs: []string{"cat-1"},
	})

	newName := "Nha xuong (sua)"
	_, err := UpdateProjectType(ctx, repo, domain.UpdateProjectTypeInput{ID: created.ID(), Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ := GetProjectTypeByID(ctx, repo, created.ID())
	if len(got.Categories) != 1 {
		t.Fatalf("expected categories untouched by omitted field, got %+v", got.Categories)
	}

	empty := []string{}
	_, err = UpdateProjectType(ctx, repo, domain.UpdateProjectTypeInput{ID: created.ID(), CategoryIDs: &empty})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ = GetProjectTypeByID(ctx, repo, created.ID())
	if len(got.Categories) != 0 {
		t.Fatalf("expected categories cleared, got %+v", got.Categories)
	}
}

func TestDeleteProjectType(t *testing.T) {
	repo := newFakeProjectTypeRepository()
	ctx := context.Background()

	created, _ := CreateProjectType(ctx, repo, domain.CreateProjectTypeInput{
		Name: "Nha xuong", Slug: "nha-xuong",
	})

	if err := DeleteProjectType(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetProjectTypeByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted project type to not be found by GetByID")
	}
}
