package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// fakeInquiryRepository is an in-memory stand-in for
// PostgresInquiryRepository, used only in tests so the application layer can
// be tested without a real DB.
type fakeInquiryRepository struct {
	inquiries map[string]*domain.Inquiry
}

func newFakeInquiryRepository() *fakeInquiryRepository {
	return &fakeInquiryRepository{inquiries: map[string]*domain.Inquiry{}}
}

func (r *fakeInquiryRepository) Create(ctx context.Context, inquiry *domain.Inquiry) (*domain.Inquiry, error) {
	id := fmt.Sprintf("id-%d", len(r.inquiries)+1)
	created := domain.RehydrateInquiry(
		id, inquiry.Name(), inquiry.Phone(),
		inquiry.Email(), inquiry.Message(),
		inquiry.ProductID(), inquiry.ProjectID(), inquiry.ServiceID(),
		inquiry.Status(), inquiry.InternalNote(), inquiry.SourceIP(), inquiry.UserAgent(),
		inquiry.CreatedAt(), inquiry.UpdatedAt(),
	)
	r.inquiries[id] = created
	return created, nil
}

func (r *fakeInquiryRepository) GetAll(ctx context.Context, filter domain.InquiryFilter) ([]*domain.Inquiry, error) {
	result := make([]*domain.Inquiry, 0, len(r.inquiries))
	for _, i := range r.inquiries {
		if filter.Status != "" && string(i.Status()) != filter.Status {
			continue
		}
		result = append(result, i)
	}
	return result, nil
}

func (r *fakeInquiryRepository) Count(ctx context.Context, filter domain.InquiryFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeInquiryRepository) GetByID(ctx context.Context, id string) (*domain.Inquiry, error) {
	i, ok := r.inquiries[id]
	if !ok {
		return nil, nil
	}
	return i, nil
}

func (r *fakeInquiryRepository) Update(ctx context.Context, inquiry *domain.Inquiry) (*domain.Inquiry, error) {
	r.inquiries[inquiry.ID()] = inquiry
	return inquiry, nil
}
