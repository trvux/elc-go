package application

import (
	"context"

	"github.com/trvux/elc-go/internal/settings/domain"
)

func GetSettings(ctx context.Context, repo domain.SettingsRepository) ([]*domain.SiteSetting, error) {
	return repo.GetAll(ctx)
}
