package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/hp-page/domain"
)

type hpPageResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        string          `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	AttributeCode   *string         `json:"attribute_code"`
	AttributeValues []string        `json:"attribute_values"`
	CategoryIDs     []string        `json:"category_ids"`
	BrandIDs        []string        `json:"brand_ids"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toHpPageResponse(p *domain.HpPage) hpPageResponse {
	return hpPageResponse{
		ID:              p.ID(),
		Name:            p.Name(),
		Slug:            p.Slug(),
		ImageURL:        p.ImageURL(),
		MetaTitle:       p.MetaTitle(),
		MetaDescription: p.MetaDescription(),
		OrderIndex:      p.OrderIndex(),
		Content:         p.Content(),
		AttributeCode:   p.AttributeCode(),
		AttributeValues: p.AttributeValues(),
		CategoryIDs:     p.CategoryIDs(),
		BrandIDs:        p.BrandIDs(),
		CreatedAt:       p.CreatedAt(),
		UpdatedAt:       p.UpdatedAt(),
		DeletedAt:       p.DeletedAt(),
	}
}

func toHpPageResponseList(pages []*domain.HpPage) []hpPageResponse {
	result := make([]hpPageResponse, len(pages))
	for i, p := range pages {
		result[i] = toHpPageResponse(p)
	}
	return result
}

type createHpPageRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        string          `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	AttributeCode   *string         `json:"attribute_code"`
	AttributeValues []string        `json:"attribute_values"`
	CategoryIDs     []string        `json:"category_ids"`
	BrandIDs        []string        `json:"brand_ids"`
}

type updateHpPageRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	AttributeCode   *string         `json:"attribute_code"`
	AttributeValues []string        `json:"attribute_values"`
	CategoryIDs     []string        `json:"category_ids"`
	BrandIDs        []string        `json:"brand_ids"`
}
