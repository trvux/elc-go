-- FBFC125DVM9/FBFC140DVM9 (Daikin giấu trần nối ống gió): kich_thuoc_dan_nong
-- was never populated (see cmd/audit-attribute-values) — the original
-- migration could only capture one of two identically-labeled "Kích
-- thước" spec entries. Confirmed via daikinco.vn/dienmaythienphu.vn: the
-- outdoor unit (RZFC140DY1, and the shared casing for the 125 variant)
-- is 990x940x320mm — the exact 2nd raw specs value already sitting on
-- both products, verified against a real published spec sheet, not
-- guessed. kich_thuoc_dan_lanh (245x1,400x800mm) was already correct.
BEGIN;

INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_text)
SELECT p.id, ad.id, '990x940x320 (mm)'
FROM products p
JOIN attribute_definitions ad ON ad.category_id = p.category_id AND ad.code = 'kich_thuoc_dan_nong' AND ad.deleted_at IS NULL
WHERE p.id IN ('938a55c3-f749-49f5-af74-e7fd8f91bad1', 'bd6d3430-889f-491a-a94a-f2dc7f98138f')
ON CONFLICT (product_id, attribute_definition_id) WHERE deleted_at IS NULL DO NOTHING;

SELECT p.name, av.value_text
FROM product_attribute_values av
JOIN attribute_definitions ad ON ad.id = av.attribute_definition_id AND ad.code = 'kich_thuoc_dan_nong'
JOIN products p ON p.id = av.product_id
WHERE p.id IN ('938a55c3-f749-49f5-af74-e7fd8f91bad1', 'bd6d3430-889f-491a-a94a-f2dc7f98138f');

COMMIT;
