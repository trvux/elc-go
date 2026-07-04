package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

type serviceGroupResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	ImageURL        *string    `json:"image_url"`
	MetaTitle       *string    `json:"meta_title"`
	MetaDescription *string    `json:"meta_description"`
	IsFeatured      bool       `json:"is_featured"`
	OrderIndex      int        `json:"order_index"`
	CategoryIDs     []string   `json:"category_ids"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

func toServiceGroupResponse(sg *domain.ServiceGroup) serviceGroupResponse {
	return serviceGroupResponse{
		ID:              sg.ID(),
		Name:            sg.Name(),
		Slug:            sg.Slug(),
		ImageURL:        sg.ImageURL(),
		MetaTitle:       sg.MetaTitle(),
		MetaDescription: sg.MetaDescription(),
		IsFeatured:      sg.IsFeatured(),
		OrderIndex:      sg.OrderIndex(),
		CategoryIDs:     sg.CategoryIDs(),
		CreatedAt:       sg.CreatedAt(),
		UpdatedAt:       sg.UpdatedAt(),
		DeletedAt:       sg.DeletedAt(),
	}
}

func toServiceGroupResponseList(groups []*domain.ServiceGroup) []serviceGroupResponse {
	result := make([]serviceGroupResponse, len(groups))
	for i, sg := range groups {
		result[i] = toServiceGroupResponse(sg)
	}
	return result
}

type createServiceGroupRequest struct {
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	ImageURL        *string  `json:"image_url"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      bool     `json:"is_featured"`
	OrderIndex      int      `json:"order_index"`
	CategoryIDs     []string `json:"category_ids"`
}

type updateServiceGroupRequest struct {
	Name            *string  `json:"name"`
	Slug            *string  `json:"slug"`
	ImageURL        *string  `json:"image_url"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      *bool    `json:"is_featured"`
	OrderIndex      *int     `json:"order_index"`
	CategoryIDs     []string `json:"category_ids"`
}
