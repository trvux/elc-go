package domain

import (
	"encoding/json"
	"testing"
)

func TestNewPage(t *testing.T) {
	content := json.RawMessage(`{"type":"doc"}`)
	title := "About Us"
	slug := "about-us"
	metaTitle := "About Us Page"

	t.Run("valid inputs", func(t *testing.T) {
		p, err := NewPage(title, slug, "", content, true, &metaTitle, nil, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Title() != title || p.Slug() != slug || p.IsPublished() != true || p.OrderIndex() != 1 {
			t.Errorf("unexpected page state: %+v", p)
		}
		if p.TitleAlign() != TitleAlignCenter {
			t.Errorf("expected titleAlign to default to center, got %s", p.TitleAlign())
		}
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := NewPage("", slug, "", content, true, nil, nil, 0)
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
	})

	t.Run("empty slug", func(t *testing.T) {
		_, err := NewPage(title, "", "", content, true, nil, nil, 0)
		if err == nil {
			t.Fatal("expected error for empty slug, got nil")
		}
	})

	t.Run("invalid titleAlign", func(t *testing.T) {
		_, err := NewPage(title, slug, "sideways", content, true, nil, nil, 0)
		if err == nil {
			t.Fatal("expected error for invalid titleAlign, got nil")
		}
	})
}
