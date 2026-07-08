package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/branch/domain"
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
	result := make([]domain.ImageAsset, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ImageAsset{URL: d.URL, Alt: d.Alt, Caption: d.Caption}
	}
	return result
}

type branchResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Address         string          `json:"address"`
	Phone           string          `json:"phone"`
	Email           string          `json:"email"`
	MapsURL         string          `json:"maps_url"`
	MapsEmbed       string          `json:"maps_embed"`
	Description     json.RawMessage `json:"description"`
	Images          []imageAssetDTO `json:"images"`
	IsPublished     bool            `json:"is_published"`
	OrderIndex      int             `json:"order_index"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toBranchResponse(b *domain.Branch) branchResponse {
	return branchResponse{
		ID:              b.ID(),
		Name:            b.Name(),
		Slug:            b.Slug(),
		Address:         b.Address(),
		Phone:           b.Phone(),
		Email:           b.Email(),
		MapsURL:         b.MapsURL(),
		MapsEmbed:       b.MapsEmbed(),
		Description:     b.Description(),
		Images:          toImageAssetDTOList(b.Images()),
		IsPublished:     b.IsPublished(),
		OrderIndex:      b.OrderIndex(),
		MetaTitle:       b.MetaTitle(),
		MetaDescription: b.MetaDescription(),
		CreatedAt:       b.CreatedAt(),
		UpdatedAt:       b.UpdatedAt(),
		DeletedAt:       b.DeletedAt(),
	}
}

func toBranchResponseList(branches []*domain.Branch) []branchResponse {
	result := make([]branchResponse, len(branches))
	for i, b := range branches {
		result[i] = toBranchResponse(b)
	}
	return result
}

type createBranchRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Address         string          `json:"address"`
	Phone           string          `json:"phone"`
	Email           string          `json:"email"`
	MapsURL         string          `json:"maps_url"`
	MapsEmbed       string          `json:"maps_embed"`
	Description     json.RawMessage `json:"description"`
	Images          []imageAssetDTO `json:"images"`
	IsPublished     bool            `json:"is_published"`
	OrderIndex      int             `json:"order_index"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
}

type updateBranchRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	Address         *string         `json:"address"`
	Phone           *string         `json:"phone"`
	Email           *string         `json:"email"`
	MapsURL         *string         `json:"maps_url"`
	MapsEmbed       *string         `json:"maps_embed"`
	Description     json.RawMessage `json:"description"`
	Images          []imageAssetDTO `json:"images"`
	IsPublished     *bool           `json:"is_published"`
	OrderIndex      *int            `json:"order_index"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
}
