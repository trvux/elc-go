package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// ImageAsset re-exports the shared media type — see product/domain/types.go's
// identical alias for why this is centralized rather than duplicated.
type ImageAsset = media.ImageAsset

type Service struct {
	id               string
	title            string
	slug             string
	groupID          *string
	categoryID       *string
	originalPrice    *int64
	discountPercent  *int
	priceDisplayText *string
	labels           []string
	description      *string
	content          json.RawMessage
	images           []ImageAsset
	metaTitle        *string
	metaDescription  *string
	isFeatured       bool
	isPublished      bool
	orderIndex       int
	createdAt        time.Time
	updatedAt        time.Time
	deletedAt        *time.Time
}

// GroupRef/CategoryRef are lightweight, read-only references to entities
// owned by other modules (service-group, category). Deliberately NOT the
// full ServiceGroup/Category domain types — that would couple this module's
// domain layer to two other modules' internals for two fields (id, name)
// that are all any UI actually reads (confirmed by audit before writing this).
type GroupRef struct {
	ID   string
	Name string
}

type CategoryRef struct {
	ID   string
	Name string
}

// ServiceWithRelations is what read queries (GetAll/GetByID/GetBySlug) return
// — a Service plus the joined group/category display refs. Create/Update
// only ever deal with a plain *Service.
type ServiceWithRelations struct {
	*Service
	Group    *GroupRef
	Category *CategoryRef
}

