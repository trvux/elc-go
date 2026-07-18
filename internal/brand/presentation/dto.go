package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/brand/domain"
)

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
	WarrantyPolicy  *string         `json:"warranty_policy"`
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
		WarrantyPolicy:  b.WarrantyPolicy(),
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

type createBrandRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	LogoURL         string          `json:"logo_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	WarrantyPolicy  *string         `json:"warranty_policy,omitempty"`
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
	WarrantyPolicy  *string         `json:"warranty_policy"`
}
