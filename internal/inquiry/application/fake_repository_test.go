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

// fakeLeadNotifier records every lead it was asked to notify about, so tests
// can assert CreateInquiry calls it — and never returns an error, matching
// the real implementations' contract (log-and-swallow, see domain.LeadNotifier).
type fakeLeadNotifier struct {
	notified []*domain.Inquiry
}

func (n *fakeLeadNotifier) NotifyNewLead(ctx context.Context, inquiry *domain.Inquiry) error {
	n.notified = append(n.notified, inquiry)
	return nil
}

// fakeZaloFollowerRepository is an in-memory stand-in for
// PostgresZaloFollowerRepository.
type fakeZaloFollowerRepository struct {
	followers map[string]*domain.ZaloOAFollower
}

func newFakeZaloFollowerRepository() *fakeZaloFollowerRepository {
	return &fakeZaloFollowerRepository{followers: map[string]*domain.ZaloOAFollower{}}
}

func (r *fakeZaloFollowerRepository) Upsert(ctx context.Context, follower *domain.ZaloOAFollower) error {
	r.followers[follower.ZaloUserID()] = domain.RehydrateZaloOAFollower(
		follower.ZaloUserID(), follower.ZaloUserID(), follower.DisplayName(), true, follower.FollowedAt(),
	)
	return nil
}

func (r *fakeZaloFollowerRepository) SetActive(ctx context.Context, zaloUserID string, isActive bool) error {
	f, ok := r.followers[zaloUserID]
	if !ok {
		return nil
	}
	r.followers[zaloUserID] = domain.RehydrateZaloOAFollower(f.ID(), f.ZaloUserID(), f.DisplayName(), isActive, f.FollowedAt())
	return nil
}

func (r *fakeZaloFollowerRepository) GetAllActive(ctx context.Context) ([]*domain.ZaloOAFollower, error) {
	var result []*domain.ZaloOAFollower
	for _, f := range r.followers {
		if f.IsActive() {
			result = append(result, f)
		}
	}
	return result, nil
}
