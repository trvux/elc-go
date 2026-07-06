package application

import (
	"context"

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
func UpdateInquiryStatus(ctx context.Context, repo domain.InquiryRepository, input UpdateInquiryStatusInput) (*domain.Inquiry, error) {
	inquiry, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if inquiry == nil {
		return nil, apperr.NewNotFoundError("inquiry")
	}

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

	return repo.Update(ctx, inquiry)
}
