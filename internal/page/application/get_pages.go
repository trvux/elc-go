package application

import (
	"context"

	"github.com/trvux/elc-go/internal/page/domain"
)

func GetPages(ctx context.Context, repo domain.PageRepository, filter domain.PageFilter) ([]*domain.Page, error) {
	return repo.GetAll(ctx, filter)
}

func CountPages(ctx context.Context, repo domain.PageRepository, filter domain.PageFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetPageByID(ctx context.Context, repo domain.PageRepository, id string) (*domain.Page, error) {
	return repo.GetByID(ctx, id)
}

func GetPageBySlug(ctx context.Context, repo domain.PageRepository, slug string) (*domain.Page, error) {
	return repo.GetBySlug(ctx, slug)
}
