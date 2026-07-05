package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestResetPassword(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "OldPassw0rd!", domain.RoleSuperAdmin)
	session, _, err := domain.NewSession(user.ID(), "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	activeSession, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	token, raw, err := domain.NewPasswordResetToken(user.ID(), user.Email(), PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := ResetPassword(ctx, userRepo, tokenRepo, sessionRepo, hasher, raw, "NewPassw0rd!"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := userRepo.GetByID(ctx, user.ID())
	if err := hasher.Compare(updated.PasswordHash(), "NewPassw0rd!"); err != nil {
		t.Error("expected password hash to have been updated")
	}
	if sessionRepo.sessions[activeSession.ID()].IsValid() {
		t.Error("expected all sessions to be revoked after password reset")
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}

	err := ResetPassword(context.Background(), userRepo, tokenRepo, sessionRepo, hasher, "never-issued", "NewPassw0rd!")
	if err == nil {
		t.Fatal("expected error for a reset token that was never issued")
	}
}

func TestResetPassword_WeakPasswordRejected(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "OldPassw0rd!", domain.RoleSuperAdmin)
	token, raw, err := domain.NewPasswordResetToken(user.ID(), user.Email(), PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := ResetPassword(ctx, userRepo, tokenRepo, sessionRepo, hasher, raw, "weak"); err == nil {
		t.Fatal("expected weak password to be rejected")
	}
}
