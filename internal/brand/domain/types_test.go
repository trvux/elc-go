package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewBrand(t *testing.T) {
	t.Run("valid input creates a brand", func(t *testing.T) {
		b, err := NewBrand("Apple", "apple", "https://example.com/logo.png", nil, nil, false, 0, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Name() != "Apple" || b.Slug() != "apple" {
			t.Errorf("unexpected brand: %+v", b)
		}
		if b.IsDeleted() {
			t.Error("expected new brand to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewBrand("", "apple", "", nil, nil, false, 0, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewBrand("Apple", "", "", nil, nil, false, 0, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestBrand_UpdateName(t *testing.T) {
	b, _ := NewBrand("Apple", "apple", "", nil, nil, false, 0, nil, nil)

	if err := b.UpdateName("Apple Updated"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Name() != "Apple Updated" {
		t.Errorf("expected updated name, got %s", b.Name())
	}

	if err := b.UpdateName(""); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestBrand_UpdateSlug(t *testing.T) {
	b, _ := NewBrand("Apple", "apple", "", nil, nil, false, 0, nil, nil)

	if err := b.UpdateSlug("apple-inc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Slug() != "apple-inc" {
		t.Errorf("expected updated slug, got %s", b.Slug())
	}

	if err := b.UpdateSlug(""); err == nil {
		t.Fatal("expected validation error for empty slug")
	}
}

func TestBrand_MarkDeletedAndRestore(t *testing.T) {
	b, _ := NewBrand("Apple", "apple", "", nil, nil, false, 0, nil, nil)

	b.MarkDeleted(b.CreatedAt())
	if !b.IsDeleted() {
		t.Error("expected brand to be deleted")
	}

	b.Restore()
	if b.IsDeleted() {
		t.Error("expected brand to be restored")
	}
	if b.DeletedAt() != nil {
		t.Error("expected deletedAt to be cleared after restore")
	}
}

func TestBrand_SetFAQ(t *testing.T) {
	b, _ := NewBrand("Apple", "apple", "", nil, nil, false, 0, nil, nil)

	faq := []FAQItem{{Question: "Q1", Answer: "A1"}}
	b.SetFAQ(faq)

	if len(b.FAQ()) != 1 || b.FAQ()[0].Question != "Q1" {
		t.Errorf("expected faq to be set, got %+v", b.FAQ())
	}
}
