package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/service/domain"
)

type seoDTO struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Noindex     bool    `json:"noindex,omitempty"`
}

func toSeoDTO(seo domain.Seo) seoDTO {
	return seoDTO{Title: seo.Title, Description: seo.Description, Noindex: seo.Noindex}
}

func toSeoDomain(seo seoDTO) domain.Seo {
	return domain.Seo{Title: seo.Title, Description: seo.Description, Noindex: seo.Noindex}
}

type refResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type serviceResponse struct {
	ID               string          `json:"id"`
	Title            string          `json:"title"`
	Slug             string          `json:"slug"`
	GroupID          *string         `json:"group_id"`
	CategoryID       *string         `json:"category_id"`
	OriginalPrice    *int64          `json:"original_price"`
	SalePrice        *int64          `json:"sale_price"`
	DiscountPercent  *int            `json:"discount_percent"`
	PriceDisplayText *string         `json:"price_display_text"`
	Labels           []string        `json:"labels"`
	Description      *string         `json:"description"`
	Content          json.RawMessage `json:"content"`
	Image            *string         `json:"image"`
	MetaTitle        *string         `json:"meta_title"`
	MetaDescription  *string         `json:"meta_description"`
	Seo              seoDTO          `json:"seo"`
	IsFeatured       bool            `json:"is_featured"`
	IsPublished      bool            `json:"is_published"`
	OrderIndex       int             `json:"order_index"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at"`
	Group            *refResponse    `json:"group"`
	Category         *refResponse    `json:"category"`
}

func toServiceResponse(sr *domain.ServiceWithRelations) serviceResponse {
	resp := serviceResponse{
		ID:               sr.ID(),
		Title:            sr.Title(),
		Slug:             sr.Slug(),
		GroupID:          sr.GroupID(),
		CategoryID:       sr.CategoryID(),
		OriginalPrice:    sr.OriginalPrice(),
		SalePrice:        sr.SalePrice(),
		DiscountPercent:  sr.DiscountPercent(),
		PriceDisplayText: sr.PriceDisplayText(),
		Labels:           sr.Labels(),
		Description:      sr.Description(),
		Content:          sr.Content(),
		Image:            sr.Image(),
		MetaTitle:        sr.MetaTitle(),
		MetaDescription:  sr.MetaDescription(),
		Seo:              toSeoDTO(sr.Seo()),
		IsFeatured:       sr.IsFeatured(),
		IsPublished:      sr.IsPublished(),
		OrderIndex:       sr.OrderIndex(),
		CreatedAt:        sr.CreatedAt(),
		UpdatedAt:        sr.UpdatedAt(),
		DeletedAt:        sr.DeletedAt(),
	}
	if sr.Group != nil {
		resp.Group = &refResponse{ID: sr.Group.ID, Name: sr.Group.Name}
	}
	if sr.Category != nil {
		resp.Category = &refResponse{ID: sr.Category.ID, Name: sr.Category.Name}
	}
	return resp
}

// toPlainServiceResponse is used for Create/Update, which only ever return a
// plain *domain.Service (no joined relations — see docs/service.md).
func toPlainServiceResponse(s *domain.Service) serviceResponse {
	return serviceResponse{
		ID:               s.ID(),
		Title:            s.Title(),
		Slug:             s.Slug(),
		GroupID:          s.GroupID(),
		CategoryID:       s.CategoryID(),
		OriginalPrice:    s.OriginalPrice(),
		SalePrice:        s.SalePrice(),
		DiscountPercent:  s.DiscountPercent(),
		PriceDisplayText: s.PriceDisplayText(),
		Labels:           s.Labels(),
		Description:      s.Description(),
		Content:          s.Content(),
		Image:            s.Image(),
		MetaTitle:        s.MetaTitle(),
		MetaDescription:  s.MetaDescription(),
		Seo:              toSeoDTO(s.Seo()),
		IsFeatured:       s.IsFeatured(),
		IsPublished:      s.IsPublished(),
		OrderIndex:       s.OrderIndex(),
		CreatedAt:        s.CreatedAt(),
		UpdatedAt:        s.UpdatedAt(),
		DeletedAt:        s.DeletedAt(),
	}
}

func toServiceResponseList(services []*domain.ServiceWithRelations) []serviceResponse {
	result := make([]serviceResponse, len(services))
	for i, s := range services {
		result[i] = toServiceResponse(s)
	}
	return result
}

type createServiceRequest struct {
	Title            string          `json:"title"`
	Slug             string          `json:"slug"`
	GroupID          *string         `json:"group_id"`
	CategoryID       *string         `json:"category_id"`
	OriginalPrice    *int64          `json:"original_price"`
	DiscountPercent  *int            `json:"discount_percent"`
	PriceDisplayText *string         `json:"price_display_text"`
	Labels           []string        `json:"labels"`
	Description      *string         `json:"description"`
	Content          json.RawMessage `json:"content"`
	Image            *string         `json:"image"`
	MetaTitle        *string         `json:"meta_title"`
	MetaDescription  *string         `json:"meta_description"`
	Seo              seoDTO          `json:"seo"`
	IsFeatured       bool            `json:"is_featured"`
	IsPublished      bool            `json:"is_published"`
	OrderIndex       int             `json:"order_index"`
}

type updateServiceRequest struct {
	Title            *string         `json:"title"`
	Slug             *string         `json:"slug"`
	GroupID          *string         `json:"group_id"`
	CategoryID       *string         `json:"category_id"`
	OriginalPrice    *int64          `json:"original_price"`
	DiscountPercent  *int            `json:"discount_percent"`
	PriceDisplayText *string         `json:"price_display_text"`
	Labels           []string        `json:"labels"`
	Description      *string         `json:"description"`
	Content          json.RawMessage `json:"content"`
	Image            *string         `json:"image"`
	MetaTitle        *string         `json:"meta_title"`
	MetaDescription  *string         `json:"meta_description"`
	Seo              *seoDTO         `json:"seo"`
	IsFeatured       *bool           `json:"is_featured"`
	IsPublished      *bool           `json:"is_published"`
	OrderIndex       *int            `json:"order_index"`
}
