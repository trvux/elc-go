package application

import (
	"context"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

// LookupZone resolves the shipping zone for a customer's province + ward
// (phường/xã — the real bottom tier of Vietnam's address hierarchy since
// quận/huyện was abolished in the July 2025 reform), in this order:
//  1. Among zones assigned to provinceCode, the one whose WardCodes()
//     contains wardCode exactly.
//  2. The zone assigned to provinceCode with no wards at all (its
//     province-wide catch-all).
//  3. The site-wide default zone (ShippingZoneRepository.GetDefault), for a
//     province with no zone configured yet.
//
// Passing an empty wardCode skips straight to step 2/3 — this is how
// GetDefaultZoneForDisplay-style callers reuse the same fallback chain to
// produce the value embedded in static JSON-LD.
func LookupZone(ctx context.Context, repo domain.ShippingZoneRepository, provinceCode, wardCode string) (*domain.ShippingZone, error) {
	zones, err := repo.FindByProvince(ctx, provinceCode)
	if err != nil {
		return nil, err
	}

	var catchAll *domain.ShippingZone
	for _, z := range zones {
		if len(z.WardCodes()) == 0 {
			if catchAll == nil {
				catchAll = z
			}
			continue
		}
		if wardCode == "" {
			continue
		}
		for _, wc := range z.WardCodes() {
			if wc == wardCode {
				return z, nil
			}
		}
	}

	if catchAll != nil {
		return catchAll, nil
	}

	return repo.GetDefault(ctx)
}
