package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/author/domain"
)

type authorResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	AvatarURL string     `json:"avatar_url"`
	Bio       string     `json:"bio"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func toAuthorResponse(a *domain.Author) authorResponse {
	return authorResponse{
		ID:        a.ID(),
		Name:      a.Name(),
		Slug:      a.Slug(),
		AvatarURL: a.AvatarURL(),
		Bio:       a.Bio(),
		CreatedAt: a.CreatedAt(),
		UpdatedAt: a.UpdatedAt(),
		DeletedAt: a.DeletedAt(),
	}
}

func toAuthorResponseList(authors []*domain.Author) []authorResponse {
	result := make([]authorResponse, len(authors))
	for i, a := range authors {
		result[i] = toAuthorResponse(a)
	}
	return result
}

type createAuthorRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
}

type updateAuthorRequest struct {
	Name      *string `json:"name"`
	Slug      *string `json:"slug"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
}
