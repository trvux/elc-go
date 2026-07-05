package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/category/domain"
)

type faqItemDTO struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type groupRefResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	ImageURL        *string `json:"image_url"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	IsFeatured      bool    `json:"is_featured"`
	OrderIndex      int     `json:"order_index"`
}

type categoryResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	GroupID         *string           `json:"group_id"`
	Group           *groupRefResponse `json:"group"`
	ImageURL        *string           `json:"image_url"`
	MetaTitle       *string           `json:"meta_title"`
	MetaDescription *string           `json:"meta_description"`
	IsFeatured      bool              `json:"is_featured"`
	OrderIndex      int               `json:"order_index"`
	Content         json.RawMessage   `json:"content"`
	FAQ             []faqItemDTO      `json:"faq"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       *time.Time        `json:"deleted_at"`
}

func toCategoryResponse(c *domain.CategoryWithRelations) categoryResponse {
	var group *groupRefResponse
	if c.Group != nil {
		group = &groupRefResponse{
			ID:              c.Group.ID,
			Name:            c.Group.Name,
			Slug:            c.Group.Slug,
			ImageURL:        c.Group.ImageURL,
			MetaTitle:       c.Group.MetaTitle,
			MetaDescription: c.Group.MetaDescription,
			IsFeatured:      c.Group.IsFeatured,
			OrderIndex:      c.Group.OrderIndex,
		}
	}

	return categoryResponse{
		ID:              c.ID(),
		Name:            c.Name(),
		Slug:            c.Slug(),
		GroupID:         c.GroupID(),
		Group:           group,
		ImageURL:        c.ImageURL(),
		MetaTitle:       c.MetaTitle(),
		MetaDescription: c.MetaDescription(),
		IsFeatured:      c.IsFeatured(),
		OrderIndex:      c.OrderIndex(),
		Content:         c.Content(),
		FAQ:             toFAQDTOList(c.FAQ()),
		CreatedAt:       c.CreatedAt(),
		UpdatedAt:       c.UpdatedAt(),
		DeletedAt:       c.DeletedAt(),
	}
}

func toCategoryResponseList(categories []*domain.CategoryWithRelations) []categoryResponse {
	result := make([]categoryResponse, len(categories))
	for i, c := range categories {
		result[i] = toCategoryResponse(c)
	}
	return result
}

// toBareCategoryResponse renders a plain *domain.Category (no group join) —
// used by Create/Update, which don't re-fetch the joined shape.
func toBareCategoryResponse(c *domain.Category) categoryResponse {
	return toCategoryResponse(&domain.CategoryWithRelations{Category: c})
}

func toFAQDTOList(faq []domain.FAQItem) []faqItemDTO {
	if faq == nil {
		return nil
	}
	result := make([]faqItemDTO, len(faq))
	for i, item := range faq {
		result[i] = faqItemDTO{Question: item.Question, Answer: item.Answer}
	}
	return result
}

func toFAQDomainList(faq []faqItemDTO) []domain.FAQItem {
	if faq == nil {
		return nil
	}
	result := make([]domain.FAQItem, len(faq))
	for i, item := range faq {
		result[i] = domain.FAQItem{Question: item.Question, Answer: item.Answer}
	}
	return result
}

type createCategoryRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	GroupID         *string         `json:"group_id"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}

type updateCategoryRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	GroupID         *string         `json:"group_id"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      *bool           `json:"is_featured"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}
