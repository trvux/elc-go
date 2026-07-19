package application

import (
	"context"
	"fmt"

	attributedomain "github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// validateAttributeValues enforces two rules cutting across TRA-15/TRA-18:
// every value's shape must match its attribute definition's data_type
// (multiselect stores 0..N of ValueOptions, not a single scalar; the other
// four types store exactly one scalar field), and every is_required
// definition applicable to categoryID (attached to it, or global) must have
// a corresponding value — a save request violating either is rejected with
// a field-keyed validation error, not silently accepted or left to a UI
// asterisk (see internal/attribute's is_required doc comment). This is the
// first cross-module application-layer call in the codebase — every other
// cross-module read so far is a raw SQL join in an infrastructure layer,
// but "is this value required by the category's rules" is a business rule
// attribute owns, not display data product can just join for.
func validateAttributeValues(
	ctx context.Context,
	attributeRepo attributedomain.AttributeDefinitionRepository,
	categoryID string,
	values []domain.ProductAttributeValueInput,
) error {
	applicable, err := attributeRepo.GetApplicableForCategory(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("validateAttributeValues: %w", err)
	}
	defsByID := make(map[string]*attributedomain.AttributeDefinition, len(applicable))
	for _, d := range applicable {
		defsByID[d.ID()] = d
	}

	fields := map[string][]string{}
	seen := map[string]bool{}
	for _, v := range values {
		def, ok := defsByID[v.AttributeDefinitionID]
		if !ok {
			fields["attribute_values"] = append(fields["attribute_values"],
				fmt.Sprintf("attribute definition %s is not applicable to this product's category", v.AttributeDefinitionID))
			continue
		}
		seen[def.ID()] = true

		key := "attribute_values." + def.Code()
		switch def.DataType() {
		case attributedomain.DataTypeNumber:
			if v.ValueNumber == nil {
				fields[key] = append(fields[key], "value_number is required for data_type number")
			}
		case attributedomain.DataTypeText, attributedomain.DataTypeSelect:
			if v.ValueText == nil {
				fields[key] = append(fields[key], "value_text is required for data_type "+def.DataType())
			}
		case attributedomain.DataTypeBoolean:
			if v.ValueBoolean == nil {
				fields[key] = append(fields[key], "value_boolean is required for data_type boolean")
			}
		case attributedomain.DataTypeMultiselect:
			if len(v.ValueOptions) == 0 {
				fields[key] = append(fields[key], "value_options must have at least one value for data_type multiselect")
			}
		}
	}

	for _, d := range applicable {
		if d.IsRequired() && !seen[d.ID()] {
			key := "attribute_values." + d.Code()
			fields[key] = append(fields[key], "this attribute is required for the product's category")
		}
	}

	if len(fields) > 0 {
		return apperr.NewValidationError("validation failed", fields)
	}
	return nil
}

// attributeValueRefsToInputs converts existing read-shape attribute values
// back into the write-shape, so UpdateProduct can re-validate against the
// effective attribute set even when the caller didn't resend
// AttributeValues (nil = "leave untouched" per UpdateProductInput's
// convention, same as Options/Variants).
func attributeValueRefsToInputs(refs []domain.AttributeValueRef) []domain.ProductAttributeValueInput {
	inputs := make([]domain.ProductAttributeValueInput, len(refs))
	for i, r := range refs {
		inputs[i] = domain.ProductAttributeValueInput{
			AttributeDefinitionID: r.AttributeDefinitionID,
			ValueText:             r.ValueText,
			ValueNumber:           r.ValueNumber,
			ValueBoolean:          r.ValueBoolean,
			ValueOptions:          r.ValueOptions,
		}
	}
	return inputs
}
