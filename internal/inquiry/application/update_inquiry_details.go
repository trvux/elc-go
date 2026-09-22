package application

import (
	"context"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// UpdateInquiryDetailsInput is the admin-only payload for PATCH
// /inquiries/{id} — filling in a click-origin lead's identity and/or
// recording the order value on conversion. Kept separate from
// UpdateInquiryStatusInput (PATCH /inquiries/{id}/status) — one endpoint
// per well-defined concern, matching this module's existing split between
// repo.Update (status/note) and repo.UpdateClickContext (click refresh).
type UpdateInquiryDetailsInput struct {
	ID              string
	Name            string
	Phone           string
	ConversionValue *float64
}

// UpdateInquiryDetails fills in a click-origin lead's name/phone once staff
// actually reach the customer, and/or records the order value — see
// UpdateInquiryDetailsInput's doc comment for why this is a separate
// endpoint/use case from UpdateInquiryStatus.
func UpdateInquiryDetails(ctx context.Context, repo domain.InquiryRepository, input UpdateInquiryDetailsInput) (*domain.Inquiry, error) {
	inquiry, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if inquiry == nil {
		return nil, apperr.NewNotFoundError("inquiry")
	}

	if err := inquiry.SetIdentity(input.Name, input.Phone); err != nil {
		return nil, err
	}
	inquiry.SetConversionValue(input.ConversionValue)

	return repo.UpdateDetails(ctx, inquiry)
}
