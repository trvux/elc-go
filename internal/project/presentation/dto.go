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
	if dtos == nil {
		return nil
	}
	result := make([]domain.ImageAsset, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ImageAsset{URL: d.URL, Alt: d.Alt, Caption: d.Caption}
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

type projectCategoryResponse struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name"`
	Slug      string                    `json:"slug"`
	GroupID   *string                   `json:"group_id"`
	Condition string                    `json:"condition"`
	Group     *categoryGroupRefResponse `json:"group"`
}

type projectServiceResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
	Line  string `json:"line"`
}

type projectResponse struct {
	ID                string                    `json:"id"`
	Title             string                    `json:"title"`
	TitleAlign        string                    `json:"title_align"`
	Slug              string                    `json:"slug"`
	Description       json.RawMessage           `json:"description"`
	Images            []imageAssetDTO           `json:"images"`
	IsFeatured        bool                      `json:"is_featured"`
	IsPublished       bool                      `json:"is_published"`
	MetaTitle         *string                   `json:"meta_title"`
	MetaDescription   *string                   `json:"meta_description"`
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
			Group: group,
		}
	}

	services := make([]projectServiceResponse, len(p.Services))
	for i, s := range p.Services {
		services[i] = projectServiceResponse{ID: s.ID, Title: s.Title, Slug: s.Slug, Line: s.Line}
	}

	return projectResponse{
		ID:                p.ID(),
		Title:             p.Title(),
		TitleAlign:        p.TitleAlign(),
		Slug:              p.Slug(),
		Description:       p.Description(),
		Images:            toImageAssetDTOList(p.Images()),
		IsFeatured:        p.IsFeatured(),
		IsPublished:       p.IsPublished(),
		MetaTitle:         p.MetaTitle(),
		MetaDescription:   p.MetaDescription(),
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
	TitleAlign        string                 `json:"title_align"`
	Slug              string                 `json:"slug"`
	Description       json.RawMessage        `json:"description"`
	Images            []imageAssetDTO        `json:"images"`
	IsFeatured        bool                   `json:"is_featured"`
	IsPublished       bool                   `json:"is_published"`
	MetaTitle         *string                `json:"meta_title"`
	MetaDescription   *string                `json:"meta_description"`
	OrderIndex        int                    `json:"order_index"`
	ProjectTypeID     *string                `json:"project_type_id"`
	ServiceIDs        []string               `json:"service_ids"`
	Categories        []categoryConditionDTO `json:"categories"`
	ClientName        string                 `json:"client_name"`
	Location          string                 `json:"location"`
	CompletedAt       *time.Time             `json:"completed_at"`
	TestimonialQuote  string                 `json:"testimonial_quote"`
	TestimonialAuthor string                 `json:"testimonial_author"`
}

type updateProjectRequest struct {
	Title             *string                 `json:"title"`
	TitleAlign        *string                 `json:"title_align"`
	Slug              *string                 `json:"slug"`
	Description       json.RawMessage         `json:"description"`
	Images            []imageAssetDTO         `json:"images"`
	IsFeatured        *bool                   `json:"is_featured"`
	IsPublished       *bool                   `json:"is_published"`
	MetaTitle         *string                 `json:"meta_title"`
	MetaDescription   *string                 `json:"meta_description"`
	OrderIndex        *int                    `json:"order_index"`
	ProjectTypeID     *string                 `json:"project_type_id"`
	ServiceIDs        *[]string               `json:"service_ids"`
	Categories        *[]categoryConditionDTO `json:"categories"`
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
