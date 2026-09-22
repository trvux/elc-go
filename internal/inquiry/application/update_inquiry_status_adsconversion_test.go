package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func seedInquiryWithGAClientID(t *testing.T, repo *fakeInquiryRepository, gaClientID string) *domain.Inquiry {
	t.Helper()
	inquiry, err := CreateInquiry(context.Background(), repo, domain.CreateInquiryInput{
		Name:       "Nguyen Van A",
		Phone:      "0901234567",
		GAClientID: &gaClientID,
	})
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	return inquiry
}

func TestUpdateInquiryStatus_PushesCloseConvertLeadOnFirstConversion(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: true}
	ctx := context.Background()

	updated, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(notifier.calls) != 1 {
		t.Fatalf("expected exactly 1 SendCloseConvertLead call, got %d", len(notifier.calls))
	}
	if notifier.calls[0].gaClientID != "1234567890.9876543210" {
		t.Errorf("expected the lead's ga_client_id to be used, got %q", notifier.calls[0].gaClientID)
	}
	if updated.AdsConversionSyncedAt() == nil {
		t.Error("expected ads_conversion_synced_at to be set after a successful push")
	}
}

func TestUpdateInquiryStatus_PassesConversionValueSavedJustBefore(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: true}
	ctx := context.Background()

	// Mirrors InquiryManagement's real save order: details (incl.
	// conversion_value) land first, then the status change.
	value := 15_000_000.0
	if _, err := UpdateInquiryDetails(ctx, repo, UpdateInquiryDetailsInput{
		ID:              seeded.ID(),
		Name:            seeded.Name(),
		Phone:           seeded.Phone(),
		ConversionValue: &value,
	}); err != nil {
		t.Fatalf("update details: expected no error, got %v", err)
	}

	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(notifier.calls) != 1 {
		t.Fatalf("expected exactly 1 push, got %d", len(notifier.calls))
	}
	if notifier.calls[0].value == nil || *notifier.calls[0].value != value {
		t.Errorf("expected the conversion value saved just before this call (%v) to reach the push, got %v", value, notifier.calls[0].value)
	}
}

func TestUpdateInquiryStatus_SkipsPushWhenAlreadyConverted(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: true}
	ctx := context.Background()

	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	}); err != nil {
		t.Fatalf("first conversion: expected no error, got %v", err)
	}

	note := "Đổi ghi chú sau khi đã chốt"
	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:           seeded.ID(),
		Status:       domain.InquiryStatusConverted,
		InternalNote: &note,
	}); err != nil {
		t.Fatalf("re-save while already converted: expected no error, got %v", err)
	}

	if len(notifier.calls) != 1 {
		t.Errorf("expected close_convert_lead pushed exactly once even after re-saving an already-converted lead, got %d pushes", len(notifier.calls))
	}
}

func TestUpdateInquiryStatus_SkipsPushWhenNoGAClientID(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiry(t, repo) // no GAClientID
	notifier := &fakeNotifier{configured: true}
	ctx := context.Background()

	updated, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	})
	if err != nil {
		t.Fatalf("expected no error (a missing ga_client_id must never fail the status update), got %v", err)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("expected no push attempted without a ga_client_id, got %d", len(notifier.calls))
	}
	if updated.AdsConversionSyncedAt() != nil {
		t.Error("expected ads_conversion_synced_at to stay nil when nothing was pushed")
	}
}

func TestUpdateInquiryStatus_SkipsPushWhenNotifierNotConfigured(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: false}
	ctx := context.Background()

	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("expected no push attempted when the notifier reports not configured, got %d", len(notifier.calls))
	}
}

func TestUpdateInquiryStatus_StatusUpdateSucceedsEvenWhenPushFails(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: true, failWith: errors.New("network error")}
	ctx := context.Background()

	updated, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{
		ID:     seeded.ID(),
		Status: domain.InquiryStatusConverted,
	})
	if err != nil {
		t.Fatalf("expected the status update to succeed even though the GA4 push failed, got %v", err)
	}
	if updated.Status() != domain.InquiryStatusConverted {
		t.Errorf("expected status converted regardless of push failure, got %s", updated.Status())
	}
	if updated.AdsConversionSyncedAt() != nil {
		t.Error("expected ads_conversion_synced_at to stay nil after a failed push, for cmd/retry-ads-conversion-sync to find later")
	}
}

func TestUpdateInquiryStatus_ReopenedThenReconvertedPushesAgain(t *testing.T) {
	repo := newFakeInquiryRepository()
	seeded := seedInquiryWithGAClientID(t, repo, "1234567890.9876543210")
	notifier := &fakeNotifier{configured: true}
	ctx := context.Background()

	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{ID: seeded.ID(), Status: domain.InquiryStatusConverted}); err != nil {
		t.Fatalf("first conversion: expected no error, got %v", err)
	}
	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{ID: seeded.ID(), Status: domain.InquiryStatusNew}); err != nil {
		t.Fatalf("reopen: expected no error, got %v", err)
	}
	if _, err := UpdateInquiryStatus(ctx, repo, notifier, nil, UpdateInquiryStatusInput{ID: seeded.ID(), Status: domain.InquiryStatusConverted}); err != nil {
		t.Fatalf("re-conversion: expected no error, got %v", err)
	}

	if len(notifier.calls) != 2 {
		t.Errorf("expected a genuinely new conversion (after being reopened) to push again, got %d pushes", len(notifier.calls))
	}
}
