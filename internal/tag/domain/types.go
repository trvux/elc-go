package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Tag is a cross-cutting taxonomy entity shared by news/products/projects —
// see internal/tag/migrations/000001 for why it exists (missing CMS-standard
// pattern, doubles as the real news<->product/project relation).
type Tag struct {
	id        string
	name      string
	slug      string
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

func NewTag(name, slug string) (*Tag, error) {
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
	return &Tag{name: name, slug: slug, createdAt: now, updatedAt: now}, nil
}

// RehydrateTag reconstructs from a trusted DB row — no validation. Only the
// infrastructure layer should call this.
func RehydrateTag(id, name, slug string, createdAt, updatedAt time.Time, deletedAt *time.Time) *Tag {
	return &Tag{id: id, name: name, slug: slug, createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt}
}

func (t *Tag) ID() string            { return t.id }
func (t *Tag) Name() string          { return t.name }
func (t *Tag) Slug() string          { return t.slug }
func (t *Tag) CreatedAt() time.Time  { return t.createdAt }
func (t *Tag) UpdatedAt() time.Time  { return t.updatedAt }
func (t *Tag) DeletedAt() *time.Time { return t.deletedAt }

func (t *Tag) IsDeleted() bool { return t.deletedAt != nil }

func (t *Tag) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	t.name = name
	t.updatedAt = time.Now()
	return nil
}

func (t *Tag) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	t.slug = slug
	t.updatedAt = time.Now()
	return nil
}

func (t *Tag) MarkDeleted(deletedAt time.Time) { t.deletedAt = &deletedAt }

func (t *Tag) Restore() {
	t.deletedAt = nil
	t.updatedAt = time.Now()
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

type CreateTagInput struct {
	Name string
	Slug string
}

type UpdateTagInput struct {
	ID   string
	Name *string
	Slug *string
}

type TagFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
