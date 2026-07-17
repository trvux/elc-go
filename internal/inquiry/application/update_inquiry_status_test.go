package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func seedInquiry(t *testing.T, repo *fakeInquiryRepository) *domain.Inquiry {
	t.Helper()
	inquiry, err := CreateInquiry(context.Background(), repo, domain.CreateInquiryInput{
		Name:  "Nguyen Van A",
		Phone: "0901234567",
	})
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	return inquiry
}

func TestUpdateInquiryStatus(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiry(t, repo)
	ctx := context.Background()

	updated, err := UpdateInquiryStatus(ctx, repo, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusContacted,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status() != domain.InquiryStatusContacted {
		t.Errorf("expected status contacted, got %s", updated.Status())
	}
}

func TestUpdateInquiryStatus_NotFound(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := UpdateInquiryStatus(ctx, repo, UpdateInquiryStatusInput{
		ID:     "does-not-exist",
		Status: domain.InquiryStatusContacted,
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUpdateInquiryStatus_InvalidStatus(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiry(t, repo)
	ctx := context.Background()

	_, err := UpdateInquiryStatus(ctx, repo, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: "bogus",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestUpdateInquiryStatus_InternalNoteOnly(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiry(t, repo)
	ctx := context.Background()

	note := "Called, no answer"
	updated, err := UpdateInquiryStatus(ctx, repo, UpdateInquiryStatusInput{
		ID:           seeded.ID(),
		InternalNote: &note,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.InternalNote() == nil || *updated.InternalNote() != note {
		t.Errorf("expected internal note %q, got %v", note, updated.InternalNote())
	}
	if updated.Status() != domain.InquiryStatusNew {
		t.Errorf("expected status to remain unchanged (new), got %s", updated.Status())
	}
}
