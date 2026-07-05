package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestForgotPassword_KnownEmail(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	emailSender := &fakeEmailSender{}
	ctx := context.Background()

	seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	if err := ForgotPassword(ctx, userRepo, tokenRepo, emailSender, "tranvux@example.com"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(emailSender.resetsSent) != 1 {
		t.Errorf("expected exactly one reset email, got %d", len(emailSender.resetsSent))
	}
}

func TestForgotPassword_UnknownEmailIsSilentlyANoOp(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	emailSender := &fakeEmailSender{}
	ctx := context.Background()

	// Must not error and must not send an email — revealing "this email
	// doesn't exist" is exactly what this function must avoid leaking.
	if err := ForgotPassword(ctx, userRepo, tokenRepo, emailSender, "ghost@example.com"); err != nil {
		t.Fatalf("expected no error for unknown email, got %v", err)
	}
	if len(emailSender.resetsSent) != 0 {
		t.Error("expected no email to be sent for an unknown address")
	}
}
