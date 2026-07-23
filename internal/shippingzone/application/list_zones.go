package application

import (
	"context"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func GetZones(ctx context.Context, repo domain.ShippingZoneRepository, filter domain.ZoneFilter) ([]*domain.ShippingZone, error) {
	return repo.GetAll(ctx, filter)
}

func GetZoneByID(ctx context.Context, repo domain.ShippingZoneRepository, id string) (*domain.ShippingZone, error) {
	return repo.GetByID(ctx, id)
}

func GetDefaultZone(ctx context.Context, repo domain.ShippingZoneRepository) (*domain.ShippingZone, error) {
	return repo.GetDefault(ctx)
}
