package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/system-page/domain"
)

type SystemPageDTO struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	MetaTitle       *string   `json:"meta_title"`
	MetaDescription *string   `json:"meta_description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func toSystemPageDTO(p *domain.SystemPage) SystemPageDTO {
	return SystemPageDTO{
		ID:              p.ID(),
		Name:            p.Name(),
		Slug:            p.Slug(),
		MetaTitle:       p.MetaTitle(),
		MetaDescription: p.MetaDescription(),
		CreatedAt:       p.CreatedAt(),
		UpdatedAt:       p.UpdatedAt(),
	}
}

func toSystemPageDTOList(pages []*domain.SystemPage) []SystemPageDTO {
	result := make([]SystemPageDTO, len(pages))
	for i, p := range pages {
		result[i] = toSystemPageDTO(p)
	}
	return result
}

type updateSystemPageRequest struct {
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
}
