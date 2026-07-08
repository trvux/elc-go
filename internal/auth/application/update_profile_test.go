package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestUpdateProfile_ChangeNameAndAvatar(t *testing.T) {
	userRepo := newFakeUserRepository()
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleUser)

	name := "Trần Vũ"
	avatar := "https://cdn.example.com/avatars/tranvux.webp"
	updated, err := UpdateProfile(ctx, userRepo, user, UpdateProfileInput{Name: &name, AvatarURL: &avatar})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name() != name {
		t.Errorf("expected name %q, got %q", name, updated.Name())
	}
	if updated.AvatarURL() != avatar {
		t.Errorf("expected avatar %q, got %q", avatar, updated.AvatarURL())
	}
	// Email untouched since it wasn't part of the input.
	if updated.Email() != "tranvux@example.com" {
		t.Errorf("expected email to stay unchanged, got %q", updated.Email())
	}
}

func TestUpdateProfile_ChangeEmail(t *testing.T) {
	userRepo := newFakeUserRepository()
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "old@example.com", "Vlu15112002@", domain.RoleUser)

	newEmail := "new@example.com"
	updated, err := UpdateProfile(ctx, userRepo, user, UpdateProfileInput{Email: &newEmail})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Email() != newEmail {
		t.Errorf("expected email %q, got %q", newEmail, updated.Email())
	}
}

func TestUpdateProfile_EmailAlreadyTakenByAnotherUser(t *testing.T) {
	userRepo := newFakeUserRepository()
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleUser)
	seedActiveUser(t, userRepo, "other", "taken@example.com", "Vlu15112002@", domain.RoleUser)

	taken := "taken@example.com"
	if _, err := UpdateProfile(ctx, userRepo, user, UpdateProfileInput{Email: &taken}); err == nil {
		t.Fatal("expected error when changing email to one already used by another account")
	}
}

func TestUpdateProfile_InvalidEmailRejected(t *testing.T) {
	userRepo := newFakeUserRepository()
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleUser)

	bad := "not-an-email"
	if _, err := UpdateProfile(ctx, userRepo, user, UpdateProfileInput{Email: &bad}); err == nil {
		t.Fatal("expected error for an invalid email")
	}
}
