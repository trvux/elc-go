package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type Page struct {
	id              string
	title           string
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

func NewPage(
	title, slug string,
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
	if len(fieldErrors) > 0 {
		return nil, apperr.NewValidationError("invalid page input", fieldErrors)
	}

	if content == nil {
		content = json.RawMessage("{}")
	}

	return &Page{
		title:           title,
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
	title, slug string,
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

func (p *Page) ID() string              { return p.id }
func (p *Page) Title() string           { return p.title }
func (p *Page) Slug() string            { return p.slug }
func (p *Page) Content() json.RawMessage { return p.content }
func (p *Page) IsPublished() bool       { return p.isPublished }
func (p *Page) MetaTitle() *string       { return p.metaTitle }
func (p *Page) MetaDescription() *string { return p.metaDescription }
func (p *Page) OrderIndex() int          { return p.orderIndex }
func (p *Page) CreatedAt() time.Time    { return p.createdAt }
func (p *Page) UpdatedAt() time.Time    { return p.updatedAt }
func (p *Page) DeletedAt() *time.Time   { return p.deletedAt }

func (p *Page) Update(
	title, slug string,
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
	if len(fieldErrors) > 0 {
		return apperr.NewValidationError("invalid page input", fieldErrors)
	}

	p.title = title
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
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
}
