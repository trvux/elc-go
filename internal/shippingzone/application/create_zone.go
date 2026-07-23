package application

import (
	"context"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func CreateZone(ctx context.Context, repo domain.ShippingZoneRepository, input domain.CreateZoneInput) (*domain.ShippingZone, error) {
	z, err := domain.NewShippingZone(
		input.Name,
		input.FeeVND,
		input.MinDays,
		input.MaxDays,
		input.IsDefault,
		input.ProvinceCodes,
		input.WardCodes,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, z)
}
