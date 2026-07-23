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
	attributeCode   string
	attributeValues []string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewHpPage validates and creates a new entity from user input.
func NewHpPage(
	name, slug, imageURL string,
	metaTitle, metaDescription *string,
	orderIndex int,
	content json.RawMessage,
	attributeCode string,
	attributeValues []string,
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
	if errs := validateAttributeCode(attributeCode); len(errs) > 0 {
		fields["attributeCode"] = errs
	}
	if errs := validateAttributeValues(attributeValues); len(errs) > 0 {
		fields["attributeValues"] = errs
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
	attributeCode string,
	attributeValues []string,
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
func (h *HpPage) AttributeCode() string     { return h.attributeCode }
func (h *HpPage) AttributeValues() []string { return h.attributeValues }
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

func (h *HpPage) UpdateAttributeCode(attributeCode string) error {
	if errs := validateAttributeCode(attributeCode); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"attributeCode": errs})
	}
	h.attributeCode = attributeCode
	h.updatedAt = time.Now()
	return nil
}

func (h *HpPage) UpdateAttributeValues(attributeValues []string) error {
	if errs := validateAttributeValues(attributeValues); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"attributeValues": errs})
	}
	h.attributeValues = attributeValues
	h.updatedAt = time.Now()
	return nil
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

func validateAttributeCode(code string) []string {
	if code == "" {
		return []string{"attributeCode is required"}
	}
	return nil
}

func validateAttributeValues(values []string) []string {
	if len(values) == 0 {
		return []string{"attributeValues must have at least one value"}
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
	AttributeCode   string
	AttributeValues []string
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
}

type HpPageFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
