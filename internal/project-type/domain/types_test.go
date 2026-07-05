package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewProjectType(t *testing.T) {
	t.Run("valid input creates a project type", func(t *testing.T) {
		pt, err := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pt.Name() != "Nha xuong" || pt.Slug() != "nha-xuong" {
			t.Errorf("unexpected project type: %+v", pt)
		}
		if pt.IsDeleted() {
			t.Error("expected new project type to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewProjectType("", "nha-xuong", nil, nil, nil, false, 0)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewProjectType("Nha xuong", "", nil, nil, nil, false, 0)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestProjectType_UpdateName(t *testing.T) {
	pt, _ := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0)

	if err := pt.UpdateName("Nha xuong updated"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.Name() != "Nha xuong updated" {
		t.Errorf("expected updated name, got %s", pt.Name())
	}

	if err := pt.UpdateName(""); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestProjectType_UpdateSlug(t *testing.T) {
	pt, _ := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0)

	if err := pt.UpdateSlug("nha-xuong-moi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.Slug() != "nha-xuong-moi" {
		t.Errorf("expected updated slug, got %s", pt.Slug())
	}

	if err := pt.UpdateSlug(""); err == nil {
		t.Fatal("expected validation error for empty slug")
	}
}

func TestProjectType_MarkDeletedAndRestore(t *testing.T) {
	pt, _ := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0)

	pt.MarkDeleted(pt.CreatedAt())
	if !pt.IsDeleted() {
		t.Error("expected project type to be deleted")
	}

	pt.Restore()
	if pt.IsDeleted() {
		t.Error("expected project type to be restored")
	}
	if pt.DeletedAt() != nil {
		t.Error("expected deletedAt to be cleared after restore")
	}
}
