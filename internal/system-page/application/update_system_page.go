package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/system-page/domain"
)

func UpdateSystemPage(ctx context.Context, repo domain.SystemPageRepository, input domain.UpdateSystemPageInput) (*domain.SystemPage, error) {
	p, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NewNotFoundError("system page")
	}

	if err := p.UpdateMeta(input.MetaTitle, input.MetaDescription); err != nil {
		return nil, err
	}

	return repo.Update(ctx, p)
}
