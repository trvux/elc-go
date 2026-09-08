package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

type categoryGroupRefResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	ImageURL        *string `json:"image_url"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	IsFeatured      bool    `json:"is_featured"`
	OrderIndex      int     `json:"order_index"`
}

type categoryRefResponse struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	GroupID         *string                   `json:"group_id"`
	Slug            string                    `json:"slug"`
	ImageURL        *string                   `json:"image_url"`
	MetaTitle       *string                   `json:"meta_title"`
	MetaDescription *string                   `json:"meta_description"`
	IsFeatured      bool                      `json:"is_featured"`
	OrderIndex      int                       `json:"order_index"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	DeletedAt       *time.Time                `json:"deleted_at"`
	Group           *categoryGroupRefResponse `json:"group"`
}

type projectTypeResponse struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Slug            string                `json:"slug"`
	Image           *string               `json:"image"`
	MetaTitle       *string               `json:"meta_title"`
	MetaDescription *string               `json:"meta_description"`
	IsFeatured      bool                  `json:"is_featured"`
	OrderIndex      int                   `json:"order_index"`
	Content         json.RawMessage       `json:"content"`
	Categories      []categoryRefResponse `json:"categories"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	DeletedAt       *time.Time            `json:"deleted_at"`
}

func toCategoryRefResponse(c domain.CategoryRef) categoryRefResponse {
	var group *categoryGroupRefResponse
	if c.Group != nil {
		group = &categoryGroupRefResponse{
			ID:              c.Group.ID,
			Name:            c.Group.Name,
			Slug:            c.Group.Slug,
			ImageURL:        c.Group.ImageURL,
			MetaTitle:       c.Group.MetaTitle,
			MetaDescription: c.Group.MetaDescription,
			IsFeatured:      c.Group.IsFeatured,
			OrderIndex:      c.Group.OrderIndex,
		}
	}

	return categoryRefResponse{
		ID: c.ID, Name: c.Name, GroupID: c.GroupID, Slug: c.Slug,
		ImageURL: c.ImageURL, MetaTitle: c.MetaTitle, MetaDescription: c.MetaDescription,
		IsFeatured: c.IsFeatured, OrderIndex: c.OrderIndex,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, DeletedAt: c.DeletedAt,
		Group: group,
	}
}

func toProjectTypeResponse(pt *domain.ProjectTypeWithCategories) projectTypeResponse {
	categories := make([]categoryRefResponse, len(pt.Categories))
	for i, c := range pt.Categories {
		categories[i] = toCategoryRefResponse(c)
	}

	return projectTypeResponse{
		ID:              pt.ID(),
		Name:            pt.Name(),
		Slug:            pt.Slug(),
		Image:           pt.Image(),
		MetaTitle:       pt.MetaTitle(),
		MetaDescription: pt.MetaDescription(),
		IsFeatured:      pt.IsFeatured(),
		OrderIndex:      pt.OrderIndex(),
		Content:         pt.Content(),
		Categories:      categories,
		CreatedAt:       pt.CreatedAt(),
		UpdatedAt:       pt.UpdatedAt(),
		DeletedAt:       pt.DeletedAt(),
	}
}

func toProjectTypeResponseList(projectTypes []*domain.ProjectTypeWithCategories) []projectTypeResponse {
	result := make([]projectTypeResponse, len(projectTypes))
	for i, pt := range projectTypes {
		result[i] = toProjectTypeResponse(pt)
	}
	return result
}

// toBareProjectTypeResponse renders a plain *domain.ProjectType (no
// categories join) — used by Create/Update, which don't re-fetch the joined
// shape.
func toBareProjectTypeResponse(pt *domain.ProjectType) projectTypeResponse {
	return toProjectTypeResponse(&domain.ProjectTypeWithCategories{ProjectType: pt})
}

type createProjectTypeRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Image           *string         `json:"image"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      bool            `json:"is_featured"`
	OrderIndex      int             `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	CategoryIDs     []string        `json:"category_ids"`
}

type updateProjectTypeRequest struct {
	Name            *string         `json:"name"`
	Slug            *string         `json:"slug"`
	Image           *string         `json:"image"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsFeatured      *bool           `json:"is_featured"`
	OrderIndex      *int            `json:"order_index"`
	Content         json.RawMessage `json:"content"`
	CategoryIDs     *[]string       `json:"category_ids"`
}
