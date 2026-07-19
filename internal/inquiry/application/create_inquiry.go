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
		input.SourceIP, input.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, inquiry)
}
