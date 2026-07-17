package domain

import (
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type Category struct {
	id              string
	name            string
	slug            string
	groupID         *string
	imageUrl        *string
	metaTitle       *string
	metaDescription *string
	isFeatured      bool
	orderIndex      int
	content         json.RawMessage
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// GroupRef is a lightweight, read-only reference to a group_categories row,
// populated by a LEFT JOIN in this module's own SQL (internal/category/infrastructure)
// rather than importing internal/group — same pattern product uses for its own
// CategoryRef/BrandRef, see internal/product/domain/types.go.
type GroupRef struct {
	ID              string
	Name            string
	Slug            string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
}

// CategoryWithRelations is what read queries (GetAll/GetByID/GetBySlug) return —
// Group is nil when the category has no group_id or its group was deleted.
type CategoryWithRelations struct {
	*Category
	Group *GroupRef
}

// NewCategory validates and creates a new entity from user input.
func NewCategory(
	name, slug string,
	groupID *string,
	imageUrl *string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
) (*Category, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Category{
		name:            name,
		slug:            slug,
		groupID:         groupID,
		imageUrl:        imageUrl,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		content:         content,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateCategory reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateCategory(
	id, name, slug string,
	groupID *string,
	imageUrl *string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Category {
	return &Category{
		id:              id,
		name:            name,
		slug:            slug,
		groupID:         groupID,
		imageUrl:        imageUrl,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		content:         content,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (c *Category) ID() string               { return c.id }
func (c *Category) Name() string             { return c.name }
func (c *Category) Slug() string             { return c.slug }
func (c *Category) GroupID() *string         { return c.groupID }
func (c *Category) ImageURL() *string        { return c.imageUrl }
func (c *Category) MetaTitle() *string       { return c.metaTitle }
func (c *Category) MetaDescription() *string { return c.metaDescription }
func (c *Category) IsFeatured() bool         { return c.isFeatured }
func (c *Category) OrderIndex() int          { return c.orderIndex }
func (c *Category) Content() json.RawMessage { return c.content }
func (c *Category) CreatedAt() time.Time     { return c.createdAt }
func (c *Category) UpdatedAt() time.Time     { return c.updatedAt }
func (c *Category) DeletedAt() *time.Time    { return c.deletedAt }

func (c *Category) IsDeleted() bool {
	return c.deletedAt != nil
}

func (c *Category) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	c.name = name
	c.updatedAt = time.Now()
	return nil
}

func (c *Category) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	c.slug = slug
	c.updatedAt = time.Now()
	return nil
}

func (c *Category) UpdateGroupID(groupID *string) {
	c.groupID = groupID
	c.updatedAt = time.Now()
}

func (c *Category) UpdateImageURL(imageUrl *string) {
	c.imageUrl = imageUrl
	c.updatedAt = time.Now()
}

func (c *Category) UpdateMetaTitle(metaTitle *string) {
	c.metaTitle = metaTitle
	c.updatedAt = time.Now()
}

func (c *Category) UpdateMetaDescription(metaDescription *string) {
	c.metaDescription = metaDescription
	c.updatedAt = time.Now()
}

func (c *Category) SetFeatured(isFeatured bool) {
	c.isFeatured = isFeatured
	c.updatedAt = time.Now()
}

func (c *Category) Reorder(orderIndex int) {
	c.orderIndex = orderIndex
	c.updatedAt = time.Now()
}

func (c *Category) UpdateContent(content json.RawMessage) {
	c.content = content
	c.updatedAt = time.Now()
}

func (c *Category) MarkDeleted(deletedAt time.Time) {
	c.deletedAt = &deletedAt
}

func (c *Category) Restore() {
	c.deletedAt = nil
	c.updatedAt = time.Now()
}

func validateName(name string) []string {
	var errs []string
	if name == "" {
		errs = append(errs, "name is required")
	} else if utf8.RuneCountInString(name) > 100 {
		errs = append(errs, "name must not exceed 100 characters")
	}
	return errs
}

func validateSlug(slug string) []string {
	var errs []string
	if slug == "" {
		errs = append(errs, "slug is required")
	} else if utf8.RuneCountInString(slug) > 100 {
		errs = append(errs, "slug must not exceed 100 characters")
	}
	return errs
}

type CreateCategoryInput struct {
	Name            string
	Slug            string
	GroupID         *string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
	Content         json.RawMessage
}

type UpdateCategoryInput struct {
	ID              string
	Name            *string
	Slug            *string
	GroupID         *string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      *bool
	OrderIndex      *int
	Content         json.RawMessage
}

type CategoryFilter struct {
	Search         string
	GroupID        string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
