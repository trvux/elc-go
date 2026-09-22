package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func seedClickLead(t *testing.T, repo *fakeInquiryRepository) *domain.Inquiry {
	t.Helper()
	sessionID := "session-details-test"
	inquiry, err := RecordContactClick(context.Background(), repo, domain.CreateContactClickInput{
		Channel:   domain.ChannelZalo,
		SessionID: &sessionID,
	})
	if err != nil {
		t.Fatalf("seed click lead failed: %v", err)
	}
	return inquiry
}

func TestUpdateInquiryDetails_FillsInIdentityOnAClickOriginLead(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedClickLead(t, repo)
	ctx := context.Background()

	updated, err := UpdateInquiryDetails(ctx, repo, UpdateInquiryDetailsInput{
		ID:    seeded.ID(),
		Name:  "Nguyen Van A",
		Phone: "0901234567",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name() != "Nguyen Van A" || updated.Phone() != "0901234567" {
		t.Errorf("expected identity to be filled in, got name=%q phone=%q", updated.Name(), updated.Phone())
	}
}

func TestUpdateInquiryDetails_RejectsBlankIdentity(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedClickLead(t, repo)
	ctx := context.Background()

	_, err := UpdateInquiryDetails(ctx, repo, UpdateInquiryDetailsInput{ID: seeded.ID()})
	if err == nil {
		t.Fatal("expected validation error — once staff are setting identity, blank is always a mistake")
	}
}

func TestUpdateInquiryDetails_SetsConversionValue(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiry(t, repo)
	ctx := context.Background()

	value := 15_500_000.0
	updated, err := UpdateInquiryDetails(ctx, repo, UpdateInquiryDetailsInput{
		ID:              seeded.ID(),
		Name:            seeded.Name(),
		Phone:           seeded.Phone(),
		ConversionValue: &value,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.ConversionValue() == nil || *updated.ConversionValue() != value {
		t.Errorf("expected conversion value %v, got %v", value, updated.ConversionValue())
	}
}

func TestUpdateInquiryDetails_NotFound(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := UpdateInquiryDetails(ctx, repo, UpdateInquiryDetailsInput{
		ID:    "does-not-exist",
		Name:  "Nguyen Van A",
		Phone: "0901234567",
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}
