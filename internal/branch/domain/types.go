package domain

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// ImageAsset re-exports the shared media type — see product/domain/types.go's
// identical alias for why this is centralized rather than duplicated.
type ImageAsset = media.ImageAsset

var slugRegex = regexp.MustCompile("^[a-z0-9-]+$")

type Branch struct {
	id              string
	name            string
	slug            string
	address         string
	phone           string
	email           string
	mapsURL         string
	mapsEmbed       string
	provinceCode    *string
	provinceName    *string
	wardCode        *string
	wardName        *string
	postalCode      *string
	description     json.RawMessage
	images          []ImageAsset
	isPublished     bool
	orderIndex      int
	metaTitle       *string
	metaDescription *string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewBranch validates and creates a new branch entity from user input.
// provinceCode/provinceName/wardCode/wardName/postalCode are optional — they
// reference internal/shippingzone's provinces/wards reference data (by code,
// no DB foreign key across module boundaries) with the human-readable name
// denormalized alongside so shared/lib/seo-schema.ts (frontend) never needs
// a runtime lookup to build a real PostalAddress.
func NewBranch(
	name, slug, address, phone, email, mapsURL, mapsEmbed string,
	provinceCode, provinceName, wardCode, wardName, postalCode *string,
	description json.RawMessage,
	images []ImageAsset,
	isPublished bool,
	orderIndex int,
	metaTitle, metaDescription *string,
) (*Branch, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := validateAddress(address); len(errs) > 0 {
		fields["address"] = errs
	}
	if errs := validatePhone(phone); len(errs) > 0 {
		fields["phone"] = errs
	}
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if errs := validateMapsURL(mapsURL); len(errs) > 0 {
		fields["mapsUrl"] = errs
	}
	if errs := validateMapsEmbed(mapsEmbed); len(errs) > 0 {
		fields["mapsEmbed"] = errs
	}
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		fields["metaTitle"] = errs
	}
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		fields["metaDescription"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Branch{
		name:            name,
		slug:            slug,
		address:         address,
		phone:           phone,
		email:           email,
		mapsURL:         mapsURL,
		mapsEmbed:       mapsEmbed,
		provinceCode:    provinceCode,
		provinceName:    provinceName,
		wardCode:        wardCode,
		wardName:        wardName,
		postalCode:      postalCode,
		description:     description,
		images:          images,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateBranch reconstructs a branch entity from a trusted DB row.
func RehydrateBranch(
	id, name, slug, address, phone, email, mapsURL, mapsEmbed string,
	provinceCode, provinceName, wardCode, wardName, postalCode *string,
	description json.RawMessage,
	images []ImageAsset,
	isPublished bool,
	orderIndex int,
	metaTitle, metaDescription *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Branch {
	return &Branch{
		id:              id,
		name:            name,
		slug:            slug,
		address:         address,
		phone:           phone,
		email:           email,
		mapsURL:         mapsURL,
		mapsEmbed:       mapsEmbed,
		provinceCode:    provinceCode,
		provinceName:    provinceName,
		wardCode:        wardCode,
		wardName:        wardName,
		postalCode:      postalCode,
		description:     description,
		images:          images,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (b *Branch) ID() string                   { return b.id }
func (b *Branch) Name() string                 { return b.name }
func (b *Branch) Slug() string                 { return b.slug }
func (b *Branch) Address() string              { return b.address }
func (b *Branch) Phone() string                { return b.phone }
func (b *Branch) Email() string                { return b.email }
func (b *Branch) MapsURL() string              { return b.mapsURL }
func (b *Branch) MapsEmbed() string            { return b.mapsEmbed }
func (b *Branch) ProvinceCode() *string        { return b.provinceCode }
func (b *Branch) ProvinceName() *string        { return b.provinceName }
func (b *Branch) WardCode() *string            { return b.wardCode }
func (b *Branch) WardName() *string            { return b.wardName }
func (b *Branch) PostalCode() *string          { return b.postalCode }
func (b *Branch) Description() json.RawMessage { return b.description }
func (b *Branch) Images() []ImageAsset         { return b.images }
func (b *Branch) IsPublished() bool            { return b.isPublished }
func (b *Branch) OrderIndex() int              { return b.orderIndex }
func (b *Branch) MetaTitle() *string           { return b.metaTitle }
func (b *Branch) MetaDescription() *string     { return b.metaDescription }
func (b *Branch) CreatedAt() time.Time         { return b.createdAt }
func (b *Branch) UpdatedAt() time.Time         { return b.updatedAt }
func (b *Branch) DeletedAt() *time.Time        { return b.deletedAt }

func (b *Branch) IsDeleted() bool {
	return b.deletedAt != nil
}

func (b *Branch) Reorder(orderIndex int) {
	b.orderIndex = orderIndex
	b.updatedAt = time.Now()
}

// Update applies a partial edit from a form submission: only non-nil fields
// in input are validated and set. Validates and applies in the same order
// the fields appear below, failing fast on the first invalid field (no error
// aggregation) — this matches the single call site in
// application.UpdateBranch, which always submits the whole edit form at once.
//
// OrderIndex goes through Reorder rather than setting orderIndex directly,
// so drag-drop reorder (application.UpdateBranchOrder) and the edit form
// share one path for that field.
func (b *Branch) Update(input UpdateBranchInput) error {
	if input.Name != nil {
		if errs := validateName(*input.Name); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
		}
		b.name = *input.Name
	}
	if input.Slug != nil {
		if errs := validateSlug(*input.Slug); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
		}
		b.slug = *input.Slug
	}
	if input.Address != nil {
		if errs := validateAddress(*input.Address); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"address": errs})
		}
		b.address = *input.Address
	}
	if input.Phone != nil {
		if errs := validatePhone(*input.Phone); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"phone": errs})
		}
		b.phone = *input.Phone
	}
	if input.Email != nil {
		if errs := validateEmail(*input.Email); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
		}
		b.email = *input.Email
	}
	if input.MapsURL != nil {
		if errs := validateMapsURL(*input.MapsURL); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"mapsUrl": errs})
		}
		b.mapsURL = *input.MapsURL
	}
	if input.MapsEmbed != nil {
		if errs := validateMapsEmbed(*input.MapsEmbed); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"mapsEmbed": errs})
		}
		b.mapsEmbed = *input.MapsEmbed
	}
	// Bundled: the frontend's cascading combobox always submits
	// province/ward code+name+postalCode together (see BranchManagement.tsx),
	// so treat "any one present" as "replace the whole location" rather than
	// merging field-by-field.
	if input.ProvinceCode != nil || input.ProvinceName != nil || input.WardCode != nil || input.WardName != nil || input.PostalCode != nil {
		b.provinceCode = input.ProvinceCode
		b.provinceName = input.ProvinceName
		b.wardCode = input.WardCode
		b.wardName = input.WardName
		b.postalCode = input.PostalCode
	}
	if input.Description != nil {
		b.description = input.Description
	}
	if input.Images != nil {
		b.images = input.Images
	}
	if input.IsPublished != nil {
		b.isPublished = *input.IsPublished
	}
	if input.OrderIndex != nil {
		b.Reorder(*input.OrderIndex)
	}
	if input.MetaTitle != nil && !seo.Unchanged(b.metaTitle, input.MetaTitle) {
		if errs := seo.ValidateMetaTitle(input.MetaTitle); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
		}
		b.metaTitle = input.MetaTitle
	}
	if input.MetaDescription != nil && !seo.Unchanged(b.metaDescription, input.MetaDescription) {
		if errs := seo.ValidateMetaDescription(input.MetaDescription); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
		}
		b.metaDescription = input.MetaDescription
	}
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) MarkDeleted(deletedAt time.Time) {
	b.deletedAt = &deletedAt
}

