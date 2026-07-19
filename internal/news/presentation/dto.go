package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/news/domain"
)

type tagRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func toTagRefResponseList(tags []domain.TagRef) []tagRefResponse {
	result := make([]tagRefResponse, len(tags))
	for i, t := range tags {
		result[i] = tagRefResponse{ID: t.ID, Name: t.Name, Slug: t.Slug}
	}
	return result
}

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
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	Slug            string           `json:"slug"`
	Images          []imageAssetDTO  `json:"images"`
	Content         json.RawMessage  `json:"content"`
	Excerpt         string           `json:"excerpt"`
	CategoryID      *string          `json:"category_id"`
	AuthorID        *string          `json:"author_id"`
	IsPublished     bool             `json:"is_published"`
	MetaTitle       *string          `json:"meta_title"`
	MetaDescription *string          `json:"meta_description"`
	OrderIndex      int              `json:"order_index"`
	Tags            []tagRefResponse `json:"tags"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       *time.Time       `json:"deleted_at"`
}

func toNewsResponse(n *domain.News) newsResponse {
	return newsResponse{
		ID:              n.ID(),
		Title:           n.Title(),
		Slug:            n.Slug(),
		Images:          toImageAssetDTOList(n.Images()),
		Content:         n.Content(),
		Excerpt:         n.Excerpt(),
		CategoryID:      n.CategoryID(),
		AuthorID:        n.AuthorID(),
		IsPublished:     n.IsPublished(),
		MetaTitle:       n.MetaTitle(),
		MetaDescription: n.MetaDescription(),
		OrderIndex:      n.OrderIndex(),
		Tags:            toTagRefResponseList(n.Tags()),
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
	Slug            string          `json:"slug"`
	Images          []imageAssetDTO `json:"images"`
	Content         json.RawMessage `json:"content"`
	Excerpt         string          `json:"excerpt"`
	CategoryID      *string         `json:"category_id"`
	AuthorID        *string         `json:"author_id"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
	TagIDs          []string        `json:"tag_ids"`
}

type updateNewsRequest struct {
	Title           *string         `json:"title"`
	Slug            *string         `json:"slug"`
	Images          []imageAssetDTO `json:"images"`
	Content         json.RawMessage `json:"content"`
	Excerpt         *string         `json:"excerpt"`
	CategoryID      *string         `json:"category_id"`
	AuthorID        *string         `json:"author_id"`
	IsPublished     *bool           `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      *int            `json:"order_index"`
	TagIDs          *[]string       `json:"tag_ids"`
}

type countResponse struct {
	Count int `json:"count"`
}
