package application

import (
	"context"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// CreateInquiry persists a lead submitted from the public site.
func CreateInquiry(
	ctx context.Context,
	repo domain.InquiryRepository,
	input domain.CreateInquiryInput,
) (*domain.Inquiry, error) {
	inquiry, err := domain.NewInquiry(
		input.Name, input.Phone,
		input.Email, input.Message,
		input.ProductID, input.ProjectID, input.ServiceID,
		input.LeadType, input.SubType, input.QualifyData, input.Attachments,
		input.Channel,
		// No session_id here — dedup-by-session is only meaningful for
		// click-origin leads with no name/phone (see RecordContactClick);
		// the form always has both, so staff can already tell duplicates
		// apart.
		input.GCLID, input.UTMSource, input.UTMMedium, input.UTMCampaign, input.UTMTerm, input.UTMContent, input.GAClientID, nil,
		input.SourceIP, input.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, inquiry)
}
