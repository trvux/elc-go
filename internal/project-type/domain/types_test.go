package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewProjectType(t *testing.T) {
	t.Run("valid input creates a project type", func(t *testing.T) {
		pt, err := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0, nil)
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
		_, err := NewProjectType("", "nha-xuong", nil, nil, nil, false, 0, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewProjectType("Nha xuong", "", nil, nil, nil, false, 0, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestProjectType_Update(t *testing.T) {
	pt, _ := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0, nil)

	t.Run("update name succeeds", func(t *testing.T) {
		newName := "Nha xuong updated"
		if err := pt.Update(UpdateProjectTypeInput{Name: &newName}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pt.Name() != "Nha xuong updated" {
			t.Errorf("expected updated name, got %s", pt.Name())
		}
	})

	t.Run("update name with empty fails", func(t *testing.T) {
		empty := ""
		if err := pt.Update(UpdateProjectTypeInput{Name: &empty}); err == nil {
			t.Fatal("expected validation error for empty name")
		}
	})

	t.Run("update slug succeeds", func(t *testing.T) {
		newSlug := "nha-xuong-moi"
		if err := pt.Update(UpdateProjectTypeInput{Slug: &newSlug}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pt.Slug() != "nha-xuong-moi" {
			t.Errorf("expected updated slug, got %s", pt.Slug())
		}
	})

	t.Run("update slug with empty fails", func(t *testing.T) {
		empty := ""
		if err := pt.Update(UpdateProjectTypeInput{Slug: &empty}); err == nil {
			t.Fatal("expected validation error for empty slug")
		}
	})

	t.Run("update content succeeds", func(t *testing.T) {
		content := []byte(`{"text": "gioi thieu"}`)
		if err := pt.Update(UpdateProjectTypeInput{Content: content}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(pt.Content()) != string(content) {
			t.Errorf("expected updated content, got %s", pt.Content())
		}
	})

	t.Run("no-op update leaves updatedAt untouched", func(t *testing.T) {
		before := pt.UpdatedAt()
		if err := pt.Update(UpdateProjectTypeInput{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !pt.UpdatedAt().Equal(before) {
			t.Errorf("expected updatedAt unchanged, before=%v after=%v", before, pt.UpdatedAt())
		}
	})
}

func TestProjectType_MarkDeletedAndRestore(t *testing.T) {
	pt, _ := NewProjectType("Nha xuong", "nha-xuong", nil, nil, nil, false, 0, nil)

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
