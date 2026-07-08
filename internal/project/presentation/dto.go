package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/project/domain"
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

type projectTypeRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type categoryGroupRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type serviceGroupRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type projectCategoryResponse struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Slug       string                    `json:"slug"`
	GroupID    *string                   `json:"group_id"`
	Condition  string                    `json:"condition"`
	Group      *categoryGroupRefResponse `json:"group"`
	LowPrice   int64                     `json:"low_price"`
	HighPrice  int64                     `json:"high_price"`
	OfferCount int                       `json:"offer_count"`
}

type projectServiceResponse struct {
	ID    string                   `json:"id"`
	Title string                   `json:"title"`
	Slug  string                   `json:"slug"`
	Group *serviceGroupRefResponse `json:"group"`
}

type projectResponse struct {
	ID                string                    `json:"id"`
	Title             string                    `json:"title"`
	Slug              string                    `json:"slug"`
	Description       json.RawMessage           `json:"description"`
	Images            []imageAssetDTO           `json:"images"`
	IsFeatured        bool                      `json:"is_featured"`
	IsPublished       bool                      `json:"is_published"`
	MetaTitle         *string                   `json:"meta_title"`
	MetaDescription   *string                   `json:"meta_description"`
	Seo               seoDTO                    `json:"seo"`
	OrderIndex        int                       `json:"order_index"`
	ProjectTypeID     *string                   `json:"project_type_id"`
	ClientName        string                    `json:"client_name"`
	Location          string                    `json:"location"`
	CompletedAt       *time.Time                `json:"completed_at"`
	TestimonialQuote  string                    `json:"testimonial_quote"`
	TestimonialAuthor string                    `json:"testimonial_author"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	DeletedAt         *time.Time                `json:"deleted_at"`
	ProjectType       *projectTypeRefResponse   `json:"project_type"`
	Categories        []projectCategoryResponse `json:"categories"`
	Services          []projectServiceResponse  `json:"services"`
	Tags              []tagRefResponse          `json:"tags"`
}

func toProjectResponse(p *domain.ProjectWithRelations) projectResponse {
	var projectType *projectTypeRefResponse
	if p.ProjectType != nil {
		projectType = &projectTypeRefResponse{ID: p.ProjectType.ID, Name: p.ProjectType.Name, Slug: p.ProjectType.Slug}
	}

	categories := make([]projectCategoryResponse, len(p.Categories))
	for i, c := range p.Categories {
		var group *categoryGroupRefResponse
		if c.Group != nil {
			group = &categoryGroupRefResponse{ID: c.Group.ID, Name: c.Group.Name}
		}
		categories[i] = projectCategoryResponse{
			ID: c.ID, Name: c.Name, Slug: c.Slug, GroupID: c.GroupID, Condition: c.Condition,
			Group: group, LowPrice: c.LowPrice, HighPrice: c.HighPrice, OfferCount: c.OfferCount,
		}
	}

	services := make([]projectServiceResponse, len(p.Services))
	for i, s := range p.Services {
		var group *serviceGroupRefResponse
		if s.Group != nil {
			group = &serviceGroupRefResponse{ID: s.Group.ID, Name: s.Group.Name, Slug: s.Group.Slug}
		}
		services[i] = projectServiceResponse{ID: s.ID, Title: s.Title, Slug: s.Slug, Group: group}
	}

	return projectResponse{
		ID:                p.ID(),
		Title:             p.Title(),
		Slug:              p.Slug(),
		Description:       p.Description(),
		Images:            toImageAssetDTOList(p.Images()),
		IsFeatured:        p.IsFeatured(),
		IsPublished:       p.IsPublished(),
		MetaTitle:         p.MetaTitle(),
		MetaDescription:   p.MetaDescription(),
		Seo:               toSeoDTO(p.Seo()),
		OrderIndex:        p.OrderIndex(),
		ProjectTypeID:     p.ProjectTypeID(),
		ClientName:        p.ClientName(),
		Location:          p.Location(),
		CompletedAt:       p.CompletedAt(),
		TestimonialQuote:  p.TestimonialQuote(),
		TestimonialAuthor: p.TestimonialAuthor(),
		CreatedAt:         p.CreatedAt(),
		UpdatedAt:         p.UpdatedAt(),
		DeletedAt:         p.DeletedAt(),
		ProjectType:       projectType,
		Categories:        categories,
		Services:          services,
		Tags:              toTagRefResponseList(p.Tags),
	}
}

func toProjectResponseList(projects []*domain.ProjectWithRelations) []projectResponse {
	result := make([]projectResponse, len(projects))
	for i, p := range projects {
		result[i] = toProjectResponse(p)
	}
	return result
}

// toPlainProjectResponse renders a *domain.Project with no relations — used
// by Create/Update, which only ever return the plain row (see
// domain.ProjectRepository's doc comment).
func toPlainProjectResponse(p *domain.Project) projectResponse {
	return toProjectResponse(&domain.ProjectWithRelations{Project: p})
}

type categoryConditionDTO struct {
	CategoryID string `json:"category_id"`
	Condition  string `json:"condition"`
}

func toCategoryConditionDomainList(dtos []categoryConditionDTO) []domain.CategoryCondition {
	if dtos == nil {
		return nil
	}
	result := make([]domain.CategoryCondition, len(dtos))
	for i, d := range dtos {
		result[i] = domain.CategoryCondition{CategoryID: d.CategoryID, Condition: d.Condition}
	}
	return result
}

func toCategoryConditionDomainPtr(dtos *[]categoryConditionDTO) *[]domain.CategoryCondition {
	if dtos == nil {
		return nil
	}
	converted := toCategoryConditionDomainList(*dtos)
	if converted == nil {
		converted = []domain.CategoryCondition{}
	}
	return &converted
}

type createProjectRequest struct {
	Title             string                 `json:"title"`
	Slug              string                 `json:"slug"`
	Description       json.RawMessage        `json:"description"`
	Images            []imageAssetDTO        `json:"images"`
	IsFeatured        bool                   `json:"is_featured"`
	IsPublished       bool                   `json:"is_published"`
	MetaTitle         *string                `json:"meta_title"`
	MetaDescription   *string                `json:"meta_description"`
	Seo               seoDTO                 `json:"seo"`
	OrderIndex        int                    `json:"order_index"`
	ProjectTypeID     *string                `json:"project_type_id"`
	ServiceIDs        []string               `json:"service_ids"`
	Categories        []categoryConditionDTO `json:"categories"`
	TagIDs            []string               `json:"tag_ids"`
	ClientName        string                 `json:"client_name"`
	Location          string                 `json:"location"`
	CompletedAt       *time.Time             `json:"completed_at"`
	TestimonialQuote  string                 `json:"testimonial_quote"`
	TestimonialAuthor string                 `json:"testimonial_author"`
}

type updateProjectRequest struct {
	Title             *string                 `json:"title"`
	Slug              *string                 `json:"slug"`
	Description       json.RawMessage         `json:"description"`
	Images            []imageAssetDTO         `json:"images"`
	IsFeatured        *bool                   `json:"is_featured"`
	IsPublished       *bool                   `json:"is_published"`
	MetaTitle         *string                 `json:"meta_title"`
	MetaDescription   *string                 `json:"meta_description"`
	Seo               *seoDTO                 `json:"seo"`
	OrderIndex        *int                    `json:"order_index"`
	ProjectTypeID     *string                 `json:"project_type_id"`
	ServiceIDs        *[]string               `json:"service_ids"`
	Categories        *[]categoryConditionDTO `json:"categories"`
	TagIDs            *[]string               `json:"tag_ids"`
	ClientName        *string                 `json:"client_name"`
	Location          *string                 `json:"location"`
	CompletedAt       *time.Time              `json:"completed_at"`
	TestimonialQuote  *string                 `json:"testimonial_quote"`
	TestimonialAuthor *string                 `json:"testimonial_author"`
}

type adjacentProjectResponse struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type adjacentProjectsResponse struct {
	Prev *adjacentProjectResponse `json:"prev"`
	Next *adjacentProjectResponse `json:"next"`
}

func toAdjacentProjectResponse(a *domain.AdjacentProject) *adjacentProjectResponse {
	if a == nil {
		return nil
	}
	return &adjacentProjectResponse{Title: a.Title, Slug: a.Slug}
}

type categoryRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func toCategoryRefResponseList(refs []domain.CategoryRef) []categoryRefResponse {
	result := make([]categoryRefResponse, len(refs))
	for i, c := range refs {
		result[i] = categoryRefResponse{ID: c.ID, Name: c.Name, Slug: c.Slug}
	}
	return result
}

type countResponse struct {
	Count int `json:"count"`
}
