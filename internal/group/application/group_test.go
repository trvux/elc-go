package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/group/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestCreateGroup(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	g, err := CreateGroup(ctx, repo, domain.CreateGroupInput{Name: "Air Conditioners", Slug: "air-conditioners"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateGroup_ValidationError(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	_, err := CreateGroup(ctx, repo, domain.CreateGroupInput{Name: "", Slug: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateGroup_NotFound(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	_, err := UpdateGroup(ctx, repo, domain.UpdateGroupInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateGroup_PartialUpdate(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	created, _ := CreateGroup(ctx, repo, domain.CreateGroupInput{
		Name: "Air Conditioners", Slug: "air-conditioners", OrderIndex: 5,
	})

	newName := "Air Conditioners Updated"
	updated, err := UpdateGroup(ctx, repo, domain.UpdateGroupInput{
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

func TestDeleteAndRestoreGroup(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	created, _ := CreateGroup(ctx, repo, domain.CreateGroupInput{Name: "Air Conditioners", Slug: "air-conditioners"})

	if err := DeleteGroup(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetGroupByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted group to not be found by GetByID")
	}

	if err := RestoreGroup(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetGroupByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored group to be found again")
	}
}

func TestCreateGroup_Resurrect(t *testing.T) {
	repo := newFakeGroupRepository()
	ctx := context.Background()

	created, _ := CreateGroup(ctx, repo, domain.CreateGroupInput{Name: "Air Conditioners", Slug: "air-conditioners"})
	_ = DeleteGroup(ctx, repo, created.ID())

	// Create a new group with the same slug. It should resurrect the old one.
	resurrected, err := CreateGroup(ctx, repo, domain.CreateGroupInput{Name: "Air Conditioners New", Slug: "air-conditioners"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resurrected.ID() != created.ID() {
		t.Errorf("expected resurrected group to have ID %s, got %s", created.ID(), resurrected.ID())
	}
	if resurrected.Name() != "Air Conditioners New" {
		t.Errorf("expected resurrected group to have updated name, got %s", resurrected.Name())
	}
	if resurrected.IsDeleted() {
		t.Error("expected resurrected group to not be deleted")
	}
}
