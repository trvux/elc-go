-- product_attribute_values is product's own join table against
-- attribute_definitions (internal/attribute) — same cross-module ownership
-- pattern as product_tags (000003_add_product_tags.up.sql): the table lives
-- in product's migrations and is read/written directly by product's own
-- infrastructure layer via raw SQL, never by calling into the attribute
-- module's Go code.
--
-- Exactly one of value_text/value_number/value_boolean is populated,
-- matching attribute_definitions.data_type ('select' also uses value_text).
-- Not enforced by a CHECK constraint here — application-layer responsibility,
-- same as how product_variants doesn't CHECK sale_price < original_price.
CREATE TABLE IF NOT EXISTS product_attribute_values (
    id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id              uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_definition_id uuid NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    value_text              text,
    value_number            numeric,
    value_boolean           boolean,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    deleted_at              timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS product_attribute_values_unique_active
    ON product_attribute_values (product_id, attribute_definition_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_product_attribute_values_attribute_definition_id
    ON product_attribute_values (attribute_definition_id);
