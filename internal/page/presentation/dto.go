package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/page/domain"
)

type PageDTO struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toPageDTO(p *domain.Page) PageDTO {
	return PageDTO{
		ID:              p.ID(),
		Title:           p.Title(),
		Slug:            p.Slug(),
		Content:         p.Content(),
		IsPublished:     p.IsPublished(),
		MetaTitle:       p.MetaTitle(),
		MetaDescription: p.MetaDescription(),
		OrderIndex:      p.OrderIndex(),
		CreatedAt:       p.CreatedAt(),
		UpdatedAt:       p.UpdatedAt(),
		DeletedAt:       p.DeletedAt(),
	}
}

func toPageDTOList(pages []*domain.Page) []PageDTO {
	result := make([]PageDTO, len(pages))
	for i, p := range pages {
		result[i] = toPageDTO(p)
	}
	return result
}
