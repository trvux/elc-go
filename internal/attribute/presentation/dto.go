package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

type attributeDefinitionResponse struct {
	ID         string     `json:"id"`
	CategoryID *string    `json:"category_id"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	GroupLabel *string    `json:"group_label"`
	DataType   string     `json:"data_type"`
	Unit       *string    `json:"unit"`
	Options    []string   `json:"options"`
	OrderIndex int        `json:"order_index"`
	IsRequired bool       `json:"is_required"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

func toAttributeDefinitionResponse(d *domain.AttributeDefinition) attributeDefinitionResponse {
	return attributeDefinitionResponse{
		ID:         d.ID(),
		CategoryID: d.CategoryID(),
		Code:       d.Code(),
		Name:       d.Name(),
		GroupLabel: d.GroupLabel(),
		DataType:   d.DataType(),
		Unit:       d.Unit(),
		Options:    d.Options(),
		OrderIndex: d.OrderIndex(),
		IsRequired: d.IsRequired(),
		CreatedAt:  d.CreatedAt(),
		UpdatedAt:  d.UpdatedAt(),
		DeletedAt:  d.DeletedAt(),
	}
}

func toAttributeDefinitionResponseList(defs []*domain.AttributeDefinition) []attributeDefinitionResponse {
	result := make([]attributeDefinitionResponse, len(defs))
	for i, d := range defs {
		result[i] = toAttributeDefinitionResponse(d)
	}
	return result
}

type createAttributeDefinitionRequest struct {
	CategoryID *string  `json:"category_id"`
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	GroupLabel *string  `json:"group_label"`
	DataType   string   `json:"data_type"`
	Unit       *string  `json:"unit"`
	Options    []string `json:"options"`
	OrderIndex int      `json:"order_index"`
	IsRequired bool     `json:"is_required"`
}

type updateAttributeDefinitionRequest struct {
	Name       *string   `json:"name"`
	GroupLabel *string   `json:"group_label"`
	Unit       *string   `json:"unit"`
	Options    *[]string `json:"options"`
	OrderIndex *int      `json:"order_index"`
	IsRequired *bool     `json:"is_required"`
}
