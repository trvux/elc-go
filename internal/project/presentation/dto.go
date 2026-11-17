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
	Role    string `json:"role,omitempty"`
}

func toImageAssetDTOList(images []domain.ImageAsset) []imageAssetDTO {
	result := make([]imageAssetDTO, len(images))
	for i, img := range images {
		result[i] = imageAssetDTO{URL: img.URL, Alt: img.Alt, Caption: img.Caption, Role: img.Role}
	}
	return result
}

func toImageAssetDomainList(dtos []imageAssetDTO) []domain.ImageAsset {
	if dtos == nil {
		return nil
	}
	result := make([]domain.ImageAsset, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ImageAsset{URL: d.URL, Alt: d.Alt, Caption: d.Caption, Role: d.Role}
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

// factsDTO is the wire shape of domain.ProjectFacts, used for both requests
// and responses (nested under "facts").
type factsDTO struct {
	Province           string     `json:"province"`
	Ward               string     `json:"ward"`
	CompletedPrecision string     `json:"completed_precision"`
	ClientKind         *string    `json:"client_kind"`
	ClientConsentAt    *time.Time `json:"client_consent_at"`
	ClientConsentNote  string     `json:"client_consent_note"`
	IndoorUnits        *int       `json:"indoor_units"`
	CapacityHP         *float64   `json:"capacity_hp"`
	AirflowM3H         *int       `json:"airflow_m3h"`
	AreaM2             *float64   `json:"area_m2"`
	Floors             *int       `json:"floors"`
	DurationDays       *int       `json:"duration_days"`
	WarrantyMonths     *int       `json:"warranty_months"`
	Brands             []string   `json:"brands"`
}

func toFactsDTO(f domain.ProjectFacts) factsDTO {
	brands := f.Brands
	if brands == nil {
		brands = []string{}
	}
	return factsDTO{
		Province: f.Province, Ward: f.Ward, CompletedPrecision: f.CompletedPrecision,
		ClientKind: f.ClientKind, ClientConsentAt: f.ClientConsentAt, ClientConsentNote: f.ClientConsentNote,
		IndoorUnits: f.IndoorUnits, CapacityHP: f.CapacityHP, AirflowM3H: f.AirflowM3H, AreaM2: f.AreaM2,
		Floors: f.Floors, DurationDays: f.DurationDays, WarrantyMonths: f.WarrantyMonths, Brands: brands,
	}
}

func (d factsDTO) toDomain() domain.ProjectFacts {
	return domain.ProjectFacts{
		Province: d.Province, Ward: d.Ward, CompletedPrecision: d.CompletedPrecision,
		ClientKind: d.ClientKind, ClientConsentAt: d.ClientConsentAt, ClientConsentNote: d.ClientConsentNote,
		IndoorUnits: d.IndoorUnits, CapacityHP: d.CapacityHP, AirflowM3H: d.AirflowM3H, AreaM2: d.AreaM2,
		Floors: d.Floors, DurationDays: d.DurationDays, WarrantyMonths: d.WarrantyMonths, Brands: d.Brands,
	}
}

type projectResponse struct {
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	TitleAlign        string          `json:"title_align"`
	Slug              string          `json:"slug"`
	Description       json.RawMessage `json:"description"`
	Images            []imageAssetDTO `json:"images"`
	IsFeatured        bool            `json:"is_featured"`
	IsPublished       bool            `json:"is_published"`
	MetaTitle         *string         `json:"meta_title"`
	MetaDescription   *string         `json:"meta_description"`
	OrderIndex        int             `json:"order_index"`
	ProjectTypeID     *string         `json:"project_type_id"`
	ClientName        string          `json:"client_name"`
	Location          string          `json:"location"`
	CompletedAt       *time.Time      `json:"completed_at"`
	TestimonialQuote  string          `json:"testimonial_quote"`
	TestimonialAuthor string          `json:"testimonial_author"`
	Facts             factsDTO        `json:"facts"`
	// PublishGaps is staff-only and only set where relations were loaded.
	PublishGaps []string                  `json:"publish_gaps,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	DeletedAt   *time.Time                `json:"deleted_at"`
	ProjectType *projectTypeRefResponse   `json:"project_type"`
	Categories  []projectCategoryResponse `json:"categories"`
	Services    []projectServiceResponse  `json:"services"`
}

// toProjectResponse renders the PUBLIC view: drafts are filtered out by the
// handler, and personal data is masked here — a natural person's client name
// is blanked until consent is on record, and the consent bookkeeping fields
// never leave the server. Staff get the full row via toAdminProjectResponse.
func toProjectResponse(p *domain.ProjectWithRelations) projectResponse {
	return renderProject(p, false, false)
}

// toAdminProjectResponse is the staff view: everything, plus publish_gaps
// (what is still missing before the project could be published).
func toAdminProjectResponse(p *domain.ProjectWithRelations) projectResponse {
	return renderProject(p, true, true)
}

func renderProject(p *domain.ProjectWithRelations, admin, withGaps bool) projectResponse {
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

	facts := toFactsDTO(p.Facts())
	clientName := p.ClientName()
	var gaps []string
	if admin {
		if withGaps {
			gaps = domain.PublishGaps(p.Project, len(p.Services), len(p.Categories))
		}
	} else {
		if !p.ClientNamePublic() {
			clientName = ""
		}
		facts.ClientConsentAt = nil
		facts.ClientConsentNote = ""
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
		ClientName:        clientName,
		Location:          p.Location(),
		CompletedAt:       p.CompletedAt(),
		TestimonialQuote:  p.TestimonialQuote(),
		TestimonialAuthor: p.TestimonialAuthor(),
		Facts:             facts,
		PublishGaps:       gaps,
		CreatedAt:         p.CreatedAt(),
		UpdatedAt:         p.UpdatedAt(),
		DeletedAt:         p.DeletedAt(),
		ProjectType:       projectType,
		Categories:        categories,
		Services:          services,
	}
}

func toProjectResponseList(projects []*domain.ProjectWithRelations, admin bool) []projectResponse {
	result := make([]projectResponse, len(projects))
	for i, p := range projects {
		if admin {
			result[i] = toAdminProjectResponse(p)
		} else {
			result[i] = toProjectResponse(p)
		}
	}
	return result
}

// toPlainProjectResponse renders a *domain.Project with no relations — used
// by Create/Update, which are staff-only and only ever return the plain row
// (see domain.ProjectRepository's doc comment). No publish_gaps: the
// relation counts aren't loaded here.
func toPlainProjectResponse(p *domain.Project) projectResponse {
	return renderProject(&domain.ProjectWithRelations{Project: p}, true, false)
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
	Facts             factsDTO               `json:"facts"`
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
	Facts             *factsDTO               `json:"facts"`
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
