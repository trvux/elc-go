package domain

import (
	"testing"
	"time"
)

func TestService_SalePrice(t *testing.T) {
	tests := []struct {
		name            string
		originalPrice   *int64
		discountPercent *int
		want            *int64
	}{
		{name: "no price set", originalPrice: nil, discountPercent: nil, want: nil},
		{name: "price without discount", originalPrice: ptrInt64(100000), discountPercent: nil, want: nil},
		{name: "20 percent off 100000", originalPrice: ptrInt64(100000), discountPercent: ptrInt(20), want: ptrInt64(80000)},
		{name: "0 percent off", originalPrice: ptrInt64(50000), discountPercent: ptrInt(0), want: ptrInt64(50000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RehydrateService(
				"id-1", "Title", "", "slug", nil, nil,
				tt.originalPrice, tt.discountPercent, nil, nil, "dich-vu", nil, nil, nil,
				nil, nil, nil, false, true, 0,
				time.Now(), time.Now(), nil,
			)

			got := s.SalePrice()
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("SalePrice() = %v, want %v", got, tt.want)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("SalePrice() = %d, want %d", *got, *tt.want)
			}
		})
	}
}

func TestService_UpdatePricing_NeverGoesStale(t *testing.T) {
	s := RehydrateService(
		"id-1", "Title", "", "slug", nil, nil,
		ptrInt64(100000), ptrInt(10), nil, nil, "dich-vu", nil, nil, nil,
		nil, nil, nil, false, true, 0,
		time.Now(), time.Now(), nil,
	)

	if *s.SalePrice() != 90000 {
		t.Fatalf("expected initial sale price 90000, got %d", *s.SalePrice())
	}

	// Simulate an update that only touches discountPercent — the exact case
	// that went stale in the old TS code. Update() resolves OriginalPrice/
	// DiscountPercent together internally (see its doc comment), so a
	// request that only sets DiscountPercent still can't go stale.
	if err := s.Update(UpdateServiceInput{DiscountPercent: ptrInt(50)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *s.SalePrice() != 50000 {
		t.Errorf("expected recomputed sale price 50000, got %d", *s.SalePrice())
	}
}

func TestService_Update_RejectsInvalidDiscountPercent(t *testing.T) {
	s := RehydrateService(
		"id-1", "Title", "", "slug", nil, nil,
		ptrInt64(100000), ptrInt(10), nil, nil, "dich-vu", nil, nil, nil,
		nil, nil, nil, false, true, 0,
		time.Now(), time.Now(), nil,
	)

	if err := s.Update(UpdateServiceInput{DiscountPercent: ptrInt(150)}); err == nil {
		t.Fatal("expected a validation error for discountPercent > 100")
	}
}

func TestService_Update_NoOpLeavesUpdatedAtUnchanged(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	s := RehydrateService(
		"id-1", "Title", "", "slug", nil, nil,
		nil, nil, nil, nil, "dich-vu", nil, nil, nil,
		nil, nil, nil, false, true, 0,
		createdAt, createdAt, nil,
	)

	if err := s.Update(UpdateServiceInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.UpdatedAt().Equal(createdAt) {
		t.Errorf("expected updatedAt unchanged on a no-op Update, got %v (was %v)", s.UpdatedAt(), createdAt)
	}
}

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }

func newTestService(t *testing.T, line string, prompts []string) *Service {
	t.Helper()
	s, err := NewService("Title", "slug", "", nil, nil, nil, nil, nil, nil, line, prompts, nil, nil, nil, nil, nil, false, true, 0)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return s
}

func TestNewService_LineDefaultsToDichVu(t *testing.T) {
	if got := newTestService(t, "", nil).Line(); got != LineDichVu {
		t.Errorf("Line() = %q, want %q", got, LineDichVu)
	}
}

func TestNewService_RejectsUnknownLine(t *testing.T) {
	_, err := NewService("Title", "slug", "", nil, nil, nil, nil, nil, nil, "khac", nil, nil, nil, nil, nil, nil, false, true, 0)
	if err == nil {
		t.Fatal("expected a validation error for an unknown line")
	}
}

func TestNewService_BriefPromptsAreTrimmedAndBlanksDropped(t *testing.T) {
	s := newTestService(t, LineThiCong, []string{"  Địa chỉ công trình ", "", "   ", "Diện tích"})
	got := s.BriefPrompts()
	want := []string{"Địa chỉ công trình", "Diện tích"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("BriefPrompts() = %q, want %q", got, want)
	}
}

func TestNewService_NoBriefPromptsIsEmptyNotNil(t *testing.T) {
	if got := newTestService(t, LineThiCong, nil).BriefPrompts(); got == nil || len(got) != 0 {
		t.Errorf("BriefPrompts() = %#v, want empty non-nil slice", got)
	}
}

func TestService_Update_Line(t *testing.T) {
	s := newTestService(t, LineDichVu, nil)

	bad := "khac"
	if err := s.Update(UpdateServiceInput{Line: &bad}); err == nil {
		t.Fatal("expected an error for an unknown line")
	}
	if s.Line() != LineDichVu {
		t.Errorf("line changed despite validation error: %q", s.Line())
	}

	good := LineThiCong
	if err := s.Update(UpdateServiceInput{Line: &good}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Line() != LineThiCong {
		t.Errorf("Line() = %q, want %q", s.Line(), LineThiCong)
	}
}

func TestService_Update_BriefPrompts(t *testing.T) {
	s := newTestService(t, LineThiCong, []string{"A", "B"})

	// Not part of the request: untouched.
	if err := s.Update(UpdateServiceInput{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(s.BriefPrompts()) != 2 {
		t.Fatalf("prompts changed by an unrelated update: %q", s.BriefPrompts())
	}

	// Replaced, cleaned.
	if err := s.Update(UpdateServiceInput{BriefPrompts: []string{" C ", ""}}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := s.BriefPrompts(); len(got) != 1 || got[0] != "C" {
		t.Fatalf("BriefPrompts() = %q, want [C]", got)
	}

	// An empty (non-nil) list clears it.
	if err := s.Update(UpdateServiceInput{BriefPrompts: []string{}}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(s.BriefPrompts()) != 0 {
		t.Fatalf("expected the list to be cleared, got %q", s.BriefPrompts())
	}
}
