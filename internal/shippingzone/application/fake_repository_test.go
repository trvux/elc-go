package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

type fakeShippingZoneRepository struct {
	items     map[string]*domain.ShippingZone
	provinces map[string][]string // zoneID -> province codes
}

func newFakeShippingZoneRepository() *fakeShippingZoneRepository {
	return &fakeShippingZoneRepository{
		items:     map[string]*domain.ShippingZone{},
		provinces: map[string][]string{},
	}
}

func (r *fakeShippingZoneRepository) GetAll(ctx context.Context, filter domain.ZoneFilter) ([]*domain.ShippingZone, error) {
	var result []*domain.ShippingZone
	for _, z := range r.items {
		if !filter.IncludeDeleted && z.IsDeleted() {
			continue
		}
		result = append(result, z)
	}
	return result, nil
}

func (r *fakeShippingZoneRepository) GetByID(ctx context.Context, id string) (*domain.ShippingZone, error) {
	z, ok := r.items[id]
	if !ok || z.IsDeleted() {
		return nil, nil
	}
	return z, nil
}

func (r *fakeShippingZoneRepository) GetDefault(ctx context.Context) (*domain.ShippingZone, error) {
	for _, z := range r.items {
		if z.IsDefault() && !z.IsDeleted() {
			return z, nil
		}
	}
	return nil, nil
}

func (r *fakeShippingZoneRepository) FindByProvince(ctx context.Context, provinceCode string) ([]*domain.ShippingZone, error) {
	var result []*domain.ShippingZone
	for zoneID, codes := range r.provinces {
		z, ok := r.items[zoneID]
		if !ok || z.IsDeleted() {
			continue
		}
		for _, c := range codes {
			if c == provinceCode {
				result = append(result, z)
				break
			}
		}
	}
	return result, nil
}

func (r *fakeShippingZoneRepository) Create(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	id := fmt.Sprintf("zone-%d", len(r.items)+1)
	created := domain.RehydrateShippingZone(
		id, zone.Name(), zone.FeeVND(), zone.MinDays(), zone.MaxDays(), zone.IsDefault(),
		zone.ProvinceCodes(), zone.WardCodes(), zone.CreatedAt(), zone.UpdatedAt(), nil,
	)
	r.items[id] = created
	r.provinces[id] = zone.ProvinceCodes()
	return created, nil
}

func (r *fakeShippingZoneRepository) Update(ctx context.Context, zone *domain.ShippingZone) (*domain.ShippingZone, error) {
	r.items[zone.ID()] = zone
	r.provinces[zone.ID()] = zone.ProvinceCodes()
	return zone, nil
}

func (r *fakeShippingZoneRepository) SoftDelete(ctx context.Context, id string) error {
	z, ok := r.items[id]
	if !ok {
		return nil
	}
	z.MarkDeleted(z.CreatedAt())
	return nil
}
