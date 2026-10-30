package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/news/domain"
)

type imageAssetDTO struct {
	URL     string `json:"url"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

func toImageAssetDTOList(images []domain.ImageAsset) []imageAssetDTO {
	result := make([]imageAssetDTO, len(images))
	for i, img := range images {
		result[i] = imageAssetDTO{URL: img.URL, Alt: img.Alt, Caption: img.Caption}
	}
	return result
}

func toImageAssetDomainList(dtos []imageAssetDTO) []domain.ImageAsset {
	if dtos == nil {
		return nil
	}
	result := make([]domain.ImageAsset, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ImageAsset{URL: d.URL, Alt: d.Alt, Caption: d.Caption}
	}
	return result
}

type newsResponse struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Images          []imageAssetDTO `json:"images"`
	Content         json.RawMessage `json:"content"`
	Excerpt         string          `json:"excerpt"`
	CategoryID      *string         `json:"category_id"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toNewsResponse(n *domain.News) newsResponse {
	return newsResponse{
		ID:              n.ID(),
		Title:           n.Title(),
		TitleAlign:      n.TitleAlign(),
		Slug:            n.Slug(),
		Images:          toImageAssetDTOList(n.Images()),
		Content:         n.Content(),
		Excerpt:         n.Excerpt(),
		CategoryID:      n.CategoryID(),
		IsPublished:     n.IsPublished(),
		MetaTitle:       n.MetaTitle(),
		MetaDescription: n.MetaDescription(),
		OrderIndex:      n.OrderIndex(),
		CreatedAt:       n.CreatedAt(),
		UpdatedAt:       n.UpdatedAt(),
		DeletedAt:       n.DeletedAt(),
	}
}

func toNewsResponseList(items []*domain.News) []newsResponse {
	result := make([]newsResponse, len(items))
	for i, n := range items {
		result[i] = toNewsResponse(n)
	}
	return result
}

type createNewsRequest struct {
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Images          []imageAssetDTO `json:"images"`
	Content         json.RawMessage `json:"content"`
	Excerpt         string          `json:"excerpt"`
	CategoryID      *string         `json:"category_id"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
}

type updateNewsRequest struct {
	Title           *string         `json:"title"`
	TitleAlign      *string         `json:"title_align"`
	Slug            *string         `json:"slug"`
	Images          []imageAssetDTO `json:"images"`
	Content         json.RawMessage `json:"content"`
	Excerpt         *string         `json:"excerpt"`
	CategoryID      *string         `json:"category_id"`
	IsPublished     *bool           `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      *int            `json:"order_index"`
}

type countResponse struct {
	Count int `json:"count"`
}
