package domain

import (
	"testing"
	"time"
)

func TestNewMagicLinkToken(t *testing.T) {
	token, raw, code, err := NewMagicLinkToken("New.Member@Example.com", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if raw == "" {
		t.Fatal("expected a raw token to be returned")
	}
	if token.TokenHash() != HashToken(raw) {
		t.Error("stored hash must match hash of the raw token")
	}
	if token.Email() != "new.member@example.com" {
		t.Errorf("expected lowercase email, got %s", token.Email())
	}
	if len(code) != 6 {
		t.Errorf("expected a 6-digit code, got %q", code)
	}
	if token.Code() != code {
		t.Error("stored code must match the returned raw code")
	}
	if token.Purpose() != TokenPurposeMagicLink {
		t.Errorf("expected purpose magic_link, got %s", token.Purpose())
	}
	if !token.IsUsable() {
		t.Error("freshly created token should be usable")
	}
}

func TestNewMagicLinkToken_ValidationError(t *testing.T) {
	if _, _, _, err := NewMagicLinkToken("not-an-email", time.Hour); err == nil {
		t.Error("expected validation error for bad email")
	}
}

func TestVerificationToken_IsUsable(t *testing.T) {
	token, _, _, err := NewMagicLinkToken("a@b.com", -time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.IsUsable() {
		t.Error("expired token should not be usable")
	}

	fresh, _, _, err := NewMagicLinkToken("a@b.com", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// simulate consumption via rehydration, since domain has no public setter
	consumed := RehydrateVerificationToken(
		fresh.ID(), fresh.Purpose(), fresh.TokenHash(), fresh.Email(), fresh.Code(),
		fresh.Attempts(), fresh.ExpiresAt(), ptrTime(time.Now()), fresh.CreatedAt(),
	)
	if consumed.IsUsable() {
		t.Error("consumed token should not be usable")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
