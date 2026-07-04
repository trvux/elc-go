package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func TestGetContacts(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()
	repo.contacts["id-1"] = domain.RehydrateContact("id-1", "phone", nil, "0901234567", true, 0)
	repo.contacts["id-2"] = domain.RehydrateContact("id-2", "email", nil, "a@b.com", true, 1)

	contacts, err := GetContacts(ctx, repo, domain.ContactFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(contacts))
	}
}

func TestGetContactByID_NotFound(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()

	contact, err := GetContactByID(ctx, repo, "missing")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contact != nil {
		t.Errorf("expected nil contact, got %v", contact)
	}
}
