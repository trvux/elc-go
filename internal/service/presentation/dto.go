package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/service/domain"
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

// briefFieldDTO is the wire shape of domain.BriefField.
type briefFieldDTO struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Kind    string   `json:"kind"`
	Unit    string   `json:"unit,omitempty"`
	Options []string `json:"options,omitempty"`
}

func toBriefFieldDTOList(fields []domain.BriefField) []briefFieldDTO {
	result := make([]briefFieldDTO, len(fields))
	for i, f := range fields {
		result[i] = briefFieldDTO{Key: f.Key, Label: f.Label, Kind: f.Kind, Unit: f.Unit, Options: f.Options}
	}
	return result
}

// toBriefFieldDomainList keeps nil as nil: for an update, "not sent" must stay
// distinct from "sent empty" (which clears the list).
func toBriefFieldDomainList(dtos []briefFieldDTO) []domain.BriefField {
	if dtos == nil {
		return nil
	}
	result := make([]domain.BriefField, len(dtos))
	for i, d := range dtos {
		result[i] = domain.BriefField{Key: d.Key, Label: d.Label, Kind: d.Kind, Unit: d.Unit, Options: d.Options}
	}
	return result
}

type serviceResponse struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Labels          []string        `json:"labels"`
	Line            string          `json:"line"`
	BriefPrompts    []string        `json:"brief_prompts"`
	BriefFields     []briefFieldDTO `json:"brief_fields"`
	CategoryIDs     []string        `json:"category_ids"`
	Description     *string         `json:"description"`
	Content         json.RawMessage `json:"content"`
	Images          []imageAssetDTO `json:"images"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsPublished     bool            `json:"is_published"`
	OrderIndex      int             `json:"order_index"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
}

func toServiceResponse(sr *domain.ServiceWithRelations) serviceResponse {
	resp := serviceResponse{
		ID:              sr.ID(),
		Title:           sr.Title(),
		TitleAlign:      sr.TitleAlign(),
		Slug:            sr.Slug(),
		Labels:          sr.Labels(),
		Line:            sr.Line(),
		BriefPrompts:    orEmpty(sr.BriefPrompts()),
		BriefFields:     toBriefFieldDTOList(sr.BriefFields()),
		CategoryIDs:     orEmpty(sr.CategoryIDs),
		Description:     sr.Description(),
		Content:         sr.Content(),
		Images:          toImageAssetDTOList(sr.Images()),
		MetaTitle:       sr.MetaTitle(),
		MetaDescription: sr.MetaDescription(),
		IsPublished:     sr.IsPublished(),
		OrderIndex:      sr.OrderIndex(),
		CreatedAt:       sr.CreatedAt(),
		UpdatedAt:       sr.UpdatedAt(),
		DeletedAt:       sr.DeletedAt(),
	}
	return resp
}

// toPlainServiceResponse is used for Create/Update, which only ever return a
// plain *domain.Service (no joined relations — see docs/service.md).
func toPlainServiceResponse(s *domain.Service) serviceResponse {
	return serviceResponse{
		ID:              s.ID(),
		Title:           s.Title(),
		TitleAlign:      s.TitleAlign(),
		Slug:            s.Slug(),
		Labels:          s.Labels(),
		Line:            s.Line(),
		BriefPrompts:    orEmpty(s.BriefPrompts()),
		BriefFields:     toBriefFieldDTOList(s.BriefFields()),
		CategoryIDs:     []string{},
		Description:     s.Description(),
		Content:         s.Content(),
		Images:          toImageAssetDTOList(s.Images()),
		MetaTitle:       s.MetaTitle(),
		MetaDescription: s.MetaDescription(),
		IsPublished:     s.IsPublished(),
		OrderIndex:      s.OrderIndex(),
		CreatedAt:       s.CreatedAt(),
		UpdatedAt:       s.UpdatedAt(),
		DeletedAt:       s.DeletedAt(),
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
	Title           string          `json:"title"`
	TitleAlign      string          `json:"title_align"`
	Slug            string          `json:"slug"`
	Labels          []string        `json:"labels"`
	Line            string          `json:"line"`
	BriefPrompts    []string        `json:"brief_prompts"`
	BriefFields     []briefFieldDTO `json:"brief_fields"`
	CategoryIDs     []string        `json:"category_ids"`
	Description     *string         `json:"description"`
	Content         json.RawMessage `json:"content"`
	Images          []imageAssetDTO `json:"images"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsPublished     bool            `json:"is_published"`
	OrderIndex      int             `json:"order_index"`
}

type updateServiceRequest struct {
	Title           *string         `json:"title"`
	TitleAlign      *string         `json:"title_align"`
	Slug            *string         `json:"slug"`
	Labels          []string        `json:"labels"`
	Line            *string         `json:"line"`
	BriefPrompts    []string        `json:"brief_prompts"`
	BriefFields     []briefFieldDTO `json:"brief_fields"`
	CategoryIDs     []string        `json:"category_ids"`
	Description     *string         `json:"description"`
	Content         json.RawMessage `json:"content"`
	Images          []imageAssetDTO `json:"images"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	IsPublished     *bool           `json:"is_published"`
	OrderIndex      *int            `json:"order_index"`
}

// orEmpty keeps JSON output as [] instead of null for an unset list.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
