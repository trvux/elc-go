package application

import (
	"context"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// CreateInquiry persists a lead first (this must succeed for the request to
// succeed) and only then best-effort notifies staff. Unlike
// auth.CreateInvite (which fails the request if EmailSender errors, since an
// invite has no other side channel), a notify failure here must never lose a
// lead that's already durably saved — LeadNotifier implementations are
// responsible for logging their own failures and always returning nil, so
// the error is deliberately ignored here rather than treated as fatal.
func CreateInquiry(
	ctx context.Context,
	repo domain.InquiryRepository,
	notifier domain.LeadNotifier,
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

	created, err := repo.Create(ctx, inquiry)
	if err != nil {
		return nil, err
	}

	_ = notifier.NotifyNewLead(ctx, created)

	return created, nil
}
