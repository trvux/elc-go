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
	// Frontend always resubmits the entire filter configuration on every
	// save (no partial-patch semantics for this form) — applied
	// unconditionally so clearing a filter (e.g. removing the attribute
	// code) actually takes effect instead of being skipped as "not
	// provided".
	page.UpdateAttributeCode(input.AttributeCode)
	page.UpdateAttributeValues(input.AttributeValues)
	page.UpdateCategoryIDs(input.CategoryIDs)
	page.UpdateBrandIDs(input.BrandIDs)

	if !page.HasAnyFilter() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"filters": {"page must have at least one filter: attribute, category, or brand"},
		})
	}

	return repo.Update(ctx, page)
}
