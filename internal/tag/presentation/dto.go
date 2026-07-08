package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/tag/domain"
)

type tagResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func toTagResponse(t *domain.Tag) tagResponse {
	return tagResponse{
		ID:        t.ID(),
		Name:      t.Name(),
		Slug:      t.Slug(),
		CreatedAt: t.CreatedAt(),
		UpdatedAt: t.UpdatedAt(),
		DeletedAt: t.DeletedAt(),
	}
}

func toTagResponseList(tags []*domain.Tag) []tagResponse {
	result := make([]tagResponse, len(tags))
	for i, t := range tags {
		result[i] = toTagResponse(t)
	}
	return result
}

type createTagRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type updateTagRequest struct {
	Name *string `json:"name"`
	Slug *string `json:"slug"`
}
