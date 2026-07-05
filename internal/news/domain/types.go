package domain

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

var slugRegex = regexp.MustCompile("^[a-z0-9-]+$")

type News struct {
	id              string
	title           string
	slug            string
	image           string
	content         json.RawMessage
	categoryID      *string
	isPublished     bool
	metaTitle       *string
	metaDescription *string
	orderIndex      int
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewNews validates and creates a new entity from user input.
func NewNews(
	title, slug, image string,
	content json.RawMessage,
	categoryID *string,
	isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
) (*News, error) {
	fields := map[string][]string{}

	if errs := validateTitle(title); len(errs) > 0 {
		fields["title"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := validateMetaTitle(metaTitle); len(errs) > 0 {
		fields["metaTitle"] = errs
	}
	if errs := validateMetaDescription(metaDescription); len(errs) > 0 {
		fields["metaDescription"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	if len(content) == 0 {
		content = json.RawMessage(`{}`)
	}

	now := time.Now()
	return &News{
		title:           title,
		slug:            slug,
		image:           image,
		content:         content,
		categoryID:      categoryID,
		isPublished:     isPublished,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateNews reconstructs from a trusted DB row — no validation. Only the
// infrastructure layer should call this.
func RehydrateNews(
	id, title, slug, image string,
	content json.RawMessage,
	categoryID *string,
	isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *News {
	return &News{
		id:              id,
		title:           title,
		slug:            slug,
		image:           image,
		content:         content,
		categoryID:      categoryID,
		isPublished:     isPublished,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (n *News) ID() string               { return n.id }
func (n *News) Title() string            { return n.title }
func (n *News) Slug() string             { return n.slug }
func (n *News) Image() string            { return n.image }
func (n *News) Content() json.RawMessage { return n.content }
func (n *News) CategoryID() *string      { return n.categoryID }
func (n *News) IsPublished() bool        { return n.isPublished }
func (n *News) MetaTitle() *string       { return n.metaTitle }
func (n *News) MetaDescription() *string { return n.metaDescription }
func (n *News) OrderIndex() int          { return n.orderIndex }
func (n *News) CreatedAt() time.Time     { return n.createdAt }
func (n *News) UpdatedAt() time.Time     { return n.updatedAt }
func (n *News) DeletedAt() *time.Time    { return n.deletedAt }

func (n *News) IsDeleted() bool {
	return n.deletedAt != nil
}

func (n *News) UpdateTitle(title string) error {
	if errs := validateTitle(title); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"title": errs})
	}
	n.title = title
	n.updatedAt = time.Now()
	return nil
}

func (n *News) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	n.slug = slug
	n.updatedAt = time.Now()
	return nil
}

func (n *News) UpdateImage(image string) {
	n.image = image
	n.updatedAt = time.Now()
}

func (n *News) UpdateContent(content json.RawMessage) {
	if len(content) == 0 {
		content = json.RawMessage(`{}`)
	}
	n.content = content
	n.updatedAt = time.Now()
}

func (n *News) UpdateCategoryID(categoryID *string) {
	n.categoryID = categoryID
	n.updatedAt = time.Now()
}

func (n *News) SetPublished(isPublished bool) {
	n.isPublished = isPublished
	n.updatedAt = time.Now()
}

func (n *News) UpdateMetaTitle(metaTitle *string) error {
	if errs := validateMetaTitle(metaTitle); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
	}
	n.metaTitle = metaTitle
	n.updatedAt = time.Now()
	return nil
}

func (n *News) UpdateMetaDescription(metaDescription *string) error {
	if errs := validateMetaDescription(metaDescription); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
	}
	n.metaDescription = metaDescription
	n.updatedAt = time.Now()
	return nil
}

func (n *News) Reorder(orderIndex int) {
	n.orderIndex = orderIndex
	n.updatedAt = time.Now()
}

func (n *News) MarkDeleted(deletedAt time.Time) {
	n.deletedAt = &deletedAt
}

func (n *News) Restore() {
	n.deletedAt = nil
	n.updatedAt = time.Now()
}

func validateTitle(title string) []string {
	if title == "" {
		return []string{"Tiêu đề tin tức không được để trống"}
	}
	if len(title) > 200 {
		return []string{"Tiêu đề tin tức không được quá 200 ký tự"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"Slug không được để trống"}
	}
	if len(slug) > 200 {
		return []string{"Slug không được quá 200 ký tự"}
	}
	if !slugRegex.MatchString(slug) {
		return []string{"Slug chỉ được chứa chữ thường, số và dấu gạch ngang"}
	}
	return nil
}

func validateMetaTitle(metaTitle *string) []string {
	if metaTitle != nil && len(*metaTitle) > 70 {
		return []string{"Tiêu đề SEO không nên quá 70 ký tự"}
	}
	return nil
}

func validateMetaDescription(metaDescription *string) []string {
	if metaDescription != nil && len(*metaDescription) > 160 {
		return []string{"Mô tả SEO không nên quá 160 ký tự"}
	}
	return nil
}

type CreateNewsInput struct {
	Title           string
	Slug            string
	Image           string
	Content         json.RawMessage
	CategoryID      *string
	IsPublished     bool
	MetaTitle       *string
	MetaDescription *string
	OrderIndex      int
}

type UpdateNewsInput struct {
	ID              string
	Title           *string
	Slug            *string
	Image           *string
	Content         json.RawMessage
	CategoryID      *string
	IsPublished     *bool
	MetaTitle       *string
	MetaDescription *string
	OrderIndex      *int
}

type NewsFilter struct {
	IsPublished    *bool
	Search         string
	CategoryID     *string
	ExcludeID      *string
	Limit          int
	Offset         int
	IncludeDeleted bool
	SortBy         string // "created_at" | "order_index"
	SortOrder      string // "asc" | "desc"
}
