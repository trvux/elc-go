package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/brand/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestCreateBrand(t *testing.T) {
	repo := newFakeBrandRepository()
	ctx := context.Background()

	b, err := CreateBrand(ctx, repo, domain.CreateBrandInput{Name: "Apple", Slug: "apple"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateBrand_ValidationError(t *testing.T) {
	repo := newFakeBrandRepository()
	ctx := context.Background()

	_, err := CreateBrand(ctx, repo, domain.CreateBrandInput{Name: "", Slug: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateBrand_NotFound(t *testing.T) {
	repo := newFakeBrandRepository()
	ctx := context.Background()

	_, err := UpdateBrand(ctx, repo, domain.UpdateBrandInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateBrand_PartialUpdate(t *testing.T) {
	repo := newFakeBrandRepository()
	ctx := context.Background()

	created, _ := CreateBrand(ctx, repo, domain.CreateBrandInput{
		Name: "Apple", Slug: "apple", OrderIndex: 5,
	})

	newName := "Apple Inc"
	updated, err := UpdateBrand(ctx, repo, domain.UpdateBrandInput{
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

func TestDeleteAndRestoreBrand(t *testing.T) {
	repo := newFakeBrandRepository()
	ctx := context.Background()

	created, _ := CreateBrand(ctx, repo, domain.CreateBrandInput{Name: "Apple", Slug: "apple"})

	if err := DeleteBrand(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetBrandByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted brand to not be found by GetByID")
	}

	if err := RestoreBrand(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetBrandByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored brand to be found again")
	}
}
