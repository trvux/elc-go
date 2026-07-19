package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

type attributeDefinitionResponse struct {
	ID          string     `json:"id"`
	CategoryIDs []string   `json:"category_ids"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	GroupLabel  *string    `json:"group_label"`
	DataType    string     `json:"data_type"`
	Unit        *string    `json:"unit"`
	Options     []string   `json:"options"`
	OrderIndex  int        `json:"order_index"`
	IsRequired  bool       `json:"is_required"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

func toAttributeDefinitionResponse(d *domain.AttributeDefinitionWithCategories) attributeDefinitionResponse {
	return attributeDefinitionResponse{
		ID:          d.ID(),
		CategoryIDs: orEmptyStrings(d.CategoryIDs),
		Code:        d.Code(),
		Name:        d.Name(),
		GroupLabel:  d.GroupLabel(),
		DataType:    d.DataType(),
		Unit:        d.Unit(),
		Options:     d.Options(),
		OrderIndex:  d.OrderIndex(),
		IsRequired:  d.IsRequired(),
		CreatedAt:   d.CreatedAt(),
		UpdatedAt:   d.UpdatedAt(),
		DeletedAt:   d.DeletedAt(),
	}
}

// toAttributeDefinitionResponsePlain renders Create/Update's return value —
// neither operation touches category_attribute_definitions, so CategoryIDs
// is always empty here; GetByID is the source of truth for a definition's
// current category associations.
func toAttributeDefinitionResponsePlain(d *domain.AttributeDefinition) attributeDefinitionResponse {
	return attributeDefinitionResponse{
		ID:          d.ID(),
		Code:        d.Code(),
		Name:        d.Name(),
		GroupLabel:  d.GroupLabel(),
		DataType:    d.DataType(),
		Unit:        d.Unit(),
		Options:     d.Options(),
		OrderIndex:  d.OrderIndex(),
		IsRequired:  d.IsRequired(),
		CreatedAt:   d.CreatedAt(),
		UpdatedAt:   d.UpdatedAt(),
		DeletedAt:   d.DeletedAt(),
		CategoryIDs: []string{},
	}
}

func toAttributeDefinitionResponseList(defs []*domain.AttributeDefinitionWithCategories) []attributeDefinitionResponse {
	result := make([]attributeDefinitionResponse, len(defs))
	for i, d := range defs {
		result[i] = toAttributeDefinitionResponse(d)
	}
	return result
}

func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

type createAttributeDefinitionRequest struct {
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

type attachCategoriesRequest struct {
	CategoryIDs []string `json:"category_ids"`
}
