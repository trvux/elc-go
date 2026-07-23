package application

import (
	"context"

	"github.com/trvux/elc-go/internal/hp-page/domain"
)

func GetHpPages(ctx context.Context, repo domain.HpPageRepository, filter domain.HpPageFilter) ([]*domain.HpPage, error) {
	return repo.GetAll(ctx, filter)
}

func GetHpPageByID(ctx context.Context, repo domain.HpPageRepository, id string) (*domain.HpPage, error) {
	return repo.GetByID(ctx, id)
}

func GetHpPageBySlug(ctx context.Context, repo domain.HpPageRepository, slug string) (*domain.HpPage, error) {
	return repo.GetBySlug(ctx, slug)
}
