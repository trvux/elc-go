package application

import (
	"context"

	"github.com/trvux/elc-go/internal/settings/domain"
)

func UpdateSettings(ctx context.Context, repo domain.SettingsRepository, settings []*domain.SiteSetting) error {
	return repo.UpdateMany(ctx, settings)
}
