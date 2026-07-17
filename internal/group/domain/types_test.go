package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewGroup(t *testing.T) {
	t.Run("valid input creates a group", func(t *testing.T) {
		imageUrl := "https://example.com/logo.png"
		g, err := NewGroup("Air Conditioners", "air-conditioners", &imageUrl, nil, nil, false, false, 0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if g.Name() != "Air Conditioners" || g.Slug() != "air-conditioners" {
			t.Errorf("unexpected group: %+v", g)
		}
		if g.IsDeleted() {
			t.Error("expected new group to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewGroup("", "air-conditioners", nil, nil, nil, false, false, 0, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
	})

	t.Run("overlong name fails validation", func(t *testing.T) {
		longName := strings.Repeat("a", 101)
		_, err := NewGroup(longName, "air-conditioners", nil, nil, nil, false, false, 0, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewGroup("Air Conditioners", "", nil, nil, nil, false, false, 0, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestGroup_UpdateName(t *testing.T) {
	g, _ := NewGroup("Air Conditioners", "air-conditioners", nil, nil, nil, false, false, 0, nil)

	if err := g.UpdateName("Air Conditioners Updated"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name() != "Air Conditioners Updated" {
		t.Errorf("expected updated name, got %s", g.Name())
	}

	if err := g.UpdateName(""); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestGroup_UpdateSlug(t *testing.T) {
	g, _ := NewGroup("Air Conditioners", "air-conditioners", nil, nil, nil, false, false, 0, nil)

	if err := g.UpdateSlug("ac"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Slug() != "ac" {
		t.Errorf("expected updated slug, got %s", g.Slug())
	}

	if err := g.UpdateSlug(""); err == nil {
		t.Fatal("expected validation error for empty slug")
	}
}

func TestGroup_MarkDeletedAndRestore(t *testing.T) {
	g, _ := NewGroup("Air Conditioners", "air-conditioners", nil, nil, nil, false, false, 0, nil)

	g.MarkDeleted(g.CreatedAt())
	if !g.IsDeleted() {
		t.Error("expected group to be deleted")
	}

	g.Restore()
	if g.IsDeleted() {
		t.Error("expected group to be restored")
	}
	if g.DeletedAt() != nil {
		t.Error("expected deletedAt to be cleared after restore")
	}
}
