package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func newTestProduct(t *testing.T) *Product {
	t.Helper()
	p, err := NewProduct(
		"cat-1", "brand-1", "Máy lạnh Daikin", "may-lanh-daikin",
		nil, nil,
		false, 0,
		nil, nil,
		nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return p
}

func TestNewProduct(t *testing.T) {
	t.Run("valid input creates a draft product", func(t *testing.T) {
		p := newTestProduct(t)
		if p.Name() != "Máy lạnh Daikin" || p.Slug() != "may-lanh-daikin" {
			t.Errorf("unexpected product: %+v", p)
		}
		if p.Status() != ProductStatusDraft {
			t.Errorf("expected new product to start as draft, got %s", p.Status())
		}
		if p.IsDeleted() {
			t.Error("expected new product to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "", "slug", nil, nil, false, 0, nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "name", "", nil, nil, false, 0, nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty category_id fails validation", func(t *testing.T) {
		_, err := NewProduct("", "brand-1", "name", "slug", nil, nil, false, 0, nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty brand_id fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "", "name", "slug", nil, nil, false, 0, nil, nil, nil, nil)
		assertValidationError(t, err)
	})
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
}

func TestProduct_UpdateName(t *testing.T) {
	p := newTestProduct(t)

	if err := p.UpdateName("Máy lạnh Daikin Updated"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "Máy lạnh Daikin Updated" {
		t.Errorf("expected updated name, got %s", p.Name())
	}
	if err := p.UpdateName(""); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestProduct_MarkDeletedAndRestore(t *testing.T) {
	p := newTestProduct(t)

	p.MarkDeleted(p.CreatedAt())
	if !p.IsDeleted() {
		t.Error("expected product to be deleted")
	}

	p.Restore()
	if p.IsDeleted() {
		t.Error("expected product to be restored")
	}
	if p.DeletedAt() != nil {
		t.Error("expected deletedAt to be cleared after restore")
	}
}

func TestProduct_SetFeaturedReorder(t *testing.T) {
	p := newTestProduct(t)

	p.SetFeatured(true)
	if !p.IsFeatured() {
		t.Error("expected product to be featured")
	}

	p.Reorder(5)
	if p.OrderIndex() != 5 {
		t.Errorf("expected orderIndex 5, got %d", p.OrderIndex())
	}
}

func TestProduct_StatusTransitions(t *testing.T) {
	t.Run("happy path: draft -> proposed -> published -> archived -> published", func(t *testing.T) {
		p := newTestProduct(t)

		if err := p.SubmitForReview(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status() != ProductStatusProposed {
			t.Errorf("expected proposed, got %s", p.Status())
		}

		if err := p.Approve(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status() != ProductStatusPublished {
			t.Errorf("expected published, got %s", p.Status())
		}

		if err := p.Archive(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status() != ProductStatusArchived {
			t.Errorf("expected archived, got %s", p.Status())
		}

		if err := p.Unarchive(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status() != ProductStatusPublished {
			t.Errorf("expected published after unarchive, got %s", p.Status())
		}
	})

	t.Run("reject sends a proposed product back to draft with a reason", func(t *testing.T) {
		p := newTestProduct(t)
		if err := p.SubmitForReview(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := p.Reject("missing warranty info"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status() != ProductStatusDraft {
			t.Errorf("expected draft after reject, got %s", p.Status())
		}
		if p.RejectionReason() == nil || *p.RejectionReason() != "missing warranty info" {
			t.Errorf("expected rejection reason to be set, got %v", p.RejectionReason())
		}
	})

	t.Run("approve clears an earlier rejection reason", func(t *testing.T) {
		p := newTestProduct(t)
		_ = p.SubmitForReview()
		_ = p.Reject("fix this")
		_ = p.SubmitForReview()

		if err := p.Approve(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.RejectionReason() != nil {
			t.Errorf("expected rejection reason cleared after approve, got %v", p.RejectionReason())
		}
	})

	t.Run("invalid transitions fail", func(t *testing.T) {
		p := newTestProduct(t)

		if err := p.Approve(); err == nil {
			t.Error("expected error approving a draft product")
		}
		if err := p.Archive(); err == nil {
			t.Error("expected error archiving a draft product")
		}
		if err := p.Unarchive(); err == nil {
			t.Error("expected error unarchiving a draft product")
		}
	})
}
