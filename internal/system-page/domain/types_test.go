package domain

import (
	"strings"
	"testing"
	"time"
)

func TestSystemPage_UpdateMeta(t *testing.T) {
	t.Run("valid update", func(t *testing.T) {
		p := RehydrateSystemPage("id-1", "Trang chủ", "home", nil, nil, time.Now(), time.Now())

		title := "New title"
		desc := "New description"
		if err := p.UpdateMeta(&title, &desc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *p.MetaTitle() != title || *p.MetaDescription() != desc {
			t.Errorf("unexpected entity state: %+v", p)
		}
	})

	t.Run("nil meta is allowed", func(t *testing.T) {
		p := RehydrateSystemPage("id-1", "Trang chủ", "home", nil, nil, time.Now(), time.Now())

		if err := p.UpdateMeta(nil, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.MetaTitle() != nil || p.MetaDescription() != nil {
			t.Errorf("expected nil meta fields, got: %+v", p)
		}
	})

	t.Run("meta title too long", func(t *testing.T) {
		p := RehydrateSystemPage("id-1", "Trang chủ", "home", nil, nil, time.Now(), time.Now())

		title := strings.Repeat("a", 71)
		if err := p.UpdateMeta(&title, nil); err == nil {
			t.Fatal("expected error for meta title over 70 chars, got nil")
		}
	})

	t.Run("meta description too long", func(t *testing.T) {
		p := RehydrateSystemPage("id-1", "Trang chủ", "home", nil, nil, time.Now(), time.Now())

		desc := strings.Repeat("a", 161)
		if err := p.UpdateMeta(nil, &desc); err == nil {
			t.Fatal("expected error for meta description over 160 chars, got nil")
		}
	})
}
