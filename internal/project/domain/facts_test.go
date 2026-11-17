package domain

import (
	"strings"
	"testing"
	"time"
)

func strPtr(s string) *string { return &s }

func newTestProject(t *testing.T, clientName string, facts ProjectFacts) *Project {
	t.Helper()
	p, err := NewProject("T", "t", "", nil, nil, false, false, nil, nil, 0, nil, clientName, "", nil, "", "", facts)
	if err != nil {
		t.Fatalf("NewProject: %v", err)
	}
	return p
}

func TestClientNamePublic(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		client string
		facts  ProjectFacts
		want   bool
	}{
		{"empty name", "", ProjectFacts{ClientKind: strPtr(ClientKindBusiness)}, false},
		{"business is public", "Cong ty A", ProjectFacts{ClientKind: strPtr(ClientKindBusiness)}, true},
		{"person without consent", "Mrs. Tham", ProjectFacts{ClientKind: strPtr(ClientKindPerson)}, false},
		{"nil kind is treated as person", "Mrs. Tham", ProjectFacts{}, false},
		{"person with consent", "Mrs. Tham", ProjectFacts{ClientKind: strPtr(ClientKindPerson), ClientConsentAt: &now}, true},
		{"nil kind with consent", "Mrs. Tham", ProjectFacts{ClientConsentAt: &now}, true},
	}
	for _, c := range cases {
		if got := newTestProject(t, c.client, c.facts).ClientNamePublic(); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestPublishGaps_DateConsistency(t *testing.T) {
	done := time.Now()
	hasGap := func(p *Project, sub string) bool {
		for _, g := range PublishGaps(p, 1, 1) {
			if strings.Contains(g, sub) {
				return true
			}
		}
		return false
	}

	// date set but precision unknown -> ask for precision
	p := newTestProject(t, "", ProjectFacts{})
	p.completedAt = &done
	if !hasGap(p, "độ chính xác") {
		t.Error("a date with unknown precision should be a gap")
	}
	// unknown date + unknown precision is a legitimate state, not a gap
	if hasGap(newTestProject(t, "", ProjectFacts{}), "ngày hoàn thành") {
		t.Error("unknown date with unknown precision must be allowed")
	}
	// precision claims a date that is missing
	if !hasGap(newTestProject(t, "", ProjectFacts{CompletedPrecision: PrecisionMonth}), "ngày hoàn thành") {
		t.Error("a precision without a date should be a gap")
	}
}

func TestFactsNormalizeBrands(t *testing.T) {
	p := newTestProject(t, "", ProjectFacts{Brands: []string{" Daikin ", "daikin", "", "Menred"}})
	got := p.Facts().Brands
	if len(got) != 2 || got[0] != "Daikin" || got[1] != "Menred" {
		t.Errorf("brands should be trimmed/de-duplicated, got %v", got)
	}
}
