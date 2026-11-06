package domain

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
	"github.com/trvux/elc-go/internal/platform/titlealign"
)

// ImageAsset re-exports the shared media type — see product/domain/types.go's
// identical alias for why this is centralized rather than duplicated.
type ImageAsset = media.ImageAsset

type Service struct {
	id              string
	title           string
	titleAlign      string
	slug            string
	labels          []string
	line            string
	briefPrompts    []string
	description     *string
	content         json.RawMessage
	images          []ImageAsset
	metaTitle       *string
	metaDescription *string
	isPublished     bool
	orderIndex      int
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// ServiceWithRelations is what read queries (GetAll/GetByID/GetBySlug) return
// — a Service (kept as a wrapper for the read queries). Create/Update
// only ever deal with a plain *Service.
type ServiceWithRelations struct {
	*Service
	// CategoryIDs are the product categories this service applies to
	// (service_categories), loaded by the read queries.
	CategoryIDs []string
}

// TitleAlignLeft/Center/Right re-export platform/titlealign — see its doc
// comment.
const (
	TitleAlignLeft   = titlealign.Left
	TitleAlignCenter = titlealign.Center
	TitleAlignRight  = titlealign.Right
)

// The two offerings a service can belong to. Drives the hub section, the
// call-to-action wording and a tracking dimension (replaces service groups).
const (
	LineThiCong = "thi-cong" // system installation projects: survey, then quote
	LineDichVu  = "dich-vu"  // services on air conditioners the customer already has
)

// ValidLine reports whether line is one of the two known offerings.
func ValidLine(line string) bool {
	return line == LineThiCong || line == LineDichVu
}

// NewService validates and creates a new entity from user input.
func NewService(
	title, slug string,
	titleAlign string,
	labels []string,
	line string,
	briefPrompts []string,
	description *string,
	content json.RawMessage,
	images []ImageAsset,
	metaTitle, metaDescription *string,
	isPublished bool,
	orderIndex int,
) (*Service, error) {
	fields := map[string][]string{}

	if errs := validateTitle(title); len(errs) > 0 {
		fields["title"] = errs
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
	titleAlign = titlealign.OrDefault(titleAlign, titlealign.Left)
	if !titlealign.Valid(titleAlign) {
		fields["titleAlign"] = []string{"titleAlign must be 'left', 'center' or 'right'"}
	}
	if line == "" {
		line = LineDichVu
	}
	if !ValidLine(line) {
		fields["line"] = []string{"line must be 'thi-cong' or 'dich-vu'"}
	}
	briefPrompts = cleanBriefPrompts(briefPrompts)

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Service{
		title:           title,
		titleAlign:      titleAlign,
		slug:            slug,
		labels:          labels,
		line:            line,
		briefPrompts:    briefPrompts,
		description:     description,
		content:         content,
		images:          images,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateService reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateService(
	id, title string,
	titleAlign string,
	slug string,
	labels []string,
	line string,
	briefPrompts []string,
	description *string,
	content json.RawMessage,
	images []ImageAsset,
	metaTitle, metaDescription *string,
	isPublished bool,
	orderIndex int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Service {
	return &Service{
		id: id, title: title, titleAlign: titleAlign, slug: slug,
		labels: labels,
		line:   line, briefPrompts: briefPrompts,
		description: description, content: content,
		images: images, metaTitle: metaTitle, metaDescription: metaDescription,
		isPublished: isPublished, orderIndex: orderIndex,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (s *Service) ID() string               { return s.id }
func (s *Service) Title() string            { return s.title }
func (s *Service) TitleAlign() string       { return s.titleAlign }
func (s *Service) Slug() string             { return s.slug }
func (s *Service) Labels() []string         { return s.labels }
func (s *Service) Line() string             { return s.line }
func (s *Service) BriefPrompts() []string   { return s.briefPrompts }
func (s *Service) Description() *string     { return s.description }
func (s *Service) Content() json.RawMessage { return s.content }
func (s *Service) Images() []ImageAsset     { return s.images }
func (s *Service) MetaTitle() *string       { return s.metaTitle }
func (s *Service) MetaDescription() *string { return s.metaDescription }
func (s *Service) IsPublished() bool        { return s.isPublished }
func (s *Service) OrderIndex() int          { return s.orderIndex }
func (s *Service) CreatedAt() time.Time     { return s.createdAt }
func (s *Service) UpdatedAt() time.Time     { return s.updatedAt }
func (s *Service) DeletedAt() *time.Time    { return s.deletedAt }

func (s *Service) IsDeleted() bool {
	return s.deletedAt != nil
}

// Update applies a partial edit in one call — consolidated from 13 separate
// per-field UpdateX()/SetX() methods (UpdateTitle, UpdateSlug, UpdateGroupID,
// UpdateCategoryID, UpdatePricing, UpdatePriceDisplayText, SetLabels,
// UpdateDescription, UpdateContent, UpdateImages, UpdateMetaTitle,
// UpdateMetaDescription, SetFeatured, SetPublished), each of which had
// exactly one call site (application.UpdateService, confirmed by grep before
// this change) — same rationale and pattern as branch's consolidation, see
// docs/rfc/2026-08-18-branch-domain-consolidate-update.md and CLAUDE.md's
// "apply lazily, whenever a module's domain entity is touched anyway"
// convention (triggered here by the 3.8 pricing-validation fix touching
// UpdatePricing). changed tracks whether any field actually mutated s, so a
// no-op call (all fields nil, or metaTitle/metaDescription resent unchanged)
// leaves updatedAt untouched, matching the old per-method behavior.
// OrderIndex is excluded from `changed`: Reorder bumps updatedAt itself.
func (s *Service) Update(input UpdateServiceInput) error {
	changed := false

	if input.Title != nil {
		if errs := validateTitle(*input.Title); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"title": errs})
		}
		s.title = *input.Title
		changed = true
	}
	if input.TitleAlign != nil {
		if !titlealign.Valid(*input.TitleAlign) {
			return apperr.NewValidationError("validation failed", map[string][]string{"titleAlign": {"titleAlign must be 'left', 'center' or 'right'"}})
		}
		s.titleAlign = *input.TitleAlign
		changed = true
	}
	if input.Slug != nil {
		if errs := validateSlug(*input.Slug); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
		}
		s.slug = *input.Slug
		changed = true
	}
	if input.Labels != nil {
		s.labels = input.Labels
		changed = true
	}
	if input.Line != nil {
		if !ValidLine(*input.Line) {
			return apperr.NewValidationError("validation failed", map[string][]string{"line": {"line must be 'thi-cong' or 'dich-vu'"}})
		}
		s.line = *input.Line
		changed = true
	}
	// nil means "not part of this request"; an empty non-nil slice clears the list.
	if input.BriefPrompts != nil {
		s.briefPrompts = cleanBriefPrompts(input.BriefPrompts)
		changed = true
	}
	if input.Description != nil {
		s.description = input.Description
		changed = true
	}
	if input.Content != nil {
		s.content = input.Content
		changed = true
	}
	if input.Images != nil {
		s.images = input.Images
		changed = true
	}
	if input.MetaTitle != nil && !seo.Unchanged(s.metaTitle, input.MetaTitle) {
		if errs := seo.ValidateMetaTitle(input.MetaTitle); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
		}
		s.metaTitle = input.MetaTitle
		changed = true
	}
	if input.MetaDescription != nil && !seo.Unchanged(s.metaDescription, input.MetaDescription) {
		if errs := seo.ValidateMetaDescription(input.MetaDescription); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
		}
		s.metaDescription = input.MetaDescription
		changed = true
	}
	if input.IsPublished != nil {
		s.isPublished = *input.IsPublished
		changed = true
	}
	if input.OrderIndex != nil {
		s.Reorder(*input.OrderIndex)
	}
	if changed {
		s.updatedAt = time.Now()
	}
	return nil
}

func (s *Service) Reorder(orderIndex int) {
	s.orderIndex = orderIndex
	s.updatedAt = time.Now()
}

func (s *Service) MarkDeleted(deletedAt time.Time) {
	s.deletedAt = &deletedAt
}

func (s *Service) Restore() {
	s.deletedAt = nil
	s.updatedAt = time.Now()
}

// cleanBriefPrompts trims each prompt and drops blank ones, so an admin textarea
// with stray empty lines never produces empty bullets or empty lines in the
// Zalo message. Always returns a non-nil slice (the column is NOT NULL).
func cleanBriefPrompts(prompts []string) []string {
	cleaned := make([]string, 0, len(prompts))
	for _, p := range prompts {
		if t := strings.TrimSpace(p); t != "" {
			cleaned = append(cleaned, t)
		}
	}
	return cleaned
}

func validateTitle(title string) []string {
	if title == "" {
		return []string{"title is required"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"slug is required"}
	}
	return nil
}

type CreateServiceInput struct {
	Title        string
	TitleAlign   string
	Slug         string
	Labels       []string
	Line         string
	BriefPrompts []string
	// CategoryIDs links the service to product categories, applied after create.
	CategoryIDs     []string
	Description     *string
	Content         json.RawMessage
	Images          []ImageAsset
	MetaTitle       *string
	MetaDescription *string
	IsPublished     bool
	OrderIndex      int
}

type UpdateServiceInput struct {
	ID           string
	Title        *string
	TitleAlign   *string
	Slug         *string
	Labels       []string
	Line         *string
	BriefPrompts []string
	// CategoryIDs: nil leaves the links alone; an empty non-nil slice clears them.
	CategoryIDs     []string
	Description     *string
	Content         json.RawMessage
	Images          []ImageAsset
	MetaTitle       *string
	MetaDescription *string
	IsPublished     *bool
	OrderIndex      *int
}

type ServiceFilter struct {
	IsPublished    *bool
	Search         string
	IncludeDeleted bool
}
