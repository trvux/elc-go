package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestAcceptInvite(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	token, raw, err := domain.NewInviteToken("new.admin@example.com", domain.RoleAdmin, "inviter-id", InviteTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	user, err := AcceptInvite(ctx, userRepo, tokenRepo, hasher, AcceptInviteInput{
		Token:    raw,
		Username: "newadmin",
		Password: "Vlu15112002@",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Username() != "newadmin" || user.Email() != "new.admin@example.com" {
		t.Errorf("unexpected user: %+v", user)
	}
	if user.Role() != domain.RoleAdmin {
		t.Errorf("expected role from invite to carry over, got %s", user.Role())
	}
}

func TestAcceptInvite_InvalidToken(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	_, err := AcceptInvite(ctx, userRepo, tokenRepo, hasher, AcceptInviteInput{
		Token:    "not-a-real-token",
		Username: "stranger",
		Password: "Vlu15112002@",
	})
	if err == nil {
		t.Fatal("expected error for an invite token that was never issued")
	}
}

func TestAcceptInvite_TokenCannotBeReused(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	token, raw, err := domain.NewInviteToken("new.admin@example.com", domain.RoleAdmin, "inviter-id", InviteTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := AcceptInvite(ctx, userRepo, tokenRepo, hasher, AcceptInviteInput{
		Token: raw, Username: "newadmin", Password: "Vlu15112002@",
	}); err != nil {
		t.Fatalf("expected first accept to succeed, got %v", err)
	}

	if _, err := AcceptInvite(ctx, userRepo, tokenRepo, hasher, AcceptInviteInput{
		Token: raw, Username: "otheruser", Password: "Vlu15112002@",
	}); err == nil {
		t.Fatal("expected second accept of the same token to fail")
	}
}

func TestAcceptInvite_WeakPasswordRejected(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	token, raw, err := domain.NewInviteToken("new.admin@example.com", domain.RoleAdmin, "inviter-id", InviteTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := AcceptInvite(ctx, userRepo, tokenRepo, hasher, AcceptInviteInput{
		Token: raw, Username: "newadmin", Password: "weak",
	}); err == nil {
		t.Fatal("expected weak password to be rejected")
	}
}
