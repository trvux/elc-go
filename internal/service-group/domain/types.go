package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type ServiceGroup struct {
	id              string
	name            string
	slug            string
	imageURL        *string
	metaTitle       *string
	metaDescription *string
	isFeatured      bool
	orderIndex      int
	categoryIDs     []string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewServiceGroup validates and creates a new entity from user input.
func NewServiceGroup(name, slug string, imageURL, metaTitle, metaDescription *string, isFeatured bool, orderIndex int, categoryIDs []string) (*ServiceGroup, error) {
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
	return &ServiceGroup{
		name:            name,
		slug:            slug,
		imageURL:        imageURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		categoryIDs:     categoryIDs,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateServiceGroup reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateServiceGroup(
	id, name, slug string,
	imageURL, metaTitle, metaDescription *string,
	isFeatured bool,
	orderIndex int,
	categoryIDs []string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *ServiceGroup {
	return &ServiceGroup{
		id:              id,
		name:            name,
		slug:            slug,
		imageURL:        imageURL,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		isFeatured:      isFeatured,
		orderIndex:      orderIndex,
		categoryIDs:     categoryIDs,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (s *ServiceGroup) ID() string               { return s.id }
func (s *ServiceGroup) Name() string             { return s.name }
func (s *ServiceGroup) Slug() string             { return s.slug }
func (s *ServiceGroup) ImageURL() *string        { return s.imageURL }
func (s *ServiceGroup) MetaTitle() *string       { return s.metaTitle }
func (s *ServiceGroup) MetaDescription() *string { return s.metaDescription }
func (s *ServiceGroup) IsFeatured() bool         { return s.isFeatured }
func (s *ServiceGroup) OrderIndex() int          { return s.orderIndex }
func (s *ServiceGroup) CategoryIDs() []string    { return s.categoryIDs }
func (s *ServiceGroup) CreatedAt() time.Time     { return s.createdAt }
func (s *ServiceGroup) UpdatedAt() time.Time     { return s.updatedAt }
func (s *ServiceGroup) DeletedAt() *time.Time    { return s.deletedAt }

func (s *ServiceGroup) IsDeleted() bool {
	return s.deletedAt != nil
}

func (s *ServiceGroup) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	s.name = name
	s.updatedAt = time.Now()
	return nil
}

func (s *ServiceGroup) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	s.slug = slug
	s.updatedAt = time.Now()
	return nil
}

func (s *ServiceGroup) UpdateImageURL(imageURL *string) {
	s.imageURL = imageURL
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) UpdateMetaTitle(metaTitle *string) {
	s.metaTitle = metaTitle
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) UpdateMetaDescription(metaDescription *string) {
	s.metaDescription = metaDescription
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) SetFeatured(isFeatured bool) {
	s.isFeatured = isFeatured
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) Reorder(orderIndex int) {
	s.orderIndex = orderIndex
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) SetCategoryIDs(categoryIDs []string) {
	s.categoryIDs = categoryIDs
	s.updatedAt = time.Now()
}

func (s *ServiceGroup) MarkDeleted(deletedAt time.Time) {
	s.deletedAt = &deletedAt
}

func (s *ServiceGroup) Restore() {
	s.deletedAt = nil
	s.updatedAt = time.Now()
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

type CreateServiceGroupInput struct {
	Name            string
	Slug            string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
	CategoryIDs     []string
}

type UpdateServiceGroupInput struct {
	ID              string
	Name            *string
	Slug            *string
	ImageURL        *string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      *bool
	OrderIndex      *int
	CategoryIDs     []string
}

type ServiceGroupFilter struct {
	IncludeDeleted bool
	IsFeatured     *bool
}
