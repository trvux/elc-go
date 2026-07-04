package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/contact/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestUpdateContact_NotFound(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()

	_, err := UpdateContact(ctx, repo, domain.UpdateContactInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateContact_PartialUpdate(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()
	repo.contacts["id-1"] = domain.RehydrateContact("id-1", "phone", nil, "0901234567", true, 0)

	newValue := "0909999999"
	updated, err := UpdateContact(ctx, repo, domain.UpdateContactInput{ID: "id-1", Value: &newValue})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Value() != "0909999999" {
		t.Errorf("expected value updated to 0909999999, got %s", updated.Value())
	}
	// Type was not part of the input, so it must stay unchanged.
	if updated.Type() != "phone" {
		t.Errorf("expected type to remain unchanged (phone), got %s", updated.Type())
	}
}

func TestUpdateContact_InvalidField(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()
	repo.contacts["id-1"] = domain.RehydrateContact("id-1", "phone", nil, "0901234567", true, 0)

	emptyValue := ""
	_, err := UpdateContact(ctx, repo, domain.UpdateContactInput{ID: "id-1", Value: &emptyValue})
	if err == nil {
		t.Fatal("expected validation error for empty value, got nil")
	}
}
