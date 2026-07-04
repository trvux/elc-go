package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewContact(t *testing.T) {
	tests := []struct {
		name        string
		contactType string
		label       *string
		value       string
		wantErr     bool
	}{
		{name: "valid input", contactType: "phone", value: "0901234567", wantErr: false},
		{name: "empty type", contactType: "", value: "0901234567", wantErr: true},
		{name: "empty value", contactType: "phone", value: "", wantErr: true},
		{name: "label too long", contactType: "phone", value: "0901234567", label: strPtr(longLabel()), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contact, err := NewContact(tt.contactType, tt.label, tt.value, true, 0)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var appErr *apperr.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected *apperr.AppError, got %T", err)
				}
				if appErr.Code != "VALIDATION_ERROR" {
					t.Errorf("expected code VALIDATION_ERROR, got %s", appErr.Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if contact.Type() != tt.contactType {
				t.Errorf("expected type %s, got %s", tt.contactType, contact.Type())
			}
			if contact.Value() != tt.value {
				t.Errorf("expected value %s, got %s", tt.value, contact.Value())
			}
		})
	}
}

func TestContact_Href(t *testing.T) {
	tests := []struct {
		name        string
		contactType string
		value       string
		want        string
	}{
		{name: "phone strips spaces", contactType: "phone", value: "090 123 4567", want: "tel:0901234567"},
		{name: "email", contactType: "email", value: "a@b.com", want: "mailto:a@b.com"},
		{name: "zalo", contactType: "zalo", value: "0901234567", want: "https://zalo.me/0901234567"},
		{name: "messenger", contactType: "messenger", value: "user.name", want: "https://m.me/user.name"},
		{name: "facebook", contactType: "facebook", value: "user.name", want: "https://facebook.com/user.name"},
		{name: "unknown type falls back to raw value", contactType: "website", value: "example.com", want: "example.com"},
		{name: "http prefix bypasses type logic entirely", contactType: "facebook", value: "http://example.com", want: "http://example.com"},
		{name: "empty value", contactType: "phone", value: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// RehydrateContact bypasses validation, so it can build edge cases
			// (like an empty value) that NewContact would reject outright.
			contact := RehydrateContact("id-1", tt.contactType, nil, tt.value, true, 0)

			if got := contact.Href(); got != tt.want {
				t.Errorf("Href() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContact_IsExternal(t *testing.T) {
	tests := []struct {
		name        string
		contactType string
		value       string
		want        bool
	}{
		{name: "phone is not external", contactType: "phone", value: "0901234567", want: false},
		{name: "email is not external", contactType: "email", value: "a@b.com", want: false},
		{name: "facebook is external", contactType: "facebook", value: "user.name", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contact := RehydrateContact("id-1", tt.contactType, nil, tt.value, true, 0)

			if got := contact.IsExternal(); got != tt.want {
				t.Errorf("IsExternal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContact_UpdateValue(t *testing.T) {
	contact := RehydrateContact("id-1", "phone", nil, "0901234567", true, 0)

	if err := contact.UpdateValue("0909999999"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contact.Value() != "0909999999" {
		t.Errorf("expected value to be updated, got %s", contact.Value())
	}

	if err := contact.UpdateValue(""); err == nil {
		t.Fatal("expected error for empty value, got nil")
	}
	// invariant: a rejected update must not mutate the entity
	if contact.Value() != "0909999999" {
		t.Errorf("value should be unchanged after rejected update, got %s", contact.Value())
	}
}

func strPtr(s string) *string { return &s }

func longLabel() string {
	b := make([]byte, 101)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
