package domain

import (
	"testing"
	"time"
)

func TestNewInviteToken(t *testing.T) {
	token, raw, err := NewInviteToken("New.Admin@Example.com", RoleAdmin, "inviter-id", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if raw == "" {
		t.Fatal("expected a raw token to be returned")
	}
	if token.TokenHash() != HashToken(raw) {
		t.Error("stored hash must match hash of the raw token")
	}
	if token.Email() != "new.admin@example.com" {
		t.Errorf("expected lowercase email, got %s", token.Email())
	}
	if !token.IsUsable() {
		t.Error("freshly created token should be usable")
	}
}

func TestNewInviteToken_ValidationError(t *testing.T) {
	if _, _, err := NewInviteToken("not-an-email", RoleAdmin, "inviter-id", time.Hour); err == nil {
		t.Error("expected validation error for bad email")
	}
	if _, _, err := NewInviteToken("a@b.com", Role("owner"), "inviter-id", time.Hour); err == nil {
		t.Error("expected validation error for bad role")
	}
	if _, _, err := NewInviteToken("a@b.com", RoleAdmin, "", time.Hour); err == nil {
		t.Error("expected validation error for missing invited_by")
	}
}

func TestVerificationToken_IsUsable(t *testing.T) {
	token, _, err := NewInviteToken("a@b.com", RoleAdmin, "inviter-id", -time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.IsUsable() {
		t.Error("expired token should not be usable")
	}

	fresh, _, err := NewInviteToken("a@b.com", RoleAdmin, "inviter-id", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// simulate consumption via rehydration, since domain has no public setter
	consumed := RehydrateVerificationToken(
		fresh.ID(), fresh.Purpose(), fresh.TokenHash(), fresh.Email(), fresh.Role(),
		fresh.InvitedBy(), fresh.UserID(), fresh.ExpiresAt(), ptrTime(time.Now()), fresh.CreatedAt(),
	)
	if consumed.IsUsable() {
		t.Error("consumed token should not be usable")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
