package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func TestDeleteContact(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()
	repo.contacts["id-1"] = domain.RehydrateContact("id-1", "phone", nil, "0901234567", true, 0)

	if err := DeleteContact(ctx, repo, "id-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := repo.contacts["id-1"]; ok {
		t.Error("expected contact to be removed from repository")
	}
}
