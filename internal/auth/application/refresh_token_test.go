package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestRefreshToken_Rotates(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)
	session, raw, err := domain.NewSession(user.ID(), "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	created, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := RefreshToken(ctx, userRepo, sessionRepo, issuer, raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.RefreshToken == raw {
		t.Error("expected a newly rotated refresh token, not the same one")
	}
	if sessionRepo.sessions[created.ID()].IsValid() {
		t.Error("expected old session to be revoked after rotation")
	}
}

func TestRefreshToken_RejectsRevokedSession(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	user := seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)
	session, raw, err := domain.NewSession(user.ID(), "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	created, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := sessionRepo.Revoke(ctx, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := RefreshToken(ctx, userRepo, sessionRepo, issuer, raw); err == nil {
		t.Fatal("expected refresh with a revoked session to fail")
	}
}
