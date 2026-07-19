package domain

import (
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type Group struct {
	id              string
	name            string
	slug            string
	imageUrl        *string
	metaTitle       *string
	metaDescription *string
	isFeatured      bool
	isHidden        bool
	orderIndex      int
	content         json.RawMessage
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewGroup validates and creates a new entity from user input.
func NewGroup(
	name, slug string,
	imageUrl *string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	isHidden bool,
	orderIndex int,
	content json.RawMessage,
) (*Group, error) {
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
	return &Group{
		name:            name,
		slug:            slug,
		imageUrl:        imageUrl,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		isHidden:        isHidden,
		orderIndex:      orderIndex,
		content:         content,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateGroup reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateGroup(
	id, name, slug string,
	imageUrl *string,
	metaTitle, metaDescription *string,
	isFeatured bool,
	isHidden bool,
	orderIndex int,
	content json.RawMessage,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Group {
	return &Group{
		id:              id,
		name:            name,
		slug:            slug,
		imageUrl:        imageUrl,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		isHidden:        isHidden,
		orderIndex:      orderIndex,
		content:         content,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (g *Group) ID() string               { return g.id }
func (g *Group) Name() string             { return g.name }
func (g *Group) Slug() string             { return g.slug }
func (g *Group) ImageURL() *string        { return g.imageUrl }
func (g *Group) MetaTitle() *string       { return g.metaTitle }
func (g *Group) MetaDescription() *string { return g.metaDescription }
func (g *Group) IsFeatured() bool         { return g.isFeatured }
func (g *Group) IsHidden() bool           { return g.isHidden }
func (g *Group) OrderIndex() int          { return g.orderIndex }
func (g *Group) Content() json.RawMessage { return g.content }
func (g *Group) CreatedAt() time.Time     { return g.createdAt }
func (g *Group) UpdatedAt() time.Time     { return g.updatedAt }
func (g *Group) DeletedAt() *time.Time    { return g.deletedAt }

func (g *Group) IsDeleted() bool {
	return g.deletedAt != nil
}

func (g *Group) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	g.name = name
	g.updatedAt = time.Now()
	return nil
}

func (g *Group) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	g.slug = slug
	g.updatedAt = time.Now()
	return nil
}

func (g *Group) UpdateImageURL(imageUrl *string) {
	g.imageUrl = imageUrl
	g.updatedAt = time.Now()
}

func (g *Group) UpdateMetaTitle(metaTitle *string) {
	g.metaTitle = metaTitle
	g.updatedAt = time.Now()
}

func (g *Group) UpdateMetaDescription(metaDescription *string) {
	g.metaDescription = metaDescription
	g.updatedAt = time.Now()
}

func (g *Group) SetFeatured(isFeatured bool) {
	g.isFeatured = isFeatured
	g.updatedAt = time.Now()
}

func (g *Group) SetHidden(isHidden bool) {
	g.isHidden = isHidden
	g.updatedAt = time.Now()
}

func (g *Group) Reorder(orderIndex int) {
	g.orderIndex = orderIndex
	g.updatedAt = time.Now()
}

func (g *Group) UpdateContent(content json.RawMessage) {
	g.content = content
	g.updatedAt = time.Now()
}

func (g *Group) MarkDeleted(deletedAt time.Time) {
	g.deletedAt = &deletedAt
}

func (g *Group) Restore() {
	g.deletedAt = nil
	g.updatedAt = time.Now()
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

type CreateGroupInput struct {
	Name            string
	Slug            string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	IsHidden        bool
	OrderIndex      int
	Content         json.RawMessage
}

type UpdateGroupInput struct {
	ID              string
	Name            *string
	Slug            *string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      *bool
	IsHidden        *bool
	OrderIndex      *int
	Content         json.RawMessage
}

type GroupFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
