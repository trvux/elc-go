package domain

import (
	"testing"
	"time"
)

func TestService_Update_NoOpLeavesUpdatedAtUnchanged(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	s := RehydrateService(
		"id-1", "Title", "", "slug", nil, nil, "dich-vu", nil,
		nil, nil, nil, nil, nil, true, 0,
		createdAt, createdAt, nil,
	)

	if err := s.Update(UpdateServiceInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.UpdatedAt().Equal(createdAt) {
		t.Errorf("expected updatedAt unchanged on a no-op Update, got %v (was %v)", s.UpdatedAt(), createdAt)
	}
}

func newTestService(t *testing.T, line string, prompts []string) *Service {
	t.Helper()
	s, err := NewService("Title", "slug", "", nil, nil, line, prompts, nil, nil, nil, nil, nil, true, 0)
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
	_, err := NewService("Title", "slug", "", nil, nil, "khac", nil, nil, nil, nil, nil, nil, true, 0)
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
