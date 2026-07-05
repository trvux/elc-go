package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/news/domain"
)

type newsResponse struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Image           string          `json:"image"`
	Content         json.RawMessage `json:"content"`
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
		Slug:            n.Slug(),
		Image:           n.Image(),
		Content:         n.Content(),
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
	Slug            string          `json:"slug"`
	Image           string          `json:"image"`
	Content         json.RawMessage `json:"content"`
	CategoryID      *string         `json:"category_id"`
	IsPublished     bool            `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      int             `json:"order_index"`
}

type updateNewsRequest struct {
	Title           *string         `json:"title"`
	Slug            *string         `json:"slug"`
	Image           *string         `json:"image"`
	Content         json.RawMessage `json:"content"`
	CategoryID      *string         `json:"category_id"`
	IsPublished     *bool           `json:"is_published"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	OrderIndex      *int            `json:"order_index"`
}

type countResponse struct {
	Count int `json:"count"`
}
