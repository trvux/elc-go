package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// SystemPage represents SEO metadata for one of a fixed, admin-seeded set of
// hub pages (home, tin-tuc, du-an, ...). Rows are never created or deleted
// through this module — only meta_title/meta_description are editable.
type SystemPage struct {
	id              string
	name            string
	slug            string
	metaTitle       *string
	metaDescription *string
	createdAt       time.Time
	updatedAt       time.Time
}

// RehydrateSystemPage reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateSystemPage(
	id, name, slug string,
	metaTitle, metaDescription *string,
	createdAt, updatedAt time.Time,
) *SystemPage {
	return &SystemPage{
		id:              id,
		name:            name,
		slug:            slug,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

func (p *SystemPage) ID() string               { return p.id }
func (p *SystemPage) Name() string             { return p.name }
func (p *SystemPage) Slug() string             { return p.slug }
func (p *SystemPage) MetaTitle() *string       { return p.metaTitle }
func (p *SystemPage) MetaDescription() *string { return p.metaDescription }
func (p *SystemPage) CreatedAt() time.Time     { return p.createdAt }
func (p *SystemPage) UpdatedAt() time.Time     { return p.updatedAt }

// UpdateMeta uses the same length limits as every other content module —
// see internal/platform/seo. Only re-validates a field against the limit if
// it's actually changing — the edit form resends both fields together, so a
// pre-existing title that already exceeds the limit would otherwise block
// saving a fix to the description alone (or vice versa).
func (p *SystemPage) UpdateMeta(metaTitle, metaDescription *string) error {
	fields := map[string][]string{}
	if !seo.Unchanged(p.metaTitle, metaTitle) {
		if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
			fields["meta_title"] = errs
		}
	}
	if !seo.Unchanged(p.metaDescription, metaDescription) {
		if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
			fields["meta_description"] = errs
		}
	}
	if len(fields) > 0 {
		return apperr.NewValidationError("validation failed", fields)
	}

	p.metaTitle = metaTitle
	p.metaDescription = metaDescription
	p.updatedAt = time.Now()
	return nil
}

type UpdateSystemPageInput struct {
	ID              string
	MetaTitle       *string
	MetaDescription *string
}
