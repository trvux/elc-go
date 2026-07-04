package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/group/domain"
)

type faqItemDTO struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type groupResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toGroupResponse(g *domain.Group) groupResponse {
	return groupResponse{
		ID:              g.ID(),
		Name:            g.Name(),
		Slug:            g.Slug(),
		ImageURL:        g.ImageURL(),
		MetaTitle:       g.MetaTitle(),
		MetaDescription: g.MetaDescription(),
		IsFeatured:      g.IsFeatured(),
		OrderIndex:      g.OrderIndex(),
		Content:         g.Content(),
		FAQ:             toFAQDTOList(g.FAQ()),
		CreatedAt:       g.CreatedAt(),
		UpdatedAt:       g.UpdatedAt(),
		DeletedAt:       g.DeletedAt(),
	}
}

func toGroupResponseList(groups []*domain.Group) []groupResponse {
	result := make([]groupResponse, len(groups))
	for i, g := range groups {
		result[i] = toGroupResponse(g)
	}
	return result
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

type createGroupRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}

type updateGroupRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      *bool           `json:"is_featured"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}
