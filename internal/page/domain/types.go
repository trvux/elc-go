package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
	"github.com/trvux/elc-go/internal/platform/titlealign"
)

type Page struct {
	id              string
	title           string
	titleAlign      string
	slug            string
	content         json.RawMessage
	isPublished     bool
	metaTitle       *string
	metaDescription *string
	orderIndex      int
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// TitleAlignLeft/Center/Right re-export platform/titlealign — see its doc
// comment.
const (
	TitleAlignLeft   = titlealign.Left
	TitleAlignCenter = titlealign.Center
	TitleAlignRight  = titlealign.Right
)

func NewPage(
	title, slug string,
	titleAlign string,
	content json.RawMessage,
	isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
) (*Page, error) {
	fieldErrors := make(map[string][]string)
	if title == "" {
		fieldErrors["title"] = []string{"title cannot be empty"}
	}
	if slug == "" {
		fieldErrors["slug"] = []string{"slug cannot be empty"}
	}
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		fieldErrors["metaTitle"] = errs
	}
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		fieldErrors["metaDescription"] = errs
	}
	titleAlign = titlealign.OrDefault(titleAlign, titlealign.Center)
	if !titlealign.Valid(titleAlign) {
		fieldErrors["titleAlign"] = []string{"titleAlign must be 'left', 'center' or 'right'"}
	}
	if len(fieldErrors) > 0 {
		return nil, apperr.NewValidationError("invalid page input", fieldErrors)
	}

	if content == nil {
		content = json.RawMessage("{}")
	}

	return &Page{
		title:           title,
		titleAlign:      titleAlign,
		slug:            slug,
		content:         content,
		isPublished:     isPublished,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
	}, nil
}

func RehydratePage(
	id string,
	title string,
	titleAlign string,
	slug string,
	content json.RawMessage,
	isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Page {
	return &Page{
		id:              id,
		title:           title,
		titleAlign:      titleAlign,
		slug:            slug,
		content:         content,
		isPublished:     isPublished,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		orderIndex:      orderIndex,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (p *Page) ID() string               { return p.id }
func (p *Page) Title() string            { return p.title }
func (p *Page) TitleAlign() string       { return p.titleAlign }
func (p *Page) Slug() string             { return p.slug }
func (p *Page) Content() json.RawMessage { return p.content }
func (p *Page) IsPublished() bool        { return p.isPublished }
func (p *Page) MetaTitle() *string       { return p.metaTitle }
func (p *Page) MetaDescription() *string { return p.metaDescription }
func (p *Page) OrderIndex() int          { return p.orderIndex }
func (p *Page) CreatedAt() time.Time     { return p.createdAt }
func (p *Page) UpdatedAt() time.Time     { return p.updatedAt }
func (p *Page) DeletedAt() *time.Time    { return p.deletedAt }

func (p *Page) Update(
	title, slug string,
	titleAlign string,
	content json.RawMessage,
	isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
) error {
	fieldErrors := make(map[string][]string)
	if title == "" {
		fieldErrors["title"] = []string{"title cannot be empty"}
	}
	if slug == "" {
		fieldErrors["slug"] = []string{"slug cannot be empty"}
	}
	titleAlign = titlealign.OrDefault(titleAlign, titlealign.Center)
	if !titlealign.Valid(titleAlign) {
		fieldErrors["titleAlign"] = []string{"titleAlign must be 'left', 'center' or 'right'"}
	}
	// Only re-validate a meta field against the length limit if it's
	// actually changing — Update always resends the whole record (no
	// pointer-based partial-update convention here), so a pre-existing
	// title/description that already exceeds the limit would otherwise
	// block every edit to the page, not just ones that touch its SEO copy.
	if !seo.Unchanged(p.metaTitle, metaTitle) {
		if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
			fieldErrors["metaTitle"] = errs
		}
	}
	if !seo.Unchanged(p.metaDescription, metaDescription) {
		if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
			fieldErrors["metaDescription"] = errs
		}
	}
	if len(fieldErrors) > 0 {
		return apperr.NewValidationError("invalid page input", fieldErrors)
	}

	p.title = title
	p.titleAlign = titleAlign
	p.slug = slug
	if content != nil {
		p.content = content
	}
	p.isPublished = isPublished
	p.metaTitle = metaTitle
	p.metaDescription = metaDescription
	p.orderIndex = orderIndex
	p.updatedAt = time.Now()
	return nil
}

type CreatePageInput struct {
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
}

type UpdatePageInput struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
}
