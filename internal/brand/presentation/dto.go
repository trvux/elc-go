package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/brand/domain"
)

type faqItemDTO struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type brandResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	LogoURL         string          `json:"logo_url"`
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

func toBrandResponse(b *domain.Brand) brandResponse {
	return brandResponse{
		ID:              b.ID(),
		Name:            b.Name(),
		Slug:            b.Slug(),
		LogoURL:         b.LogoURL(),
		MetaTitle:       b.MetaTitle(),
		MetaDescription: b.MetaDescription(),
		IsFeatured:      b.IsFeatured(),
		OrderIndex:      b.OrderIndex(),
		Content:         b.Content(),
		FAQ:             toFAQDTOList(b.FAQ()),
		CreatedAt:       b.CreatedAt(),
		UpdatedAt:       b.UpdatedAt(),
		DeletedAt:       b.DeletedAt(),
	}
}

func toBrandResponseList(brands []*domain.Brand) []brandResponse {
	result := make([]brandResponse, len(brands))
	for i, b := range brands {
		result[i] = toBrandResponse(b)
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

type createBrandRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	LogoURL         string          `json:"logo_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}

type updateBrandRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	LogoURL         *string         `json:"logo_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      *bool           `json:"is_featured"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	FAQ             []faqItemDTO    `json:"faq"`
}
