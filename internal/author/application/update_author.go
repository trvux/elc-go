package application

import (
	"context"

	"github.com/trvux/elc-go/internal/author/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateAuthor(ctx context.Context, repo domain.AuthorRepository, input domain.UpdateAuthorInput) (*domain.Author, error) {
	author, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, apperr.NewNotFoundError("author")
	}

	if input.Name != nil {
		if err := author.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := author.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.AvatarURL != nil {
		author.UpdateAvatarURL(*input.AvatarURL)
	}
	if input.Bio != nil {
		author.UpdateBio(*input.Bio)
	}

	return repo.Update(ctx, author)
}
