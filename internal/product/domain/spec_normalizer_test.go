package domain

import (
	"slices"
	"testing"
)

func containsFacet(facets []string, facet string) bool {
	return slices.Contains(facets, facet)
}

func TestNormalizeProductSpecs_CapacityFromName(t *testing.T) {
	facets := NormalizeProductSpecs("Máy lạnh Daikin 1.5HP Inverter", nil)

	if !containsFacet(facets, "Công suất::1.5 HP") {
		t.Errorf("expected capacity facet '1.5 HP' from name, got %+v", facets)
	}
}

func TestNormalizeProductSpecs_InverterDetectionFromName(t *testing.T) {
	facets := NormalizeProductSpecs("Máy lạnh Daikin 1.5HP Inverter", nil)

	// "Inverter" alone in the name isn't cooling-direction/gas/capacity text
	// handled by name-derived facets (only cooling direction, capacity, gas
	// are derived from name) — Công nghệ only comes from spec items, so this
	// asserts the *absence* of a bogus Công nghệ facet from name text alone.
	for _, f := range facets {
		if f == "Công nghệ::Có Inverter" {
			t.Errorf("did not expect a Công nghệ facet derived from name text, got %+v", facets)
		}
	}
}

func TestNormalizeProductSpecs_CoolingDirectionFromName(t *testing.T) {
	facets := NormalizeProductSpecs("Máy lạnh treo tường Daikin 1HP - một chiều Inverter", nil)

	if !containsFacet(facets, "Số chiều::1 Chiều") {
		t.Errorf("expected cooling direction facet '1 Chiều', got %+v", facets)
	}
	if !containsFacet(facets, "Công suất::1 HP") {
		t.Errorf("expected capacity facet '1 HP', got %+v", facets)
	}
}

func TestNormalizeProductSpecs_GasTypeFromName(t *testing.T) {
	facets := NormalizeProductSpecs("Máy lạnh Daikin Gas R32 Inverter", nil)

	if !containsFacet(facets, "Loại Gas::Gas R32") {
		t.Errorf("expected gas type facet 'Gas R32', got %+v", facets)
	}
}

func TestNormalizeProductSpecs_SpecItemWithSubItems(t *testing.T) {
	specs := []SpecItem{
		{
			Label: "Công suất làm lạnh",
			Items: []SpecSubItem{
				{Label: "", Value: "2 Hp"},
				{Label: "", Value: "18.000 Btu/h"},
			},
		},
	}

	facets := NormalizeProductSpecs("Máy điều hòa tủ đứng", specs)

	if !containsFacet(facets, "Công suất::2 HP") {
		t.Errorf("expected capacity facet '2 HP' derived from sub-item, got %+v", facets)
	}
}

func TestNormalizeProductSpecs_InverterSpecItem(t *testing.T) {
	value := "Có"
	specs := []SpecItem{{Label: "Inverter", Value: &value}}

	facets := NormalizeProductSpecs("Máy lạnh", specs)

	if !containsFacet(facets, "Công nghệ::Có Inverter") {
		t.Errorf("expected 'Công nghệ::Có Inverter', got %+v", facets)
	}
}

func TestNormalizeProductSpecs_NonInverterSpecItem(t *testing.T) {
	value := "Không"
	specs := []SpecItem{{Label: "Công nghệ tiết kiệm điện", Value: &value}}

	facets := NormalizeProductSpecs("Máy lạnh", specs)

	if !containsFacet(facets, "Công nghệ::Không Inverter") {
		t.Errorf("expected 'Công nghệ::Không Inverter', got %+v", facets)
	}
}

func TestNormalizeProductSpecs_GasSpecItem(t *testing.T) {
	value := "R410A"
	specs := []SpecItem{{Label: "Môi chất làm lạnh", Value: &value}}

	facets := NormalizeProductSpecs("Máy lạnh", specs)

	if !containsFacet(facets, "Loại Gas::Gas R410A") {
		t.Errorf("expected 'Loại Gas::Gas R410A', got %+v", facets)
	}
}

func TestNormalizeProductSpecs_FilterEfficiencySpecItem(t *testing.T) {
	value := "99.97%"
	specs := []SpecItem{{Label: "Hiệu suất lọc không khí", Value: &value}}

	facets := NormalizeProductSpecs("Máy lọc không khí", specs)

	if !containsFacet(facets, "Hiệu suất lọc::Trên 99.9% (Ultra)") {
		t.Errorf("expected facet %q, got %+v", "Hiệu suất lọc::Trên 99.9% (Ultra)", facets)
	}
}

func TestNormalizeProductSpecs_UnrecognizedLabelIsSkippedSilently(t *testing.T) {
	value := "some value"
	specs := []SpecItem{{Label: "Xuất xứ", Value: &value}}

	// Name deliberately has none of the HVAC keywords getCoolingDirection/
	// getCapacityFromText/getGasType look for (no "lạnh", "chiều", digit+hp,
	// gas code) — otherwise the name-derived facets would make this test's
	// "no facets at all" assertion flaky for unrelated reasons.
	facets := NormalizeProductSpecs("Sản phẩm demo", specs)

	if len(facets) != 0 {
		t.Errorf("expected unrecognized spec label to be skipped, got %+v", facets)
	}
}

func TestNormalizeProductSpecs_EmptyValueIsSkipped(t *testing.T) {
	value := ""
	specs := []SpecItem{{Label: "Inverter", Value: &value}}

	facets := NormalizeProductSpecs("Sản phẩm demo", specs)

	if len(facets) != 0 {
		t.Errorf("expected empty spec value to produce no facet, got %+v", facets)
	}
}

func TestNormalizeProductSpecs_Deduplicates(t *testing.T) {
	valueA := "Có Inverter"
	valueB := "Có"
	specs := []SpecItem{
		{Label: "Inverter", Value: &valueA},
		{Label: "Công nghệ", Value: &valueB},
	}

	facets := NormalizeProductSpecs("Máy lạnh", specs)

	count := 0
	for _, f := range facets {
		if f == "Công nghệ::Có Inverter" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'Công nghệ::Có Inverter' to be deduplicated, got %d occurrences in %+v", count, facets)
	}
}

func TestNormalizeSpecValue_AirFilterSingleWordChecksOnly(t *testing.T) {
	// Documents the ported-verbatim TS bug: multi-word raw checks ("bụi mịn",
	// "khử mùi", "tiêu chuẩn", "lọc thô") can never match because `lower` has
	// whitespace stripped first. Single-word checks (hepa, pm2.5, ion, mesh,
	// nanoe) still work.
	hepa := "HEPA"
	specs := []SpecItem{{Label: "Bộ lọc", Value: &hepa}}
	facets := NormalizeProductSpecs("Máy lọc không khí", specs)
	if !containsFacet(facets, "Lọc không khí::Lọc HEPA") {
		t.Errorf("expected 'Lọc không khí::Lọc HEPA', got %+v", facets)
	}

	pm25 := "PM2.5"
	specs2 := []SpecItem{{Label: "Bộ lọc", Value: &pm25}}
	facets2 := NormalizeProductSpecs("Máy lọc không khí", specs2)
	if !containsFacet(facets2, "Lọc không khí::Lọc bụi mịn PM2.5") {
		t.Errorf("expected 'Lọc không khí::Lọc bụi mịn PM2.5', got %+v", facets2)
	}
}
