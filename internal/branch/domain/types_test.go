package domain

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewBranch(t *testing.T) {
	desc := json.RawMessage(`{"text": "Chi nhánh miền Nam"}`)
	images := []ImageAsset{{URL: "https://example.com/branch.png"}}

	t.Run("valid input creates a branch", func(t *testing.T) {
		b, err := NewBranch(
			"ELC Q1", "elc-q1", "123 Le Loi, Q1, HCMC", "0901234567",
			"q1@elc.vn", "https://maps.google.com/q1", "<iframe src=\"https://www.google.com/maps/embed?pb=abc123\" width=\"600\" height=\"450\" style=\"border:0;\" allowfullscreen=\"\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>",
			nil, nil, nil, nil, nil,
			desc, images, true, 1, nil, nil,
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
			"q1@elc.vn", "https://maps.google.com/q1", "<iframe src=\"https://www.google.com/maps/embed?pb=abc123\" width=\"600\" height=\"450\" style=\"border:0;\" allowfullscreen=\"\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>",
			nil, nil, nil, nil, nil,
			desc, images, true, 1, nil, nil,
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
			"invalid-email", "https://maps.google.com/q1", "<iframe src=\"https://www.google.com/maps/embed?pb=abc123\" width=\"600\" height=\"450\" style=\"border:0;\" allowfullscreen=\"\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>",
			nil, nil, nil, nil, nil,
			desc, images, true, 1, nil, nil,
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
			"q1@elc.vn", "not-a-valid-url", "<iframe src=\"https://www.google.com/maps/embed?pb=abc123\" width=\"600\" height=\"450\" style=\"border:0;\" allowfullscreen=\"\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>",
			nil, nil, nil, nil, nil,
			desc, images, true, 1, nil, nil,
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

func TestValidateMapsEmbed(t *testing.T) {
	cases := []struct {
		name    string
		embed   string
		wantErr bool
	}{
		{
			name:    "valid Google Maps embed snippet",
			embed:   `<iframe src="https://www.google.com/maps/embed?pb=abc123" width="600" height="450" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="no-referrer-when-downgrade"></iframe>`,
			wantErr: false,
		},
		{name: "empty", embed: "", wantErr: true},
		{name: "no iframe at all", embed: "<div>not an iframe</div>", wantErr: true},
		{name: "iframe with no src", embed: `<iframe width="600"></iframe>`, wantErr: true},
		{name: "src not pointing at google.com", embed: `<iframe src="https://evil.example.com/embed"></iframe>`, wantErr: true},
		{name: "javascript: src", embed: `<iframe src="javascript:alert(1)"></iframe>`, wantErr: true},
		{name: "event handler attribute", embed: `<iframe src="https://www.google.com/maps/embed?pb=x" onload="alert(document.cookie)"></iframe>`, wantErr: true},
		{name: "css injection via style attribute value", embed: `<iframe src="https://www.google.com/maps/embed?pb=x" style="background:url('https://evil.example.com/exfil')"></iframe>`, wantErr: true},
		{name: "google's own style value is allowed", embed: `<iframe src="https://www.google.com/maps/embed?pb=x" style="border:0;"></iframe>`, wantErr: false},
		{name: "second tag smuggled in", embed: `<iframe src="https://www.google.com/maps/embed?pb=x"></iframe><script>alert(1)</script>`, wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs := validateMapsEmbed(c.embed)
			if c.wantErr && len(errs) == 0 {
				t.Errorf("expected validation error, got none")
			}
			if !c.wantErr && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
		})
	}
}

func TestBranch_UpdateFields(t *testing.T) {
	desc := json.RawMessage(`{"text": "Chi nhánh"}`)
	b, _ := NewBranch(
		"ELC Q1", "elc-q1", "123 Le Loi", "0901234567",
		"q1@elc.vn", "https://maps.google.com/q1", "<iframe src=\"https://www.google.com/maps/embed?pb=abc123\" width=\"600\" height=\"450\" style=\"border:0;\" allowfullscreen=\"\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>",
		nil, nil, nil, nil, nil,
		desc, nil, true, 1, nil, nil,
	)

	t.Run("update name succeeds", func(t *testing.T) {
		newName := "ELC Q3"
		if err := b.Update(UpdateBranchInput{Name: &newName}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Name() != "ELC Q3" {
			t.Errorf("expected ELC Q3, got %s", b.Name())
		}
	})

	t.Run("update name with empty fails", func(t *testing.T) {
		empty := ""
		if err := b.Update(UpdateBranchInput{Name: &empty}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("update email succeeds", func(t *testing.T) {
		newEmail := "q3@elc.vn"
		if err := b.Update(UpdateBranchInput{Email: &newEmail}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Email() != "q3@elc.vn" {
			t.Errorf("expected q3@elc.vn, got %s", b.Email())
		}
	})

	t.Run("no-op update leaves updatedAt untouched", func(t *testing.T) {
		before := b.UpdatedAt()
		if err := b.Update(UpdateBranchInput{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b.UpdatedAt().Equal(before) {
			t.Errorf("expected updatedAt unchanged, before=%v after=%v", before, b.UpdatedAt())
		}
	})
}
