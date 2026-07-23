package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Province is a simple reference row (current, post-July-2025-merger
// province/city list) — see internal/shippingzone/migrations for the seed
// data and a note on why this isn't a rigid nationwide district dataset.
type Province struct {
	Code string
	Name string
}

// Ward is a phường/xã reference row — the real bottom tier of Vietnam's
// administrative hierarchy since the July 2025 reform abolished quận/huyện
// (tỉnh/thành -> phường/xã directly now). Seeded from
// thanglequoc/vietnamese-provinces-database — see the migration.
type Ward struct {
	Code         string
	Name         string
	ProvinceCode string
}

// ShippingZone is an admin-defined delivery area: a name, a fee/day-range,
// and the set of provinces + optional specific wards it matches. A province
// can be linked to more than one zone (a ward-narrowed zone plus a
// catch-all zone for the same province) — see LookupZone in the application
// layer for the matching order.
type ShippingZone struct {
	id            string
	name          string
	feeVND        int64
	minDays       int
	maxDays       int
	isDefault     bool
	provinceCodes []string
	wardCodes     []string
	createdAt     time.Time
	updatedAt     time.Time
	deletedAt     *time.Time
}

func NewShippingZone(
	name string,
	feeVND int64,
	minDays, maxDays int,
	isDefault bool,
	provinceCodes []string,
	wardCodes []string,
) (*ShippingZone, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if feeVND < 0 {
		fields["feeVND"] = append(fields["feeVND"], "feeVND must not be negative")
	}
	if minDays < 0 {
		fields["minDays"] = append(fields["minDays"], "minDays must not be negative")
	}
	if maxDays < minDays {
		fields["maxDays"] = append(fields["maxDays"], "maxDays must be greater than or equal to minDays")
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &ShippingZone{
		name:          name,
		feeVND:        feeVND,
		minDays:       minDays,
		maxDays:       maxDays,
		isDefault:     isDefault,
		provinceCodes: provinceCodes,
		wardCodes:     wardCodes,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// RehydrateShippingZone reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateShippingZone(
	id, name string,
	feeVND int64,
	minDays, maxDays int,
	isDefault bool,
	provinceCodes []string,
	wardCodes []string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *ShippingZone {
	return &ShippingZone{
		id:            id,
		name:          name,
		feeVND:        feeVND,
		minDays:       minDays,
		maxDays:       maxDays,
		isDefault:     isDefault,
		provinceCodes: provinceCodes,
		wardCodes:     wardCodes,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deletedAt:     deletedAt,
	}
}

func (z *ShippingZone) ID() string              { return z.id }
func (z *ShippingZone) Name() string            { return z.name }
func (z *ShippingZone) FeeVND() int64           { return z.feeVND }
func (z *ShippingZone) MinDays() int            { return z.minDays }
func (z *ShippingZone) MaxDays() int            { return z.maxDays }
func (z *ShippingZone) IsDefault() bool         { return z.isDefault }
func (z *ShippingZone) ProvinceCodes() []string { return z.provinceCodes }
func (z *ShippingZone) WardCodes() []string     { return z.wardCodes }
func (z *ShippingZone) CreatedAt() time.Time    { return z.createdAt }
func (z *ShippingZone) UpdatedAt() time.Time    { return z.updatedAt }
func (z *ShippingZone) DeletedAt() *time.Time   { return z.deletedAt }
func (z *ShippingZone) IsDeleted() bool         { return z.deletedAt != nil }

func (z *ShippingZone) MarkDeleted(deletedAt time.Time) {
	z.deletedAt = &deletedAt
}

func (z *ShippingZone) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	z.name = name
	z.updatedAt = time.Now()
	return nil
}

func (z *ShippingZone) UpdateFeeVND(feeVND int64) error {
	if feeVND < 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"feeVND": {"feeVND must not be negative"}})
	}
	z.feeVND = feeVND
	z.updatedAt = time.Now()
	return nil
}

func (z *ShippingZone) UpdateDayRange(minDays, maxDays int) error {
	fields := map[string][]string{}
	if minDays < 0 {
		fields["minDays"] = append(fields["minDays"], "minDays must not be negative")
	}
	if maxDays < minDays {
		fields["maxDays"] = append(fields["maxDays"], "maxDays must be greater than or equal to minDays")
	}
	if len(fields) > 0 {
		return apperr.NewValidationError("validation failed", fields)
	}
	z.minDays = minDays
	z.maxDays = maxDays
	z.updatedAt = time.Now()
	return nil
}

func (z *ShippingZone) SetDefault(isDefault bool) {
	z.isDefault = isDefault
	z.updatedAt = time.Now()
}

func (z *ShippingZone) UpdateProvinceCodes(provinceCodes []string) {
	z.provinceCodes = provinceCodes
	z.updatedAt = time.Now()
}

func (z *ShippingZone) UpdateWardCodes(wardCodes []string) {
	z.wardCodes = wardCodes
	z.updatedAt = time.Now()
}

// SetRelations is for the infrastructure layer only, to attach
// provinces/wards loaded in a second query after the zone row itself was
// already scanned — unlike UpdateProvinceCodes/UpdateWardCodes, it does not
// touch updatedAt, since nothing is actually being changed here.
func (z *ShippingZone) SetRelations(provinceCodes, wardCodes []string) {
	z.provinceCodes = provinceCodes
	z.wardCodes = wardCodes
}

func validateName(name string) []string {
	var errs []string
	if name == "" {
		errs = append(errs, "name is required")
	} else if len(name) > 150 {
		errs = append(errs, "name must not exceed 150 characters")
	}
	return errs
}

type CreateZoneInput struct {
	Name          string
	FeeVND        int64
	MinDays       int
	MaxDays       int
	IsDefault     bool
	ProvinceCodes []string
	WardCodes     []string
}

type UpdateZoneInput struct {
	ID            string
	Name          *string
	FeeVND        *int64
	MinDays       *int
	MaxDays       *int
	IsDefault     *bool
	ProvinceCodes []string
	WardCodes     []string
}

type ZoneFilter struct {
	IncludeDeleted bool
}
