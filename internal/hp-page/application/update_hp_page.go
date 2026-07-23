package application

import (
	"context"

	"github.com/trvux/elc-go/internal/hp-page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

func UpdateHpPage(ctx context.Context, repo domain.HpPageRepository, input domain.UpdateHpPageInput) (*domain.HpPage, error) {
	page, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, apperr.NewNotFoundError("hp_page")
	}

	if input.Name != nil {
		if err := page.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := page.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.ImageURL != nil {
		page.UpdateImageURL(*input.ImageURL)
	}
	if input.MetaTitle != nil && !seo.Unchanged(page.MetaTitle(), input.MetaTitle) {
		if err := page.UpdateMetaTitle(input.MetaTitle); err != nil {
			return nil, err
		}
	}
	if input.MetaDescription != nil && !seo.Unchanged(page.MetaDescription(), input.MetaDescription) {
		if err := page.UpdateMetaDescription(input.MetaDescription); err != nil {
			return nil, err
		}
	}
	if input.OrderIndex != nil {
		page.Reorder(*input.OrderIndex)
	}
	if input.Content != nil {
		page.UpdateContent(input.Content)
	}
	if input.AttributeCode != nil {
		if err := page.UpdateAttributeCode(*input.AttributeCode); err != nil {
			return nil, err
		}
	}
	if input.AttributeValues != nil {
		if err := page.UpdateAttributeValues(input.AttributeValues); err != nil {
			return nil, err
		}
	}

	return repo.Update(ctx, page)
}