func (b *Branch) Restore() {
	b.deletedAt = nil
	b.updatedAt = time.Now()
}

func validateName(name string) []string {
	if name == "" {
		return []string{"Tên chi nhánh không được để trống"}
	}
	if utf8.RuneCountInString(name) > 100 {
		return []string{"Tên chi nhánh không được quá 100 ký tự"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"Slug không được để trống"}
	}
	if utf8.RuneCountInString(slug) > 100 {
		return []string{"Slug không được quá 100 ký tự"}
	}
	if !slugRegex.MatchString(slug) {
		return []string{"Slug chỉ được chứa chữ thường, số và dấu gạch ngang"}
	}
	return nil
}

func validateAddress(address string) []string {
	if address == "" {
		return []string{"Địa chỉ không được để trống"}
	}
	return nil
}

func validatePhone(phone string) []string {
	if phone == "" {
		return []string{"Số điện thoại không được để trống"}
	}
	return nil
}

func validateEmail(email string) []string {
	if email == "" {
		return []string{"Email không hợp lệ"}
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return []string{"Email không hợp lệ"}
	}
	return nil
}

func validateMapsURL(mapsURL string) []string {
	if mapsURL == "" {
		return []string{"URL bản đồ không hợp lệ"}
	}
	_, err := url.ParseRequestURI(mapsURL)
	if err != nil {
		return []string{"URL bản đồ không hợp lệ"}
	}
	return nil
}

func validateMapsEmbed(mapsEmbed string) []string {
	if mapsEmbed == "" {
		return []string{"Mã nhúng bản đồ không được để trống"}
	}
	return nil
}

type CreateBranchInput struct {
	Name            string
	Slug            string
	Address         string
	Phone           string
	Email           string
	MapsURL         string
	MapsEmbed       string
	ProvinceCode    *string
	ProvinceName    *string
	WardCode        *string
	WardName        *string
	PostalCode      *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsPublished     bool
	OrderIndex      int
	MetaTitle       *string
	MetaDescription *string
}

type UpdateBranchInput struct {
	ID              string
	Name            *string
	Slug            *string
	Address         *string
	Phone           *string
	Email           *string
	MapsURL         *string
	MapsEmbed       *string
	ProvinceCode    *string
	ProvinceName    *string
	WardCode        *string
	WardName        *string
	PostalCode      *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsPublished     *bool
	OrderIndex      *int
	MetaTitle       *string
	MetaDescription *string
}

type BranchFilter struct {
	IsPublished *bool
	Search      string
	Limit       int
	Offset      int
}
