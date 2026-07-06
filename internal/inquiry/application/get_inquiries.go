package application

import (
	"context"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func GetInquiries(ctx context.Context, repo domain.InquiryRepository, filter domain.InquiryFilter) ([]*domain.Inquiry, error) {
	return repo.GetAll(ctx, filter)
}

func CountInquiries(ctx context.Context, repo domain.InquiryRepository, filter domain.InquiryFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetInquiryByID(ctx context.Context, repo domain.InquiryRepository, id string) (*domain.Inquiry, error) {
	return repo.GetByID(ctx, id)
}
