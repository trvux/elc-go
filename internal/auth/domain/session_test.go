package domain

import (
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	session, raw, err := NewSession("user-id", "Mozilla/5.0", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if raw == "" {
		t.Fatal("expected a raw refresh token to be returned")
	}
	if session.TokenHash() != HashToken(raw) {
		t.Error("stored hash must match hash of the raw token")
	}
	if !session.IsValid() {
		t.Error("freshly created session should be valid")
	}
}

func TestSession_Revoke(t *testing.T) {
	session, _, err := NewSession("user-id", "", "", time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	session.Revoke(time.Now())
	if session.IsValid() {
		t.Error("revoked session should not be valid")
	}
}

func TestNewSession_MissingUserID(t *testing.T) {
	if _, _, err := NewSession("", "", "", time.Hour); err == nil {
		t.Error("expected validation error for missing user_id")
	}
}
