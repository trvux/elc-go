package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestLogout(t *testing.T) {
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	session, raw, err := domain.NewSession("user-1", "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	created, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := Logout(ctx, sessionRepo, raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored := sessionRepo.sessions[created.ID()]
	if stored.IsValid() {
		t.Error("expected session to be revoked after logout")
	}
}

func TestLogout_UnknownTokenIsNotAnError(t *testing.T) {
	sessionRepo := newFakeSessionRepository()
	if err := Logout(context.Background(), sessionRepo, "never-issued"); err != nil {
		t.Fatalf("expected logout of unknown token to be a no-op, got %v", err)
	}
}
