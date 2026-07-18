package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/product/domain"
)

// insertProductAttributeValues writes the structured spec values for a
// product — same "one INSERT with multiple VALUES rows" shape as
// insertProductTags. Called from both Create and Update (Update
// delete-then-reinserts, same convention as tags/variants).
func insertProductAttributeValues(ctx context.Context, tx pgx.Tx, productID string, values []domain.ProductAttributeValueInput) error {
	if len(values) == 0 {
		return nil
	}
	for _, v := range values {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_text, value_number, value_boolean, value_options)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			productID, v.AttributeDefinitionID, v.ValueText, v.ValueNumber, v.ValueBoolean, orEmptyStrings(v.ValueOptions),
		); err != nil {
			return fmt.Errorf("product repository insertProductAttributeValues: %w", err)
		}
	}
	return nil
}

// attachAttributeValuesToProducts loads the structured spec values for one or
// more products — called with a single-element slice from GetByID/GetBySlug,
// and with a real multi-product slice from GetByIDsWithAttributeValues
// (Comparison). Plain GetAll/GetByIDs list reads still don't call this (specs
// aren't needed for list/card rendering, same "list reads don't need this"
// reasoning as attachVariantTree, see productJoin's doc comment). Joins
// directly into attribute_definitions (owned by the sibling internal/
// attribute module) for the display fields — same cross-module SQL-join
// pattern already used for CategoryRef/BrandRef/TagRef, not a call into
// attribute's Go code.
func attachAttributeValuesToProducts(ctx context.Context, pool *pgxpool.Pool, products []*domain.ProductWithRelations) error {
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID()
	}

	rows, err := pool.Query(ctx, `
		SELECT pav.id, pav.product_id, ad.id, ad.code, ad.name, ad.group_label, ad.data_type, ad.unit, ad.options,
		       pav.value_text, pav.value_number, pav.value_boolean, pav.value_options
		FROM product_attribute_values pav
		JOIN attribute_definitions ad ON ad.id = pav.attribute_definition_id AND ad.deleted_at IS NULL
		WHERE pav.product_id = ANY($1) AND pav.deleted_at IS NULL
		ORDER BY ad.group_label NULLS FIRST, ad.order_index ASC`, ids)
	if err != nil {
		return fmt.Errorf("product repository attachAttributeValues: %w", err)
	}
	defer rows.Close()

	byProductID := map[string][]domain.AttributeValueRef{}
	for rows.Next() {
		var (
			id, productID, attributeDefinitionID, code, name, dataType string
			groupLabel, unit                                           *string
			options                                                    []string
			valueText                                                  *string
			valueNumber                                                *float64
			valueBoolean                                               *bool
			valueOptions                                               []string
		)
		if err := rows.Scan(
			&id, &productID, &attributeDefinitionID, &code, &name, &groupLabel, &dataType, &unit, &options,
			&valueText, &valueNumber, &valueBoolean, &valueOptions,
		); err != nil {
			return fmt.Errorf("product repository attachAttributeValues scan: %w", err)
		}
		byProductID[productID] = append(byProductID[productID], domain.AttributeValueRef{
			ID: id, AttributeDefinitionID: attributeDefinitionID, Code: code, Name: name,
			GroupLabel: groupLabel, DataType: dataType, Unit: unit, Options: options,
			ValueText: valueText, ValueNumber: valueNumber, ValueBoolean: valueBoolean,
			ValueOptions: valueOptions,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("product repository attachAttributeValues rows: %w", err)
	}

	for _, p := range products {
		p.AttributeValues = byProductID[p.ID()]
	}
	return nil
}
