package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

type Brand struct {
	id              string
	name            string
	slug            string
	logoURL         string
	metaTitle       *string
	metaDescription *string
	isFeatured      bool
	orderIndex      int
	content         json.RawMessage
	// warrantyPolicy describes how a warranty claim works for this brand
	// (elc is a reseller, not the manufacturer — it forwards the unit to
	// the brand rather than servicing it itself, and each brand's process
	// differs), shown on every product of this brand rather than repeated
	// per-product free text.
	warrantyPolicy *string
	createdAt      time.Time
	updatedAt      time.Time
	deletedAt      *time.Time
}

// NewBrand validates and creates a new entity from user input.
func NewBrand(
	name, slug, logoURL string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
	warrantyPolicy *string,
) (*Brand, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
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
	return &Brand{
		name:            name,
		slug:            slug,
		logoURL:         logoURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		content:         content,
		warrantyPolicy:  warrantyPolicy,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateBrand reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateBrand(
	id, name, slug, logoURL string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
	warrantyPolicy *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Brand {
	return &Brand{
		id:              id,
		name:            name,
		slug:            slug,
		logoURL:         logoURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		content:         content,
		warrantyPolicy:  warrantyPolicy,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (b *Brand) ID() string               { return b.id }
func (b *Brand) Name() string             { return b.name }
func (b *Brand) Slug() string             { return b.slug }
func (b *Brand) LogoURL() string          { return b.logoURL }
func (b *Brand) MetaTitle() *string       { return b.metaTitle }
func (b *Brand) MetaDescription() *string { return b.metaDescription }
func (b *Brand) IsFeatured() bool         { return b.isFeatured }
func (b *Brand) OrderIndex() int          { return b.orderIndex }
func (b *Brand) Content() json.RawMessage { return b.content }
func (b *Brand) WarrantyPolicy() *string  { return b.warrantyPolicy }
func (b *Brand) CreatedAt() time.Time     { return b.createdAt }
func (b *Brand) UpdatedAt() time.Time     { return b.updatedAt }
func (b *Brand) DeletedAt() *time.Time    { return b.deletedAt }

func (b *Brand) IsDeleted() bool {
	return b.deletedAt != nil
}

func (b *Brand) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	b.name = name
	b.updatedAt = time.Now()
	return nil
}

func (b *Brand) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	b.slug = slug
	b.updatedAt = time.Now()
	return nil
}

func (b *Brand) UpdateLogoURL(logoURL string) {
	b.logoURL = logoURL
	b.updatedAt = time.Now()
}

func (b *Brand) UpdateMetaTitle(metaTitle *string) error {
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
	}
	b.metaTitle = metaTitle
	b.updatedAt = time.Now()
	return nil
}

func (b *Brand) UpdateMetaDescription(metaDescription *string) error {
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
	}
	b.metaDescription = metaDescription
	b.updatedAt = time.Now()
	return nil
}

func (b *Brand) SetFeatured(isFeatured bool) {
	b.isFeatured = isFeatured
	b.updatedAt = time.Now()
}

func (b *Brand) Reorder(orderIndex int) {
	b.orderIndex = orderIndex
	b.updatedAt = time.Now()
}

func (b *Brand) UpdateContent(content json.RawMessage) {
	b.content = content
	b.updatedAt = time.Now()
}

func (b *Brand) UpdateWarrantyPolicy(warrantyPolicy *string) {
	b.warrantyPolicy = warrantyPolicy
	b.updatedAt = time.Now()
}

func (b *Brand) MarkDeleted(deletedAt time.Time) {
	b.deletedAt = &deletedAt
}

func (b *Brand) Restore() {
	b.deletedAt = nil
	b.updatedAt = time.Now()
}

func validateName(name string) []string {
	if name == "" {
		return []string{"name is required"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"slug is required"}
	}
	return nil
}

type CreateBrandInput struct {
	Name            string
	Slug            string
	LogoURL         string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
	Content         json.RawMessage
	WarrantyPolicy  *string
}

type UpdateBrandInput struct {
	ID              string
	Name            *string
	Slug            *string
	LogoURL         *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      *bool
	OrderIndex      *int
	Content         json.RawMessage
	WarrantyPolicy  *string
}

type BrandFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
