package application

import (
	"context"

	"go.uber.org/zap"

	adsconversiondomain "github.com/trvux/elc-go/internal/adsconversion/domain"
	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type UpdateInquiryStatusInput struct {
	ID           string
	Status       domain.InquiryStatus
	InternalNote *string
}

// UpdateInquiryStatus moves a lead through the manual follow-up pipeline
// (new -> contacted -> converted/closed, or reopened) and/or updates the
// staff-only note. Status is applied through the entity's own named
// mutators, never set directly, so invalid transitions can't be represented.
//
// notifier/log push the close_convert_lead signal to GA4/Ads exactly once,
// at the moment a lead first transitions INTO converted — never a
// dependency the caller must satisfy: pass adsconversion infrastructure's
// NoopNotifier (or any Notifier reporting !IsConfigured()) to skip this
// safely, and a failed push here never fails the status update itself (the
// lead IS converted regardless of whether Ads found out) — see
// cmd/retry-ads-conversion-sync for retrying it afterwards.
func UpdateInquiryStatus(
	ctx context.Context,
	repo domain.InquiryRepository,
	notifier adsconversiondomain.Notifier,
	log *zap.Logger,
	input UpdateInquiryStatusInput,
) (*domain.Inquiry, error) {
	inquiry, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if inquiry == nil {
		return nil, apperr.NewNotFoundError("inquiry")
	}

	wasAlreadyConverted := inquiry.Status() == domain.InquiryStatusConverted

	if input.Status != "" {
		if !input.Status.IsValid() {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{
				"status": {"invalid status"},
			})
		}
		switch input.Status {
		case domain.InquiryStatusContacted:
			inquiry.MarkContacted()
		case domain.InquiryStatusConverted:
			inquiry.MarkConverted()
		case domain.InquiryStatusClosed:
			inquiry.Close()
		case domain.InquiryStatusNew:
			inquiry.Reopen()
		}
	}

	if input.InternalNote != nil {
		inquiry.SetInternalNote(input.InternalNote)
	}

	updated, err := repo.Update(ctx, inquiry)
	if err != nil {
		return nil, err
	}

	// justConverted, not "status is converted": re-saving an already-
	// converted lead (e.g. staff edits the note afterwards) must not
	// re-push the same sale to GA4/Ads a second time.
	justConverted := input.Status == domain.InquiryStatusConverted && !wasAlreadyConverted
	if justConverted && notifier != nil && notifier.IsConfigured() {
		syncConversionToAds(ctx, repo, notifier, log, updated)
	}

	return updated, nil
}

func syncConversionToAds(ctx context.Context, repo domain.InquiryRepository, notifier adsconversiondomain.Notifier, log *zap.Logger, inquiry *domain.Inquiry) {
	gaClientID := inquiry.GAClientID()
	if gaClientID == nil || *gaClientID == "" {
		if log != nil {
			log.Warn("adsconversion: skipping close_convert_lead, no ga_client_id captured for this lead", zap.String("inquiry_id", inquiry.ID()))
		}
		return
	}

	if err := notifier.SendCloseConvertLead(ctx, *gaClientID, inquiry.ConversionValue()); err != nil {
		if log != nil {
			log.Warn("adsconversion: close_convert_lead push failed, will need cmd/retry-ads-conversion-sync", zap.String("inquiry_id", inquiry.ID()), zap.Error(err))
		}
		return
	}

	inquiry.MarkAdsConversionSynced()
	if _, err := repo.UpdateAdsConversionSync(ctx, inquiry); err != nil && log != nil {
		log.Warn("adsconversion: close_convert_lead sent but failed to record ads_conversion_synced_at", zap.String("inquiry_id", inquiry.ID()), zap.Error(err))
	}
}
