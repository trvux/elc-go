package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

func ListRedirects(ctx context.Context, repo domain.Repository, filter domain.Filter) ([]*domain.Redirect, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit > MaxPageSize {
		filter.Limit = MaxPageSize
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Source != "" && !filter.Source.IsValid() {
		return nil, 0, apperr.NewValidationError("validation failed", map[string][]string{"source": {"nguồn không hợp lệ"}})
	}
	return repo.List(ctx, filter)
}

func DeleteRedirect(ctx context.Context, repo domain.Repository, id string) error {
	r, err := repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return apperr.NewNotFoundError("redirect")
	}
	return repo.Delete(ctx, id)
}
