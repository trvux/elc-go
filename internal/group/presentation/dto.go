package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/group/domain"
)

type groupResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	IsHidden        bool            `json:"is_hidden"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
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
		IsHidden:        g.IsHidden(),
		OrderIndex:      g.OrderIndex(),
		Content:         g.Content(),
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

type createGroupRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	IsHidden        bool            `json:"is_hidden"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
}

type updateGroupRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	ImageURL        *string         `json:"image_url"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      *bool           `json:"is_featured"`
	IsHidden        *bool           `json:"is_hidden"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
}
