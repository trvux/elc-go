package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/settings/domain"
)

func TestSettingsUseCases(t *testing.T) {
	ctx := context.Background()
	repo := newFakeSettingsRepository()

	// 1. Get settings (empty initially)
	list, err := GetSettings(ctx, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 items, got %d", len(list))
	}

	// 2. Update settings
	s1, _ := domain.NewSiteSetting("site_name", "ELC HCMC")
	s2, _ := domain.NewSiteSetting("logo_url", "https://example.com/logo.png")

	err = UpdateSettings(ctx, repo, []*domain.SiteSetting{s1, s2})
	if err != nil {
		t.Fatalf("unexpected error on UpdateSettings: %v", err)
	}

	// 3. Get settings again
	list, err = GetSettings(ctx, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}

	var foundName, foundLogo bool
	for _, item := range list {
		if item.Key() == "site_name" && item.Value() == "ELC HCMC" {
			foundName = true
		}
		if item.Key() == "logo_url" && item.Value() == "https://example.com/logo.png" {
			foundLogo = true
		}
	}

	if !foundName || !foundLogo {
		t.Errorf("could not find expected updated settings in list: %+v", list)
	}
}