// NewService validates and creates a new entity from user input.
// salePrice is intentionally not a parameter here — see SalePrice().
func NewService(
	title, slug string,
	groupID, categoryID *string,
	originalPrice *int64,
	discountPercent *int,
	priceDisplayText *string,
	labels []string,
	description *string,
	content json.RawMessage,
	images []ImageAsset,
	metaTitle, metaDescription *string,
	isFeatured, isPublished bool,
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
	if errs := validatePricing(originalPrice, discountPercent); len(errs) > 0 {
		fields["discountPercent"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Service{
		title:            title,
		slug:             slug,
		groupID:          groupID,
		categoryID:       categoryID,
		originalPrice:    originalPrice,
		discountPercent:  discountPercent,
		priceDisplayText: priceDisplayText,
		labels:           labels,
		description:      description,
		content:          content,
		images:           images,
		metaTitle:        metaTitle,
		metaDescription:  metaDescription,
		isFeatured:       isFeatured,
		isPublished:      isPublished,
		orderIndex:       orderIndex,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// RehydrateService reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateService(
	id, title, slug string,
	groupID, categoryID *string,
	originalPrice *int64,
	discountPercent *int,
	priceDisplayText *string,
	labels []string,
	description *string,
	content json.RawMessage,
	images []ImageAsset,
	metaTitle, metaDescription *string,
	isFeatured, isPublished bool,
	orderIndex int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Service {
	return &Service{
		id: id, title: title, slug: slug,
		groupID: groupID, categoryID: categoryID,
		originalPrice: originalPrice, discountPercent: discountPercent,
		priceDisplayText: priceDisplayText, labels: labels,
		description: description, content: content,
		images: images, metaTitle: metaTitle, metaDescription: metaDescription,
		isFeatured: isFeatured, isPublished: isPublished, orderIndex: orderIndex,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (s *Service) ID() string                { return s.id }
func (s *Service) Title() string             { return s.title }
func (s *Service) Slug() string              { return s.slug }
func (s *Service) GroupID() *string          { return s.groupID }
func (s *Service) CategoryID() *string       { return s.categoryID }
func (s *Service) OriginalPrice() *int64     { return s.originalPrice }
func (s *Service) DiscountPercent() *int     { return s.discountPercent }
func (s *Service) PriceDisplayText() *string { return s.priceDisplayText }
func (s *Service) Labels() []string          { return s.labels }
func (s *Service) Description() *string      { return s.description }
func (s *Service) Content() json.RawMessage  { return s.content }
func (s *Service) Images() []ImageAsset      { return s.images }
func (s *Service) MetaTitle() *string        { return s.metaTitle }
func (s *Service) MetaDescription() *string  { return s.metaDescription }
func (s *Service) IsFeatured() bool          { return s.isFeatured }
func (s *Service) IsPublished() bool         { return s.isPublished }
func (s *Service) OrderIndex() int           { return s.orderIndex }
func (s *Service) CreatedAt() time.Time      { return s.createdAt }
func (s *Service) UpdatedAt() time.Time      { return s.updatedAt }
func (s *Service) DeletedAt() *time.Time     { return s.deletedAt }

func (s *Service) IsDeleted() bool {
	return s.deletedAt != nil
}

// SalePrice is always derived, never stored as independent state — the old
// TS code kept a `sale_price` column that two different call sites
// recomputed two different ways (one used the stored value, one recomputed
// client-side), and a partial update could leave it stale. Computing it
// fresh every time removes that whole bug class permanently.
func (s *Service) SalePrice() *int64 {
	if s.originalPrice == nil || s.discountPercent == nil {
		return nil
	}
	original := *s.originalPrice
	discount := int64(*s.discountPercent)
	price := original - (original*discount)/100
	return &price
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
	if input.Slug != nil {
		if errs := validateSlug(*input.Slug); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
		}
		s.slug = *input.Slug
		changed = true
	}
	if input.GroupID != nil {
		s.groupID = input.GroupID
		changed = true
	}
	if input.CategoryID != nil {
		s.categoryID = input.CategoryID
		changed = true
	}
	// OriginalPrice/DiscountPercent are resolved together — whichever one
	// isn't part of this request keeps its CURRENT value, so SalePrice() is
	// always computed from a consistent pair. This is what prevents the old
	// TS bug where changing only discountPercent left a stored sale_price
	// stale (see SalePrice's doc comment); folding the merge in here (rather
	// than in the application layer reading getters, as the pre-consolidation
	// UpdatePricing required) means this invariant can never be bypassed by
	// a future call site that forgets to do the merge itself.
	if input.OriginalPrice != nil || input.DiscountPercent != nil {
		originalPrice := s.originalPrice
		if input.OriginalPrice != nil {
			originalPrice = input.OriginalPrice
		}
		discountPercent := s.discountPercent
		if input.DiscountPercent != nil {
			discountPercent = input.DiscountPercent
		}
		if errs := validatePricing(originalPrice, discountPercent); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"discountPercent": errs})
		}
		s.originalPrice = originalPrice
		s.discountPercent = discountPercent
		changed = true
	}
	if input.PriceDisplayText != nil {
		s.priceDisplayText = input.PriceDisplayText
		changed = true
	}
	if input.Labels != nil {
		s.labels = input.Labels
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
	if input.IsFeatured != nil {
		s.isFeatured = *input.IsFeatured
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

// validatePricing rejects a discount/price combination SalePrice() can't
// turn into a sane customer-facing number — a discountPercent outside
// [0, 100] makes SalePrice() negative or higher than originalPrice itself,
// and a negative originalPrice is never a real price. Same "reject at write
// time, don't let bad data reach the storefront" rule
// docs/rfc/2026-08-18-product-data-anomaly-detection.md already applies to
// product's variant prices — service's discount-based pricing had no
// equivalent guard until now.
func validatePricing(originalPrice *int64, discountPercent *int) []string {
	var errs []string
	if originalPrice != nil && *originalPrice < 0 {
		errs = append(errs, "originalPrice must not be negative")
	}
	if discountPercent != nil && (*discountPercent < 0 || *discountPercent > 100) {
		errs = append(errs, "discountPercent must be between 0 and 100")
	}
	return errs
}

type CreateServiceInput struct {
	Title            string
	Slug             string
	GroupID          *string
	CategoryID       *string
	OriginalPrice    *int64
	DiscountPercent  *int
	PriceDisplayText *string
	Labels           []string
	Description      *string
	Content          json.RawMessage
	Images           []ImageAsset
	MetaTitle        *string
	MetaDescription  *string
	IsFeatured       bool
	IsPublished      bool
	OrderIndex       int
}

type UpdateServiceInput struct {
	ID               string
	Title            *string
	Slug             *string
	GroupID          *string
	CategoryID       *string
	OriginalPrice    *int64
	DiscountPercent  *int
	PriceDisplayText *string
	Labels           []string
	Description      *string
	Content          json.RawMessage
	Images           []ImageAsset
	MetaTitle        *string
	MetaDescription  *string
	IsFeatured       *bool
	IsPublished      *bool
	OrderIndex       *int
}

type ServiceFilter struct {
	GroupID        *string
	CategoryID     *string
	IsFeatured     *bool
	IsPublished    *bool
	Search         string
	IncludeDeleted bool
}
