package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/faq/domain"
)

type faqResponse struct {
	ID          string    `json:"id"`
	OwnerType   string    `json:"owner_type"`
	OwnerID     string    `json:"owner_id"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer"`
	OrderIndex  int       `json:"order_index"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toFAQResponse(f *domain.FAQ) faqResponse {
	return faqResponse{
		ID:          f.ID(),
		OwnerType:   string(f.OwnerType()),
		OwnerID:     f.OwnerID(),
		Question:    f.Question(),
		Answer:      f.Answer(),
		OrderIndex:  f.OrderIndex(),
		IsPublished: f.IsPublished(),
		CreatedAt:   f.CreatedAt(),
		UpdatedAt:   f.UpdatedAt(),
	}
}

func toFAQResponseList(faqs []*domain.FAQ) []faqResponse {
	result := make([]faqResponse, len(faqs))
	for i, f := range faqs {
		result[i] = toFAQResponse(f)
	}
	return result
}

type createFAQRequest struct {
	OwnerType   string `json:"owner_type"`
	OwnerID     string `json:"owner_id"`
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	OrderIndex  int    `json:"order_index"`
	IsPublished bool   `json:"is_published"`
}

type updateFAQRequest struct {
	Question    *string `json:"question"`
	Answer      *string `json:"answer"`
	OrderIndex  *int    `json:"order_index"`
	IsPublished *bool   `json:"is_published"`
}
