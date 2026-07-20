package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/review/domain"
)

type createReviewRequest struct {
	Rating        int     `json:"rating"`
	Comment       string  `json:"comment"`
	ReviewerName  string  `json:"reviewer_name"`
	ReviewerPhone *string `json:"reviewer_phone"`
	// Honeypot — hidden from real visitors, must stay empty. Same
	// convention as internal/inquiry's createInquiryRequest.Website.
	Website string `json:"website"`
}

// reviewResponse deliberately omits reviewer_phone — it's collected and
// stored (so staff could follow up later) but this is a PUBLIC read
// endpoint shown to every site visitor, not just staff, so a reviewer's
// phone number must never appear in it.
type reviewResponse struct {
	ID           string    `json:"id"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	ReviewerName string    `json:"reviewer_name"`
	CreatedAt    time.Time `json:"created_at"`
}

func toReviewResponse(r *domain.Review) reviewResponse {
	return reviewResponse{
		ID:           r.ID(),
		Rating:       r.Rating(),
		Comment:      r.Comment(),
		ReviewerName: r.ReviewerName(),
		CreatedAt:    r.CreatedAt(),
	}
}

func toReviewResponseList(reviews []*domain.Review) []reviewResponse {
	result := make([]reviewResponse, len(reviews))
	for i, r := range reviews {
		result[i] = toReviewResponse(r)
	}
	return result
}

type reviewAggregateResponse struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

func toReviewAggregateResponse(a domain.ReviewAggregate) reviewAggregateResponse {
	return reviewAggregateResponse{Average: a.Average, Count: a.Count}
}

type reviewListResponse struct {
	Data      []reviewResponse        `json:"data"`
	Aggregate reviewAggregateResponse `json:"aggregate"`
}

// adminReviewResponse is the staff-only shape — unlike reviewResponse it
// includes reviewer_phone and which product (if any) the review is about.
// Kept as its own type rather than adding fields to reviewResponse so the
// public shape can never regress to include phone.
type adminReviewResponse struct {
	ID            string    `json:"id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	ReviewerName  string    `json:"reviewer_name"`
	ReviewerPhone *string   `json:"reviewer_phone"`
	ProductID     *string   `json:"product_id"`
	ProductName   *string   `json:"product_name"`
	ProductSlug   *string   `json:"product_slug"`
	IsPublished   bool      `json:"is_published"`
	CreatedAt     time.Time `json:"created_at"`
}

func toAdminReviewResponse(rp *domain.ReviewWithProduct) adminReviewResponse {
	resp := adminReviewResponse{
		ID:            rp.ID(),
		Rating:        rp.Rating(),
		Comment:       rp.Comment(),
		ReviewerName:  rp.ReviewerName(),
		ReviewerPhone: rp.ReviewerPhone(),
		IsPublished:   rp.IsPublished(),
		CreatedAt:     rp.CreatedAt(),
	}
	if rp.Product != nil {
		resp.ProductID = &rp.Product.ID
		resp.ProductName = &rp.Product.Name
		resp.ProductSlug = &rp.Product.Slug
	}
	return resp
}

func toAdminReviewResponseList(reviews []*domain.ReviewWithProduct) []adminReviewResponse {
	result := make([]adminReviewResponse, len(reviews))
	for i, rp := range reviews {
		result[i] = toAdminReviewResponse(rp)
	}
	return result
}

type reviewCountResponse struct {
	Count int `json:"count"`
}
