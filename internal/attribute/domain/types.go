package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

const (
	DataTypeNumber  = "number"
	DataTypeText    = "text"
	DataTypeBoolean = "boolean"
	DataTypeSelect  = "select"
)

func validDataType(dataType string) bool {
	switch dataType {
	case DataTypeNumber, DataTypeText, DataTypeBoolean, DataTypeSelect:
		return true
	}
	return false
}

// AttributeDefinition replaces the old free-text spec label a product
// admin form used to let staff type by hand (see docs/product-v2-design.md
// for the label-drift problem this fixes) — mirrors Shopify's Metafield
// Definition: defined once, reusable, typed. category_id nil means the
// definition applies across every category (e.g. "Xuất xứ").
type AttributeDefinition struct {
	id         string
	categoryID *string
	code       string
	name       string
	groupLabel *string
	dataType   string
	unit       *string
	options    []string
	orderIndex int
	isRequired bool
	createdAt  time.Time
	updatedAt  time.Time
	deletedAt  *time.Time
}

func NewAttributeDefinition(
	categoryID *string,
	code, name string,
	groupLabel *string,
	dataType string,
	unit *string,
	options []string,
	orderIndex int,
	isRequired bool,
) (*AttributeDefinition, error) {
	fields := map[string][]string{}

	if code == "" {
		fields["code"] = []string{"code is required"}
	}
	if name == "" {
		fields["name"] = []string{"name is required"}
	}
	if !validDataType(dataType) {
		fields["data_type"] = []string{"data_type must be one of: number, text, boolean, select"}
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &AttributeDefinition{
		categoryID: categoryID,
		code:       code,
		name:       name,
		groupLabel: groupLabel,
		dataType:   dataType,
		unit:       unit,
		options:    options,
		orderIndex: orderIndex,
		isRequired: isRequired,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// RehydrateAttributeDefinition reconstructs from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateAttributeDefinition(
	id string,
	categoryID *string,
	code, name string,
	groupLabel *string,
	dataType string,
	unit *string,
	options []string,
	orderIndex int,
	isRequired bool,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *AttributeDefinition {
	return &AttributeDefinition{
		id: id, categoryID: categoryID, code: code, name: name, groupLabel: groupLabel,
		dataType: dataType, unit: unit, options: options, orderIndex: orderIndex, isRequired: isRequired,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (a *AttributeDefinition) ID() string            { return a.id }
func (a *AttributeDefinition) CategoryID() *string   { return a.categoryID }
func (a *AttributeDefinition) Code() string          { return a.code }
func (a *AttributeDefinition) Name() string          { return a.name }
func (a *AttributeDefinition) GroupLabel() *string   { return a.groupLabel }
func (a *AttributeDefinition) DataType() string      { return a.dataType }
func (a *AttributeDefinition) Unit() *string         { return a.unit }
func (a *AttributeDefinition) Options() []string     { return a.options }
func (a *AttributeDefinition) OrderIndex() int       { return a.orderIndex }
func (a *AttributeDefinition) IsRequired() bool      { return a.isRequired }
func (a *AttributeDefinition) CreatedAt() time.Time  { return a.createdAt }
func (a *AttributeDefinition) UpdatedAt() time.Time  { return a.updatedAt }
func (a *AttributeDefinition) DeletedAt() *time.Time { return a.deletedAt }

func (a *AttributeDefinition) IsDeleted() bool { return a.deletedAt != nil }

// Update — category_id and data_type are immutable after creation (same
// "immutable after creation" rule product_lines.code and brand.brandID
// already follow) since changing either would orphan existing
// product_attribute_values rows with a mismatched type/scope.
func (a *AttributeDefinition) Update(
	name string,
	groupLabel *string,
	unit *string,
	options []string,
	orderIndex int,
	isRequired bool,
) error {
	if name == "" {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": {"name is required"}})
	}
	a.name = name
	a.groupLabel = groupLabel
	a.unit = unit
	a.options = options
	a.orderIndex = orderIndex
	a.isRequired = isRequired
	a.updatedAt = time.Now()
	return nil
}

func (a *AttributeDefinition) MarkDeleted(deletedAt time.Time) { a.deletedAt = &deletedAt }

func (a *AttributeDefinition) Restore() {
	a.deletedAt = nil
	a.updatedAt = time.Now()
}

type CreateAttributeDefinitionInput struct {
	CategoryID *string
	Code       string
	Name       string
	GroupLabel *string
	DataType   string
	Unit       *string
	Options    []string
	OrderIndex int
	IsRequired bool
}

type UpdateAttributeDefinitionInput struct {
	ID         string
	Name       *string
	GroupLabel *string
	Unit       *string
	Options    *[]string
	OrderIndex *int
	IsRequired *bool
}

type AttributeDefinitionFilter struct {
	CategoryID *string
	// IncludeGlobal also returns category_id IS NULL definitions alongside
	// the CategoryID filter's matches — used by the product form to fetch
	// "this category's attributes plus every universal one" in one call.
	IncludeGlobal  bool
	IncludeDeleted bool
}
