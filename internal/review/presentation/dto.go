package presentation

import "github.com/trvux/elc-go/internal/review/domain"

const timeFormat = "2006-01-02T15:04:05Z07:00"

// reviewResponse is what both the public site and the admin panel see over
// HTTP — separate from domain.Review so the entity's internal shape can
// evolve independently. ReviewerPhone/SourceIP/UserAgent are intentionally
// omitted from the public-facing shape (see toPublicReviewResponse) but
// included here for the admin moderation screen.
type reviewResponse struct {
	ID            string  `json:"id"`
	ProductID     *string `json:"product_id"`
	ServiceID     *string `json:"service_id"`
	Rating        int     `json:"rating"`
	Comment       string  `json:"comment"`
	ReviewerName  string  `json:"reviewer_name"`
	ReviewerPhone *string `json:"reviewer_phone"`
	IsPublished   bool    `json:"is_published"`
	SourceIP      *string `json:"source_ip"`
	UserAgent     *string `json:"user_agent"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toReviewResponse(r *domain.Review) reviewResponse {
	return reviewResponse{
		ID:            r.ID(),
		ProductID:     r.ProductID(),
		ServiceID:     r.ServiceID(),
		Rating:        r.Rating(),
		Comment:       r.Comment(),
		ReviewerName:  r.ReviewerName(),
		ReviewerPhone: r.ReviewerPhone(),
		IsPublished:   r.IsPublished(),
		SourceIP:      r.SourceIP(),
		UserAgent:     r.UserAgent(),
		CreatedAt:     r.CreatedAt().Format(timeFormat),
		UpdatedAt:     r.UpdatedAt().Format(timeFormat),
	}
}

func toReviewResponseList(reviews []*domain.Review) []reviewResponse {
	result := make([]reviewResponse, len(reviews))
	for idx, r := range reviews {
		result[idx] = toReviewResponse(r)
	}
	return result
}

// publicReviewResponse is what the product/service detail page renders —
// no phone/IP/user-agent, that data exists purely for internal moderation.
type publicReviewResponse struct {
	ID           string `json:"id"`
	Rating       int    `json:"rating"`
	Comment      string `json:"comment"`
	ReviewerName string `json:"reviewer_name"`
	CreatedAt    string `json:"created_at"`
}

func toPublicReviewResponse(r *domain.Review) publicReviewResponse {
	return publicReviewResponse{
		ID:           r.ID(),
		Rating:       r.Rating(),
		Comment:      r.Comment(),
		ReviewerName: r.ReviewerName(),
		CreatedAt:    r.CreatedAt().Format(timeFormat),
	}
}

func toPublicReviewResponseList(reviews []*domain.Review) []publicReviewResponse {
	result := make([]publicReviewResponse, len(reviews))
	for idx, r := range reviews {
		result[idx] = toPublicReviewResponse(r)
	}
	return result
}

type reviewSummaryResponse struct {
	Count   int     `json:"count"`
	Average float64 `json:"average"`
}

func toReviewSummaryResponse(s *domain.ReviewSummary) reviewSummaryResponse {
	return reviewSummaryResponse{Count: s.Count, Average: s.Average}
}

type countResponse struct {
	Count int `json:"count"`
}

// createReviewRequest is the public form payload. Website is a honeypot —
// a hidden field real visitors never see or fill; a bot that fills every
// field on the page trips it. See ReviewHandler.Create.
type createReviewRequest struct {
	ProductID     *string `json:"product_id"`
	ServiceID     *string `json:"service_id"`
	Rating        int     `json:"rating"`
	Comment       string  `json:"comment"`
	ReviewerName  string  `json:"reviewer_name"`
	ReviewerPhone *string `json:"reviewer_phone"`
	Website       string  `json:"website"`
}

type updateReviewPublishedRequest struct {
	IsPublished bool `json:"is_published"`
}
