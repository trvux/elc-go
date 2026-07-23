package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// HpPage is an admin-managed SEO landing page for one or more values of a
// product attribute (e.g. attributeCode "phan_khuc_hp", attributeValues
// ["1 HP"]) — lists products across every category carrying that value,
// not scoped to one category/brand. AttributeCode is a plain field rather
// than hardcoded to "phan_khuc_hp" so the same mechanism can back a
// landing page for a different attribute later with zero migration.
type HpPage struct {
	id              string
	name            string
	slug            string
	imageURL        string
	metaTitle       *string
	metaDescription *string
	orderIndex      int
	content         json.RawMessage
	attributeCode   *string
	attributeValues []string
	// categoryIDs/brandIDs let a page scope to specific categories and/or
	// brands instead of (or combined with) attributeCode/attributeValues
	// — e.g. "Máy lạnh Daikin" (category=máy lạnh's sub-categories,
	// brand=Daikin), distinct from the plain brand page (all of Daikin's
	// products, whatever categories that spans). All three filters AND
	// together, same as ProductFilter already does.
	categoryIDs []string
	brandIDs    []string
	createdAt   time.Time
	updatedAt   time.Time
	deletedAt   *time.Time
}

// NewHpPage validates and creates a new entity from user input.
func NewHpPage(
	name, slug, imageURL string,
	metaTitle, metaDescription *string,
	orderIndex int,
	content json.RawMessage,
	attributeCode *string,
	attributeValues []string,
	categoryIDs []string,
	brandIDs []string,
) (*HpPage, error) {
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
	if errs := validateHasAnyFilter(attributeCode, attributeValues, categoryIDs, brandIDs); len(errs) > 0 {
		fields["filters"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &HpPage{
		name:            name,
		slug:            slug,
		imageURL:        imageURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
		content:         content,
		attributeCode:   attributeCode,
		attributeValues: attributeValues,
		categoryIDs:     categoryIDs,
		brandIDs:        brandIDs,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateHpPage reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateHpPage(
	id, name, slug, imageURL string,
	metaTitle, metaDescription *string,
	orderIndex int,
	content json.RawMessage,
	attributeCode *string,
	attributeValues []string,
	categoryIDs []string,
	brandIDs []string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *HpPage {
	return &HpPage{
		id:              id,
		name:            name,
		slug:            slug,
		imageURL:        imageURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
		content:         content,
		attributeCode:   attributeCode,
		attributeValues: attributeValues,
		categoryIDs:     categoryIDs,
		brandIDs:        brandIDs,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (h *HpPage) ID() string                { return h.id }
func (h *HpPage) Name() string              { return h.name }
func (h *HpPage) Slug() string              { return h.slug }
func (h *HpPage) ImageURL() string          { return h.imageURL }
func (h *HpPage) MetaTitle() *string        { return h.metaTitle }
func (h *HpPage) MetaDescription() *string  { return h.metaDescription }
func (h *HpPage) OrderIndex() int           { return h.orderIndex }
func (h *HpPage) Content() json.RawMessage  { return h.content }
func (h *HpPage) AttributeCode() *string    { return h.attributeCode }
func (h *HpPage) AttributeValues() []string { return h.attributeValues }
func (h *HpPage) CategoryIDs() []string     { return h.categoryIDs }
func (h *HpPage) BrandIDs() []string        { return h.brandIDs }
func (h *HpPage) CreatedAt() time.Time      { return h.createdAt }
func (h *HpPage) UpdatedAt() time.Time      { return h.updatedAt }
func (h *HpPage) DeletedAt() *time.Time     { return h.deletedAt }

func (h *HpPage) IsDeleted() bool {
	return h.deletedAt != nil
}

func (h *HpPage) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	h.name = name
	h.updatedAt = time.Now()
	return nil
}

func (h *HpPage) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	h.slug = slug
	h.updatedAt = time.Now()
	return nil
}

func (h *HpPage) UpdateImageURL(imageURL string) {
	h.imageURL = imageURL
	h.updatedAt = time.Now()
}

func (h *HpPage) UpdateMetaTitle(metaTitle *string) error {
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
	}
	h.metaTitle = metaTitle
	h.updatedAt = time.Now()
	return nil
}

func (h *HpPage) UpdateMetaDescription(metaDescription *string) error {
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
	}
	h.metaDescription = metaDescription
	h.updatedAt = time.Now()
	return nil
}

func (h *HpPage) Reorder(orderIndex int) {
	h.orderIndex = orderIndex
	h.updatedAt = time.Now()
}

func (h *HpPage) UpdateContent(content json.RawMessage) {
	h.content = content
	h.updatedAt = time.Now()
}

// UpdateAttributeCode/UpdateAttributeValues/UpdateCategoryIDs/UpdateBrandIDs
// don't validate "at least one filter" individually — a caller applying
// several of these in sequence (see application.UpdateHpPage) would trip a
// false positive mid-sequence. That combined check runs once, after all
// requested fields are applied, via HasAnyFilter().
func (h *HpPage) UpdateAttributeCode(attributeCode *string) {
	h.attributeCode = attributeCode
	h.updatedAt = time.Now()
}

func (h *HpPage) UpdateAttributeValues(attributeValues []string) {
	h.attributeValues = attributeValues
	h.updatedAt = time.Now()
}

func (h *HpPage) UpdateCategoryIDs(categoryIDs []string) {
	h.categoryIDs = categoryIDs
	h.updatedAt = time.Now()
}

func (h *HpPage) UpdateBrandIDs(brandIDs []string) {
	h.brandIDs = brandIDs
	h.updatedAt = time.Now()
}

// HasAnyFilter reports whether the page has at least one usable filter —
// an unfiltered landing page would just list every product, which isn't a
// meaningful page. Checked after every create/update.
func (h *HpPage) HasAnyFilter() bool {
	hasAttribute := h.attributeCode != nil && *h.attributeCode != "" && len(h.attributeValues) > 0
	return hasAttribute || len(h.categoryIDs) > 0 || len(h.brandIDs) > 0
}

func (h *HpPage) MarkDeleted(deletedAt time.Time) {
	h.deletedAt = &deletedAt
}

func (h *HpPage) Restore() {
	h.deletedAt = nil
	h.updatedAt = time.Now()
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

func validateHasAnyFilter(attributeCode *string, attributeValues, categoryIDs, brandIDs []string) []string {
	hasAttribute := attributeCode != nil && *attributeCode != "" && len(attributeValues) > 0
	if !hasAttribute && len(categoryIDs) == 0 && len(brandIDs) == 0 {
		return []string{"page must have at least one filter: attribute, category, or brand"}
	}
	return nil
}

type CreateHpPageInput struct {
	Name            string
	Slug            string
	ImageURL        string
	MetaTitle       *string
	MetaDescription *string
	OrderIndex      int
	Content         json.RawMessage
	AttributeCode   *string
	AttributeValues []string
	CategoryIDs     []string
	BrandIDs        []string
}

type UpdateHpPageInput struct {
	ID              string
	Name            *string
	Slug            *string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	OrderIndex      *int
	Content         json.RawMessage
	AttributeCode   *string
	AttributeValues []string
	CategoryIDs     []string
	BrandIDs        []string
}

type HpPageFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
