package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Author is the byline entity for news/blog content — a public-facing
// person, deliberately separate from the auth module's User (admin login
// account). A news article's author_id is nullable: not every article needs
// one, and the FK is ON DELETE SET NULL so removing an author never blocks
// on or cascades into existing articles.
type Author struct {
	id        string
	name      string
	slug      string
	avatarURL string
	bio       string
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

// NewAuthor validates and creates a new entity from user input.
func NewAuthor(name, slug, avatarURL, bio string) (*Author, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Author{
		name:      name,
		slug:      slug,
		avatarURL: avatarURL,
		bio:       bio,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// RehydrateAuthor reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateAuthor(
	id, name, slug, avatarURL, bio string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Author {
	return &Author{
		id:        id,
		name:      name,
		slug:      slug,
		avatarURL: avatarURL,
		bio:       bio,
		createdAt: createdAt,
		updatedAt: updatedAt,
		deletedAt: deletedAt,
	}
}

func (a *Author) ID() string            { return a.id }
func (a *Author) Name() string          { return a.name }
func (a *Author) Slug() string          { return a.slug }
func (a *Author) AvatarURL() string     { return a.avatarURL }
func (a *Author) Bio() string           { return a.bio }
func (a *Author) CreatedAt() time.Time  { return a.createdAt }
func (a *Author) UpdatedAt() time.Time  { return a.updatedAt }
func (a *Author) DeletedAt() *time.Time { return a.deletedAt }

func (a *Author) IsDeleted() bool {
	return a.deletedAt != nil
}

func (a *Author) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	a.name = name
	a.updatedAt = time.Now()
	return nil
}

func (a *Author) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	a.slug = slug
	a.updatedAt = time.Now()
	return nil
}

func (a *Author) UpdateAvatarURL(avatarURL string) {
	a.avatarURL = avatarURL
	a.updatedAt = time.Now()
}

func (a *Author) UpdateBio(bio string) {
	a.bio = bio
	a.updatedAt = time.Now()
}

func (a *Author) MarkDeleted(deletedAt time.Time) {
	a.deletedAt = &deletedAt
}

func (a *Author) Restore() {
	a.deletedAt = nil
	a.updatedAt = time.Now()
}

func validateName(name string) []string {
	if name == "" {
		return []string{"name is required"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"slug is required"}
	}
	return nil
}

type CreateAuthorInput struct {
	Name      string
	Slug      string
	AvatarURL string
	Bio       string
}

type UpdateAuthorInput struct {
	ID        string
	Name      *string
	Slug      *string
	AvatarURL *string
	Bio       *string
}

type AuthorFilter struct {
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
