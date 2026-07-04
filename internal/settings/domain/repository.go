package domain

import "context"

type SettingsRepository interface {
	GetAll(ctx context.Context) ([]*SiteSetting, error)
	UpdateMany(ctx context.Context, settings []*SiteSetting) error
}
