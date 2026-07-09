package domain

// ProductAttributeValueInput is the write-shape for one structured spec
// value — exactly one of ValueText/ValueNumber/ValueBoolean is populated,
// matching the referenced attribute_definitions.data_type (validated by the
// frontend form, which only ever renders one control per attribute; not
// re-validated here, same trust boundary as ProductVariantInput's price
// fields). AttributeDefinitionID references a row owned by the sibling
// internal/attribute module — same cross-module-by-ID reference pattern as
// TagIDs on CreateProductInput.
type ProductAttributeValueInput struct {
	AttributeDefinitionID string
	ValueText             *string
	ValueNumber           *float64
	ValueBoolean          *bool
}

// AttributeValueRef is the read-shape, denormalized with the attribute
// definition's display fields (name/unit/data_type/options/group) via a
// direct SQL join in product's own infrastructure layer — same cross-module
// read pattern as CategoryRef/BrandRef/TagRef (see ProductWithRelations),
// not a call into internal/attribute's Go code.
type AttributeValueRef struct {
	ID                    string
	AttributeDefinitionID string
	Code                  string
	Name                  string
	GroupLabel            *string
	DataType              string
	Unit                  *string
	Options               []string
	ValueText             *string
	ValueNumber           *float64
	ValueBoolean          *bool
}
