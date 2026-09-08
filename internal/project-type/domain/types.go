package domain

import (
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

type ProjectType struct {
	id              string
	name            string
	slug            string
	image           *string
	metaTitle       *string
	metaDescription *string
	isFeatured      bool
	orderIndex      int
	content         json.RawMessage
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// CategoryGroupRef is a lightweight, read-only reference to a group_categories
// row, populated by a nested LEFT JOIN in this module's own SQL — same shape
// internal/category/domain/types.go's GroupRef uses.
type CategoryGroupRef struct {
	ID              string
	Name            string
	Slug            string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
}

// CategoryRef is one category attached to a project type via the
// project_type_category join table, plus its own group (if any) — mirrors
// the old TS mapToDomainWithCategories, which read
// `categories(*, group_categories(*))` for each project_type_category row
// and dropped any category that was itself soft-deleted (see
// modules/project-type/infrastructure/projectTypeRepo.ts). This module's own
// SQL filters deleted categories out at the JOIN instead of post-filtering.
type CategoryRef struct {
	ID              string
	Name            string
	GroupID         *string
	Slug            string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	Group           *CategoryGroupRef
}

// ProjectTypeWithCategories is what read queries (GetAll/GetByID) return —
// Create/Update only ever deal with a plain *ProjectType; the categoryIDs
// relation is passed as a separate parameter, same split
// internal/project/domain/repository.go uses for its own join-table writes.
type ProjectTypeWithCategories struct {
	*ProjectType
	Categories []CategoryRef
}

// NewProjectType validates and creates a new entity from user input.
func NewProjectType(
	name, slug string,
	image, metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
) (*ProjectType, error) {
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
	return &ProjectType{
		name:            name,
		slug:            slug,
		image:           image,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		content:         content,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateProjectType reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateProjectType(
	id, name, slug string,
	image, metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	content json.RawMessage,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *ProjectType {
	return &ProjectType{
		id:              id,
		name:            name,
		slug:            slug,
		image:           image,
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

func (p *ProjectType) ID() string               { return p.id }
func (p *ProjectType) Name() string             { return p.name }
func (p *ProjectType) Slug() string             { return p.slug }
func (p *ProjectType) Image() *string           { return p.image }
func (p *ProjectType) MetaTitle() *string       { return p.metaTitle }
func (p *ProjectType) MetaDescription() *string { return p.metaDescription }
func (p *ProjectType) IsFeatured() bool         { return p.isFeatured }
func (p *ProjectType) OrderIndex() int          { return p.orderIndex }
func (p *ProjectType) Content() json.RawMessage { return p.content }
func (p *ProjectType) CreatedAt() time.Time     { return p.createdAt }
func (p *ProjectType) UpdatedAt() time.Time     { return p.updatedAt }
func (p *ProjectType) DeletedAt() *time.Time    { return p.deletedAt }

func (p *ProjectType) IsDeleted() bool {
	return p.deletedAt != nil
}

// Update applies a partial edit from a form submission: only non-nil fields
// in input are validated and set. Validates and applies in the same order
// the fields appear below, failing fast on the first invalid field (no error
// aggregation) — matches the single call site in
// application.UpdateProjectType, which always submits the whole edit form at
// once. See docs/rfc/2026-08-18-branch-domain-consolidate-update.md for the
// pattern this follows (internal/branch's Update).
//
// Content is not validated, same as Category.UpdateContent.
func (p *ProjectType) Update(input UpdateProjectTypeInput) error {
	// changed tracks whether any field below actually mutated p, so a no-op
	// call (all fields nil, or metaTitle/metaDescription unchanged) leaves
	// updatedAt untouched — matching the old per-field UpdateX() behavior.
	changed := false

	if input.Name != nil {
		if errs := validateName(*input.Name); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
		}
		p.name = *input.Name
		changed = true
	}
	if input.Slug != nil {
		if errs := validateSlug(*input.Slug); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
		}
		p.slug = *input.Slug
		changed = true
	}
	if input.Image != nil {
		p.image = input.Image
		changed = true
	}
	if input.MetaTitle != nil && !seo.Unchanged(p.metaTitle, input.MetaTitle) {
		if errs := seo.ValidateMetaTitle(input.MetaTitle); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
		}
		p.metaTitle = input.MetaTitle
		changed = true
	}
	if input.MetaDescription != nil && !seo.Unchanged(p.metaDescription, input.MetaDescription) {
		if errs := seo.ValidateMetaDescription(input.MetaDescription); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
		}
		p.metaDescription = input.MetaDescription
		changed = true
	}
	if input.Content != nil {
		p.content = input.Content
		changed = true
	}

	if changed {
		p.updatedAt = time.Now()
	}
	return nil
}

func (p *ProjectType) SetFeatured(isFeatured bool) {
	p.isFeatured = isFeatured
	p.updatedAt = time.Now()
}

func (p *ProjectType) Reorder(orderIndex int) {
	p.orderIndex = orderIndex
	p.updatedAt = time.Now()
}

func (p *ProjectType) MarkDeleted(deletedAt time.Time) {
	p.deletedAt = &deletedAt
}

func (p *ProjectType) Restore() {
	p.deletedAt = nil
	p.updatedAt = time.Now()
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

type CreateProjectTypeInput struct {
	Name            string
	Slug            string
	Image           *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
	Content         json.RawMessage
	CategoryIDs     []string
}

type UpdateProjectTypeInput struct {
	ID              string
	Name            *string
	Slug            *string
	Image           *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      *bool
	OrderIndex      *int
	Content         json.RawMessage
	// CategoryIDs: nil means "leave relations untouched", a non-nil pointer
	// (including one pointing at an empty slice) means "replace all
	// relations with this set" — mirrors project's ServiceIDs *[]string
	// convention, see internal/project/domain/types.go.
	CategoryIDs *[]string
}

type ProjectTypeFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
