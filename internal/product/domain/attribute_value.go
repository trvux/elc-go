package domain

// ProductAttributeValueInput is the write-shape for one structured spec
// value — exactly one of ValueText/ValueNumber/ValueBoolean/ValueOptions is
// populated, matching the referenced attribute_definitions.data_type
// (number/text/boolean use their matching scalar field; select uses
// ValueText; multiselect uses ValueOptions — 0..N of the definition's
// `options`, not a single scalar). Validated against the definition's actual
// data_type in application.validateAttributeValues, not by the frontend
// form alone. AttributeDefinitionID references a row owned by the sibling
// internal/attribute module — same cross-module-by-ID reference pattern as
// TagIDs on CreateProductInput.
type ProductAttributeValueInput struct {
	AttributeDefinitionID string
	ValueText             *string
	ValueNumber           *float64
	ValueBoolean          *bool
	ValueOptions          []string
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
	ValueOptions          []string
}
