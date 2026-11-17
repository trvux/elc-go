package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Completed-date precision: the article may only know the month or year, and
// a month-precision date must never be shown/emitted as a full day.
const (
	PrecisionDay     = "day"
	PrecisionMonth   = "month"
	PrecisionYear    = "year"
	PrecisionUnknown = "unknown"
)

const (
	ClientKindPerson   = "person"
	ClientKindBusiness = "business"
)

// ImageRole values for ImageAsset.Role.
var imageRoles = map[string]bool{"": true, "before": true, "during": true, "after": true}

// ProjectFacts are the typed, per-project facts (SEO + GEO: concrete numbers
// and place instead of free text). Every numeric field is nil when unknown —
// a nil field is never rendered and never emitted to JSON-LD, so no number
// is ever invented. Equipment condition is NOT here: it is derived from
// project_category.condition.
//
// It is replaced as a whole on update (nil *ProjectFacts in UpdateProjectInput
// means "untouched"), which also makes "clear this number" expressible.
type ProjectFacts struct {
	// Province/Ward are the CURRENT administrative names. Location (on
	// Project) stays as the original article's free-text/historical name.
	Province           string
	Ward               string
	CompletedPrecision string

	// ClientKind nil is treated as "person" (the safe default).
	ClientKind        *string
	ClientConsentAt   *time.Time
	ClientConsentNote string

	IndoorUnits    *int
	CapacityHP     *float64
	AirflowM3H     *int
	AreaM2         *float64
	Floors         *int
	DurationDays   *int
	WarrantyMonths *int

	Brands []string
}

// normalize applies defaults so a zero-value ProjectFacts is storable.
func (f ProjectFacts) normalize() ProjectFacts {
	if f.CompletedPrecision == "" {
		f.CompletedPrecision = PrecisionUnknown
	}
	if f.Brands == nil {
		f.Brands = []string{}
	}
	cleaned := f.Brands[:0:0]
	seen := map[string]bool{}
	for _, b := range f.Brands {
		b = strings.TrimSpace(b)
		if b != "" && !seen[strings.ToLower(b)] {
			seen[strings.ToLower(b)] = true
			cleaned = append(cleaned, b)
		}
	}
	f.Brands = cleaned
	f.Province = strings.TrimSpace(f.Province)
	f.Ward = strings.TrimSpace(f.Ward)
	return f
}

func positiveInt(name string, v *int, errs map[string][]string) {
	if v != nil && *v <= 0 {
		errs[name] = []string{name + " must be greater than 0"}
	}
}

func positiveFloat(name string, v *float64, errs map[string][]string) {
	if v != nil && *v <= 0 {
		errs[name] = []string{name + " must be greater than 0"}
	}
}

// validate checks the (already normalized) facts.
func (f ProjectFacts) validate() error {
	errs := map[string][]string{}

	switch f.CompletedPrecision {
	case PrecisionDay, PrecisionMonth, PrecisionYear, PrecisionUnknown:
	default:
		errs["completedPrecision"] = []string{"completedPrecision must be 'day', 'month', 'year' or 'unknown'"}
	}
	if f.ClientKind != nil && *f.ClientKind != ClientKindPerson && *f.ClientKind != ClientKindBusiness {
		errs["clientKind"] = []string{"clientKind must be 'person' or 'business'"}
	}
	positiveInt("indoorUnits", f.IndoorUnits, errs)
	positiveFloat("capacityHp", f.CapacityHP, errs)
	positiveInt("airflowM3h", f.AirflowM3H, errs)
	positiveFloat("areaM2", f.AreaM2, errs)
	positiveInt("floors", f.Floors, errs)
	positiveInt("durationDays", f.DurationDays, errs)
	positiveInt("warrantyMonths", f.WarrantyMonths, errs)
	if utf8.RuneCountInString(f.Province) > 100 {
		errs["province"] = []string{"province must not exceed 100 characters"}
	}
	if utf8.RuneCountInString(f.Ward) > 100 {
		errs["ward"] = []string{"ward must not exceed 100 characters"}
	}
	for _, b := range f.Brands {
		if utf8.RuneCountInString(b) > 50 {
			errs["brands"] = []string{"each brand must not exceed 50 characters"}
			break
		}
	}

	if len(errs) > 0 {
		return apperr.NewValidationError("validation failed", errs)
	}
	return nil
}

// ClientNamePublic reports whether the client name may be shown to visitors.
// A business name is always public; a natural person's name is personal data
// and is shown only once consent is on record (nil kind counts as person).
func (p *Project) ClientNamePublic() bool {
	if p.clientName == "" {
		return false
	}
	if p.facts.ClientKind != nil && *p.facts.ClientKind == ClientKindBusiness {
		return true
	}
	return p.facts.ClientConsentAt != nil
}

// PublishGaps lists what is still missing before the project may be
// published. It is checked only on the unpublished -> published transition
// (and on create with isPublished), never on edits to an already published
// row — most existing rows would fail it (66/71 have <3 images, 0/191 have
// alt text), which would lock out editing them while they are being
// backfilled. serviceCount/categoryCount are the relation counts AFTER the
// pending write.
func PublishGaps(p *Project, serviceCount, categoryCount int) []string {
	var gaps []string
	add := func(format string, a ...any) { gaps = append(gaps, fmt.Sprintf(format, a...)) }

	if p.projectTypeID == nil || *p.projectTypeID == "" {
		add("chọn loại công trình")
	}
	if serviceCount < 1 {
		add("chọn ít nhất 1 hệ thống/dịch vụ")
	}
	if p.facts.Province == "" {
		add("nhập tỉnh/thành")
	}
	switch {
	case p.completedAt != nil && p.facts.CompletedPrecision == PrecisionUnknown:
		add("chọn độ chính xác của ngày hoàn thành")
	case p.completedAt == nil && p.facts.CompletedPrecision != PrecisionUnknown:
		add("nhập ngày hoàn thành (hoặc chọn 'chưa rõ')")
	}
	if len(p.facts.Brands) < 1 {
		add("chọn ít nhất 1 hãng thiết bị")
	}
	if categoryCount < 1 {
		add("chọn thiết bị và tình trạng (mới/qua sử dụng)")
	}
	if p.facts.IndoorUnits == nil {
		add("nhập số lượng thiết bị")
	}
	if len(p.images) < 3 {
		add("cần ít nhất 3 ảnh (hiện có %d)", len(p.images))
	}
	missingAlt := 0
	for _, img := range p.images {
		if strings.TrimSpace(img.Alt) == "" {
			missingAlt++
		}
	}
	if missingAlt > 0 {
		add("%d ảnh chưa có mô tả (alt)", missingAlt)
	}
	return gaps
}

// ValidateImages checks the structured fields of a gallery (role values).
// Alt is required only by the publish gate, not here.
func ValidateImages(images []ImageAsset) []string {
	for _, img := range images {
		if !imageRoles[img.Role] {
			return []string{"image role must be 'before', 'during' or 'after'"}
		}
	}
	return nil
}
