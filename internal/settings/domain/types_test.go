package domain

import "testing"

func TestNewSiteSetting(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		s, err := NewSiteSetting("site_name", "ELC Dien May")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Key() != "site_name" || s.Value() != "ELC Dien May" {
			t.Errorf("unexpected entity state: %+v", s)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := NewSiteSetting("", "Value")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
