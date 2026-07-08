package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project/domain"
)

func TestCreateProject(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	p, err := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	_, err := CreateProject(ctx, repo, domain.CreateProjectInput{Title: "", Slug: ""})
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

func TestCreateProject_StoresRelations(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	p, err := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a",
		Categories: []domain.CategoryCondition{{CategoryID: "cat-1", Condition: "new"}},
		ServiceIDs: []string{"svc-1", "svc-2"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, _ := GetProjectByID(ctx, repo, p.ID())
	if len(got.Categories) != 1 || got.Categories[0].ID != "cat-1" {
		t.Errorf("expected 1 category cat-1, got %+v", got.Categories)
	}
	if len(got.Services) != 2 {
		t.Errorf("expected 2 services, got %+v", got.Services)
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	_, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateProject_PartialUpdate(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	created, _ := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a", OrderIndex: 5,
	})

	newTitle := "Nha may A (sua)"
	updated, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{
		ID: created.ID(), Title: &newTitle,
	})
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

// TestUpdateProject_NilVsEmptyRelations exercises the exact distinction the
// old TS repository made via `input.categories !== undefined` /
// `input.serviceIds !== undefined`: omitting the field (nil pointer) must
// leave existing relations untouched, while explicitly sending an empty set
// (non-nil pointer to an empty slice) must clear them. See
// domain.UpdateProjectInput's doc comment.
func TestUpdateProject_NilVsEmptyRelations(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	created, _ := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a",
		Categories: []domain.CategoryCondition{{CategoryID: "cat-1", Condition: "new"}},
		ServiceIDs: []string{"svc-1"},
	})

	// Update with relations omitted (nil) — must leave them untouched.
	newTitle := "Nha may A (sua)"
	_, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{ID: created.ID(), Title: &newTitle})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ := GetProjectByID(ctx, repo, created.ID())
	if len(got.Categories) != 1 || len(got.Services) != 1 {
		t.Fatalf("expected relations untouched by omitted fields, got categories=%+v services=%+v", got.Categories, got.Services)
	}

	// Update with relations explicitly emptied — must clear them.
	emptyCategories := []domain.CategoryCondition{}
	emptyServiceIDs := []string{}
	_, err = UpdateProject(ctx, repo, domain.UpdateProjectInput{
		ID: created.ID(), Categories: &emptyCategories, ServiceIDs: &emptyServiceIDs,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ = GetProjectByID(ctx, repo, created.ID())
	if len(got.Categories) != 0 || len(got.Services) != 0 {
		t.Fatalf("expected relations cleared, got categories=%+v services=%+v", got.Categories, got.Services)
	}
}

func TestDeleteAndRestoreProject(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	created, _ := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a",
	})

	if err := DeleteProject(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetProjectByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted project to not be found by GetByID")
	}

	if err := RestoreProject(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetProjectByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored project to be found again")
	}
}

func TestToggleAndReorderProject(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	created, _ := CreateProject(ctx, repo, domain.CreateProjectInput{
		Title: "Nha may A", Slug: "nha-may-a",
	})

	if err := ToggleProjectPublish(ctx, repo, created.ID(), true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := ToggleProjectFeatured(ctx, repo, created.ID(), true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := UpdateProjectOrder(ctx, repo, created.ID(), 42); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, _ := GetProjectByID(ctx, repo, created.ID())
	if !got.IsPublished() || !got.IsFeatured() || got.OrderIndex() != 42 {
		t.Errorf("expected published/featured/orderIndex updated, got %v %v %d", got.IsPublished(), got.IsFeatured(), got.OrderIndex())
	}
}
