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
)

// ImageAsset re-exports the shared media type — see catalog/domain/types.go's
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
func NewBranch(
	name, slug, address, phone, email, mapsURL, mapsEmbed string,
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

func (b *Branch) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	b.name = name
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	b.slug = slug
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateAddress(address string) error {
	if errs := validateAddress(address); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"address": errs})
	}
	b.address = address
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdatePhone(phone string) error {
	if errs := validatePhone(phone); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"phone": errs})
	}
	b.phone = phone
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateEmail(email string) error {
	if errs := validateEmail(email); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
	}
	b.email = email
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateMapsURL(mapsURL string) error {
	if errs := validateMapsURL(mapsURL); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"mapsUrl": errs})
	}
	b.mapsURL = mapsURL
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateMapsEmbed(mapsEmbed string) error {
	if errs := validateMapsEmbed(mapsEmbed); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"mapsEmbed": errs})
	}
	b.mapsEmbed = mapsEmbed
	b.updatedAt = time.Now()
	return nil
}

func (b *Branch) UpdateDescription(desc json.RawMessage) {
	b.description = desc
	b.updatedAt = time.Now()
}

func (b *Branch) UpdateImages(images []ImageAsset) {
	b.images = images
	b.updatedAt = time.Now()
}

func (b *Branch) SetPublished(isPublished bool) {
	b.isPublished = isPublished
	b.updatedAt = time.Now()
}

func (b *Branch) Reorder(orderIndex int) {
	b.orderIndex = orderIndex
	b.updatedAt = time.Now()
}

func (b *Branch) UpdateMetaTitle(title *string) {
	b.metaTitle = title
	b.updatedAt = time.Now()
}

func (b *Branch) UpdateMetaDescription(desc *string) {
	b.metaDescription = desc
	b.updatedAt = time.Now()
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
