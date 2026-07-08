package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestChangePassword(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "OldPassw0rd!", domain.RoleUser)
	session, _, err := domain.NewSession(user.ID(), "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	activeSession, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := ChangePassword(ctx, userRepo, sessionRepo, hasher, user, "OldPassw0rd!", "NewPassw0rd!"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := userRepo.GetByID(ctx, user.ID())
	if err := hasher.Compare(updated.PasswordHash(), "NewPassw0rd!"); err != nil {
		t.Error("expected password hash to have been updated")
	}
	if sessionRepo.sessions[activeSession.ID()].IsValid() {
		t.Error("expected all sessions to be revoked after a password change")
	}
}

func TestChangePassword_WrongCurrentPasswordRejected(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "OldPassw0rd!", domain.RoleUser)

	if err := ChangePassword(ctx, userRepo, sessionRepo, hasher, user, "WrongPassword!", "NewPassw0rd!"); err == nil {
		t.Fatal("expected error when current password is wrong")
	}
}

func TestChangePassword_WeakNewPasswordRejected(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "OldPassw0rd!", domain.RoleUser)

	if err := ChangePassword(ctx, userRepo, sessionRepo, hasher, user, "OldPassw0rd!", "weak"); err == nil {
		t.Fatal("expected weak new password to be rejected")
	}
}
