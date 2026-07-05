package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/category/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestCreateCategory(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	c, err := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "May lanh treo tuong", Slug: "may-lanh-treo-tuong"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateCategory_ValidationError(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	_, err := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "", Slug: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	_, err := UpdateCategory(ctx, repo, domain.UpdateCategoryInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateCategory_PartialUpdate(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	created, _ := CreateCategory(ctx, repo, domain.CreateCategoryInput{
		Name: "May lanh treo tuong", Slug: "may-lanh-treo-tuong", OrderIndex: 5,
	})

	newName := "May lanh treo tuong Updated"
	updated, err := UpdateCategory(ctx, repo, domain.UpdateCategoryInput{
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

func TestUpdateCategory_GroupID(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	created, _ := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "May lanh", Slug: "may-lanh"})

	groupID := "group-1"
	updated, err := UpdateCategory(ctx, repo, domain.UpdateCategoryInput{ID: created.ID(), GroupID: &groupID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.GroupID() == nil || *updated.GroupID() != groupID {
		t.Errorf("expected groupID to be set, got %+v", updated.GroupID())
	}
}

func TestDeleteAndRestoreCategory(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	created, _ := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "May lanh treo tuong", Slug: "may-lanh-treo-tuong"})

	if err := DeleteCategory(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetCategoryByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted category to not be found by GetByID")
	}

	if err := RestoreCategory(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetCategoryByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored category to be found again")
	}
}

func TestCreateCategory_Resurrect(t *testing.T) {
	repo := newFakeCategoryRepository()
	ctx := context.Background()

	created, _ := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "May lanh treo tuong", Slug: "may-lanh-treo-tuong"})
	_ = DeleteCategory(ctx, repo, created.ID())

	// Create a new category with the same slug. It should resurrect the old one.
	resurrected, err := CreateCategory(ctx, repo, domain.CreateCategoryInput{Name: "May lanh treo tuong Moi", Slug: "may-lanh-treo-tuong"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resurrected.ID() != created.ID() {
		t.Errorf("expected resurrected category to have ID %s, got %s", created.ID(), resurrected.ID())
	}
	if resurrected.Name() != "May lanh treo tuong Moi" {
		t.Errorf("expected resurrected category to have updated name, got %s", resurrected.Name())
	}
	if resurrected.IsDeleted() {
		t.Error("expected resurrected category to not be deleted")
	}
}
