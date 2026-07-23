package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func UpdateZone(ctx context.Context, repo domain.ShippingZoneRepository, input domain.UpdateZoneInput) (*domain.ShippingZone, error) {
	z, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, apperr.NewNotFoundError("shipping zone")
	}

	if input.Name != nil {
		if err := z.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.FeeVND != nil {
		if err := z.UpdateFeeVND(*input.FeeVND); err != nil {
			return nil, err
		}
	}
	if input.MinDays != nil || input.MaxDays != nil {
		minDays, maxDays := z.MinDays(), z.MaxDays()
		if input.MinDays != nil {
			minDays = *input.MinDays
		}
		if input.MaxDays != nil {
			maxDays = *input.MaxDays
		}
		if err := z.UpdateDayRange(minDays, maxDays); err != nil {
			return nil, err
		}
	}
	if input.IsDefault != nil {
		z.SetDefault(*input.IsDefault)
	}
	if input.ProvinceCodes != nil {
		z.UpdateProvinceCodes(input.ProvinceCodes)
	}
	if input.WardCodes != nil {
		z.UpdateWardCodes(input.WardCodes)
	}

	return repo.Update(ctx, z)
}
