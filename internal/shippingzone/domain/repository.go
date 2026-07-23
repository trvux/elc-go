package domain

import "context"

// ShippingZoneRepository manages zones and the province/ward matching needed
// by LookupZone (internal/shippingzone/application/lookup_zone.go).
type ShippingZoneRepository interface {
	GetAll(ctx context.Context, filter ZoneFilter) ([]*ShippingZone, error)
	GetByID(ctx context.Context, id string) (*ShippingZone, error)
	GetDefault(ctx context.Context) (*ShippingZone, error)
	// FindByProvince returns every non-deleted zone assigned to provinceCode,
	// ward-narrowed zones and the province's catch-all zone (no ward rows)
	// alike, each with WardCodes() hydrated — the caller
	// (application.LookupZone) picks among them by exact ward membership.
	FindByProvince(ctx context.Context, provinceCode string) ([]*ShippingZone, error)
	Create(ctx context.Context, zone *ShippingZone) (*ShippingZone, error)
	Update(ctx context.Context, zone *ShippingZone) (*ShippingZone, error)
	SoftDelete(ctx context.Context, id string) error
}

// ProvinceRepository is a thin CRUD surface over the seeded province
// reference list, so admins can correct entries without a code change.
type ProvinceRepository interface {
	GetAll(ctx context.Context) ([]*Province, error)
	Create(ctx context.Context, province *Province) (*Province, error)
	Update(ctx context.Context, code, name string) (*Province, error)
	Delete(ctx context.Context, code string) error
}

// WardRepository is a thin read surface over the seeded ward (phường/xã)
// reference list — the real bottom tier of Vietnam's address hierarchy,
// used to power the cascading tỉnh -> phường/xã picker on both the admin
// zone form and the public location picker.
type WardRepository interface {
	GetAll(ctx context.Context) ([]*Ward, error)
	GetByProvince(ctx context.Context, provinceCode string) ([]*Ward, error)
}
