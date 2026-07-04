package domain

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewBranch(t *testing.T) {
	desc := json.RawMessage(`{"text": "Chi nhánh miền Nam"}`)
	img := "https://example.com/branch.png"

	t.Run("valid input creates a branch", func(t *testing.T) {
		b, err := NewBranch(
			"ELC Q1", "elc-q1", "123 Le Loi, Q1, HCMC", "0901234567",
			"q1@elc.vn", "https://maps.google.com/q1", "<iframe></iframe>",
			desc, &img, true, 1, nil, nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Name() != "ELC Q1" || b.Slug() != "elc-q1" || b.Address() != "123 Le Loi, Q1, HCMC" {
			t.Errorf("unexpected branch properties: %+v", b)
		}
		if b.IsDeleted() {
			t.Error("expected new branch to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewBranch(
			"", "elc-q1", "123 Le Loi, Q1, HCMC", "0901234567",
			"q1@elc.vn", "https://maps.google.com/q1", "<iframe></iframe>",
			desc, &img, true, 1, nil, nil,
		)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if _, ok := appErr.Fields["name"]; !ok {
			t.Errorf("expected validation field for name, got: %+v", appErr.Fields)
		}
	})

	t.Run("invalid email fails validation", func(t *testing.T) {
		_, err := NewBranch(
			"ELC Q1", "elc-q1", "123 Le Loi, Q1, HCMC", "0901234567",
			"invalid-email", "https://maps.google.com/q1", "<iframe></iframe>",
			desc, &img, true, 1, nil, nil,
		)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if _, ok := appErr.Fields["email"]; !ok {
			t.Errorf("expected validation field for email, got: %+v", appErr.Fields)
		}
	})

	t.Run("invalid mapsUrl fails validation", func(t *testing.T) {
		_, err := NewBranch(
			"ELC Q1", "elc-q1", "123 Le Loi, Q1, HCMC", "0901234567",
			"q1@elc.vn", "not-a-valid-url", "<iframe></iframe>",
			desc, &img, true, 1, nil, nil,
		)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if _, ok := appErr.Fields["mapsUrl"]; !ok {
			t.Errorf("expected validation field for mapsUrl, got: %+v", appErr.Fields)
		}
	})
}

func TestBranch_UpdateFields(t *testing.T) {
	desc := json.RawMessage(`{"text": "Chi nhánh"}`)
	b, _ := NewBranch(
		"ELC Q1", "elc-q1", "123 Le Loi", "0901234567",
		"q1@elc.vn", "https://maps.google.com/q1", "<iframe></iframe>",
		desc, nil, true, 1, nil, nil,
	)

	t.Run("update name succeeds", func(t *testing.T) {
		if err := b.UpdateName("ELC Q3"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Name() != "ELC Q3" {
			t.Errorf("expected ELC Q3, got %s", b.Name())
		}
	})

	t.Run("update name with empty fails", func(t *testing.T) {
		if err := b.UpdateName(""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update email succeeds", func(t *testing.T) {
		if err := b.UpdateEmail("q3@elc.vn"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Email() != "q3@elc.vn" {
			t.Errorf("expected q3@elc.vn, got %s", b.Email())
		}
	})
}
