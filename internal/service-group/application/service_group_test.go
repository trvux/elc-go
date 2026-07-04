package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/service-group/domain"
)

func TestCreateServiceGroup(t *testing.T) {
	repo := newFakeServiceGroupRepository()
	ctx := context.Background()

	sg, err := CreateServiceGroup(ctx, repo, domain.CreateServiceGroupInput{
		Name: "May lanh", Slug: "may-lanh",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sg.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateServiceGroup_ValidationError(t *testing.T) {
	repo := newFakeServiceGroupRepository()
	ctx := context.Background()

	_, err := CreateServiceGroup(ctx, repo, domain.CreateServiceGroupInput{Name: "", Slug: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateServiceGroup_NotFound(t *testing.T) {
	repo := newFakeServiceGroupRepository()
	ctx := context.Background()

	_, err := UpdateServiceGroup(ctx, repo, domain.UpdateServiceGroupInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateServiceGroup_PartialUpdate(t *testing.T) {
	repo := newFakeServiceGroupRepository()
	ctx := context.Background()

	created, _ := CreateServiceGroup(ctx, repo, domain.CreateServiceGroupInput{
		Name: "May lanh", Slug: "may-lanh", OrderIndex: 5,
	})

	newName := "May lanh treo tuong"
	updated, err := UpdateServiceGroup(ctx, repo, domain.UpdateServiceGroupInput{
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

func TestDeleteAndRestoreServiceGroup(t *testing.T) {
	repo := newFakeServiceGroupRepository()
	ctx := context.Background()

	created, _ := CreateServiceGroup(ctx, repo, domain.CreateServiceGroupInput{
		Name: "May lanh", Slug: "may-lanh",
	})

	if err := DeleteServiceGroup(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceGroupByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted group to not be found by GetByID")
	}

	if err := RestoreServiceGroup(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceGroupByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored group to be found again")
	}
}
