package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Seo is the unified SEO metadata shape stored as jsonb, replacing the old
// flat MetaTitle/MetaDescription pair (kept alongside during the migration).
// Duplicated per-module rather than shared, same reasoning as
// CategoryRef/BrandRef in catalog/domain/types.go.
type Seo struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Noindex     bool    `json:"noindex,omitempty"`
}

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
	image            *string
	metaTitle        *string
	metaDescription  *string
	seo              Seo
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
	image, metaTitle, metaDescription *string,
	seo Seo,
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
		image:            image,
		metaTitle:        metaTitle,
		metaDescription:  metaDescription,
		seo:              seo,
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
	image, metaTitle, metaDescription *string,
	seo Seo,
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
		image: image, metaTitle: metaTitle, metaDescription: metaDescription, seo: seo,
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
func (s *Service) Image() *string            { return s.image }
func (s *Service) MetaTitle() *string        { return s.metaTitle }
func (s *Service) MetaDescription() *string  { return s.metaDescription }
func (s *Service) Seo() Seo                  { return s.seo }
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

func (s *Service) UpdateTitle(title string) error {
	if errs := validateTitle(title); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"title": errs})
	}
	s.title = title
	s.updatedAt = time.Now()
	return nil
}

func (s *Service) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	s.slug = slug
	s.updatedAt = time.Now()
	return nil
}

func (s *Service) UpdateGroupID(groupID *string) {
	s.groupID = groupID
	s.updatedAt = time.Now()
}

func (s *Service) UpdateCategoryID(categoryID *string) {
	s.categoryID = categoryID
	s.updatedAt = time.Now()
}

// UpdatePricing takes both fields together (not two separate setters) on
// purpose: SalePrice() depends on both, so the caller (application layer)
// must resolve "unchanged" fields to their current value before calling
// this — see application/update_service.go. That's what prevents the old
// stale-sale_price bug from being possible here.
func (s *Service) UpdatePricing(originalPrice *int64, discountPercent *int) {
	s.originalPrice = originalPrice
	s.discountPercent = discountPercent
	s.updatedAt = time.Now()
}

func (s *Service) UpdatePriceDisplayText(text *string) {
	s.priceDisplayText = text
	s.updatedAt = time.Now()
}

func (s *Service) SetLabels(labels []string) {
	s.labels = labels
	s.updatedAt = time.Now()
}

func (s *Service) UpdateDescription(description *string) {
	s.description = description
	s.updatedAt = time.Now()
}

func (s *Service) UpdateContent(content json.RawMessage) {
	s.content = content
	s.updatedAt = time.Now()
}

func (s *Service) UpdateImage(image *string) {
	s.image = image
	s.updatedAt = time.Now()
}

func (s *Service) UpdateMetaTitle(metaTitle *string) {
	s.metaTitle = metaTitle
	s.updatedAt = time.Now()
}

func (s *Service) UpdateMetaDescription(metaDescription *string) {
	s.metaDescription = metaDescription
	s.updatedAt = time.Now()
}

func (s *Service) UpdateSeo(seo Seo) {
	s.seo = seo
	s.updatedAt = time.Now()
}

func (s *Service) SetFeatured(isFeatured bool) {
	s.isFeatured = isFeatured
	s.updatedAt = time.Now()
}

func (s *Service) SetPublished(isPublished bool) {
	s.isPublished = isPublished
	s.updatedAt = time.Now()
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
	Image            *string
	MetaTitle        *string
	MetaDescription  *string
	Seo              Seo
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
	Image            *string
	MetaTitle        *string
	MetaDescription  *string
	Seo              *Seo
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
