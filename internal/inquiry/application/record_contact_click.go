package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// RecordContactClick persists a Zalo/Messenger/Hotline click as a lead.
// Unlike CreateInquiry (the on-site form, always a fresh row), this
// upserts: a repeat click from the same visitor (session_id) on the same
// channel, while their previous click-origin lead is still open (new or
// contacted), refreshes that row's entity/attribution instead of creating
// a duplicate — see domain.Inquiry.RefreshClickContext's doc comment for
// why. A blank/missing session_id (cookie blocked) always creates a new
// row — there's nothing to dedup against.
func RecordContactClick(ctx context.Context, repo domain.InquiryRepository, input domain.CreateContactClickInput) (*domain.Inquiry, error) {
	// Validated once, up front, so both the refresh and the create path
	// below get the same guarantees — RefreshClickContext (unlike
	// NewInquiry) has no validation of its own, it trusts its caller.
	// Unlike CreateInquiry's channel (which defaults to "form" when blank —
	// a sensible default for that endpoint), there's no sensible default
	// here: every call into this endpoint IS a click on some specific
	// channel, so a blank/unknown one is always a caller bug, not a normal
	// omission.
	if !input.Channel.IsValid() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"channel": {"invalid channel"}})
	}
	if input.LeadType == "" {
		input.LeadType = domain.LeadTypeGeneral
	}
	if !input.LeadType.IsValid() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"leadType": {"invalid lead type"}})
	}

	if input.SessionID != nil && *input.SessionID != "" {
		existing, err := repo.FindOpenBySessionAndChannel(ctx, *input.SessionID, input.Channel)
		if err != nil {
			return nil, fmt.Errorf("record contact click: find open: %w", err)
		}
		if existing != nil {
			existing.RefreshClickContext(
				input.LeadType, input.SubType,
				input.ProductID, input.ProjectID, input.ServiceID,
				input.QualifyData,
				input.GCLID, input.UTMSource, input.UTMMedium, input.UTMCampaign, input.UTMTerm, input.UTMContent, input.GAClientID,
			)
			updated, err := repo.UpdateClickContext(ctx, existing)
			if err != nil {
				return nil, fmt.Errorf("record contact click: update: %w", err)
			}
			return updated, nil
		}
	}

	inquiry, err := domain.NewInquiry(
		"", "", nil, nil,
		input.ProductID, input.ProjectID, input.ServiceID,
		input.LeadType, input.SubType, input.QualifyData, nil,
		input.Channel,
		input.GCLID, input.UTMSource, input.UTMMedium, input.UTMCampaign, input.UTMTerm, input.UTMContent, input.GAClientID, input.SessionID,
		input.SourceIP, input.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, inquiry)
}
