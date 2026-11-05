package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/redirect/domain"
)

type redirectResponse struct {
	ID        string    `json:"id"`
	FromPath  string    `json:"from_path"`
	ToPath    string    `json:"to_path"`
	IsActive  bool      `json:"is_active"`
	Source    string    `json:"source"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type listResponse struct {
	Items []redirectResponse `json:"items"`
	Total int                `json:"total"`
}

func toResponse(r *domain.Redirect) redirectResponse {
	return redirectResponse{
		ID: r.ID(), FromPath: r.FromPath(), ToPath: r.ToPath(), IsActive: r.IsActive(),
		Source: string(r.Source()), Note: r.Note(), CreatedAt: r.CreatedAt(), UpdatedAt: r.UpdatedAt(),
	}
}

type createRequest struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
	Note     string `json:"note"`
}

type updateRequest struct {
	ToPath   *string `json:"to_path"`
	Note     *string `json:"note"`
	IsActive *bool   `json:"is_active"`
}
