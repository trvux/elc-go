package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/branch/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestCreateBranch(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()
	desc := json.RawMessage(`{"text": "Mieu ta"}`)

	b, err := CreateBranch(ctx, repo, domain.CreateBranchInput{
		Name:      "ELC Q1",
		Slug:      "elc-q1",
		Address:   "123 Le Loi, Q1",
		Phone:     "0901234567",
		Email:     "q1@elc.vn",
		MapsURL:   "https://maps.google.com/q1",
		MapsEmbed: "<iframe></iframe>",
		Description: desc,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateBranch_ValidationError(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()

	_, err := CreateBranch(ctx, repo, domain.CreateBranchInput{
		Name: "",
		Slug: "",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateBranch_NotFound(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()

	_, err := UpdateBranch(ctx, repo, domain.UpdateBranchInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateBranch_PartialUpdate(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()
	desc := json.RawMessage(`{"text": "Mieu ta"}`)

	created, _ := CreateBranch(ctx, repo, domain.CreateBranchInput{
		Name:      "ELC Q1",
		Slug:      "elc-q1",
		Address:   "123 Le Loi, Q1",
		Phone:     "0901234567",
		Email:     "q1@elc.vn",
		MapsURL:   "https://maps.google.com/q1",
		MapsEmbed: "<iframe></iframe>",
		Description: desc,
		OrderIndex: 5,
	})

	newName := "ELC Q1 Updated"
	updated, err := UpdateBranch(ctx, repo, domain.UpdateBranchInput{
		ID:   created.ID(),
		Name: &newName,
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

func TestDeleteBranch(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()
	desc := json.RawMessage(`{"text": "Mieu ta"}`)

	created, _ := CreateBranch(ctx, repo, domain.CreateBranchInput{
		Name:      "ELC Q1",
		Slug:      "elc-q1",
		Address:   "123 Le Loi, Q1",
		Phone:     "0901234567",
		Email:     "q1@elc.vn",
		MapsURL:   "https://maps.google.com/q1",
		MapsEmbed: "<iframe></iframe>",
		Description: desc,
	})

	if err := DeleteBranch(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetBranchByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected deleted branch to not be found by GetByID")
	}
}

func TestUpdateBranchOrder(t *testing.T) {
	repo := newFakeBranchRepository()
	ctx := context.Background()
	desc := json.RawMessage(`{"text": "Mieu ta"}`)

	created, _ := CreateBranch(ctx, repo, domain.CreateBranchInput{
		Name:      "ELC Q1",
		Slug:      "elc-q1",
		Address:   "123 Le Loi, Q1",
		Phone:     "0901234567",
		Email:     "q1@elc.vn",
		MapsURL:   "https://maps.google.com/q1",
		MapsEmbed: "<iframe></iframe>",
		Description: desc,
		OrderIndex: 1,
	})

	if err := UpdateBranchOrder(ctx, repo, created.ID(), 10); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := GetBranchByID(ctx, repo, created.ID())
	if found.OrderIndex() != 10 {
		t.Errorf("expected order index to update to 10, got %d", found.OrderIndex())
	}
}
