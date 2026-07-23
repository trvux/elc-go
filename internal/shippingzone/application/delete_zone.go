package application

import (
	"context"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func DeleteZone(ctx context.Context, repo domain.ShippingZoneRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}
