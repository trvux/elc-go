package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func TestCreateContact(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()

	label := "Hotline"
	input := domain.CreateContactInput{
		Type:       "phone",
		Label:      &label,
		Value:      "0901234567",
		IsActive:   true,
		OrderIndex: 0,
	}

	contact, err := CreateContact(ctx, repo, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contact.ID() == "" {
		t.Error("expected contact to have an ID after create")
	}
	if contact.Value() != "0901234567" {
		t.Errorf("expected value 0901234567, got %s", contact.Value())
	}
}

func TestCreateContact_ValidationError(t *testing.T) {
	repo := newFakeContactRepository()
	ctx := context.Background()

	_, err := CreateContact(ctx, repo, domain.CreateContactInput{Type: "", Value: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
