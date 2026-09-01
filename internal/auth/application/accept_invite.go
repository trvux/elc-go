package application

import (
	"context"
	"errors"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type AcceptInviteInput struct {
	Token    string
	Username string
	Password string
	// Name and Phone are optional self-reported profile info.
	Name  string
	Phone string
}

// AcceptInvite is the closest thing this system has to "register" — but it
// only ever succeeds for someone holding a valid, unexpired, unconsumed
// invite token that an existing admin generated for their exact email. A
// stranger hitting this endpoint without a token gets nothing.
func AcceptInvite(
	ctx context.Context,
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	hasher domain.PasswordHasher,
	input AcceptInviteInput,
) (*domain.User, error) {
	hash := domain.HashToken(input.Token)
	token, err := tokenRepo.GetByHash(ctx, domain.TokenPurposeInvite, hash)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if token == nil || !token.IsUsable() {
		return nil, apperr.NewUnauthorizedError("invite link is invalid or has expired")
	}

	if errs := domain.ValidatePassword(input.Password); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"password": errs})
	}

	exists, err := userRepo.ExistsByUsernameOrEmail(ctx, input.Username, token.Email())
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if exists {
		return nil, apperr.NewConflictError("username or email already in use")
	}

	passwordHash, err := hasher.Hash(input.Password)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	user, err := domain.NewUser(input.Username, token.Email(), passwordHash, input.Name, input.Phone, token.Role())
	if err != nil {
		return nil, err
	}

	created, err := userRepo.Create(ctx, user)
	if err != nil {
		// A rare concurrent AcceptInvite for the same username/email can
		// lose the ExistsByUsernameOrEmail race above and hit the table's
		// unique constraint instead — the repository already maps that to a
		// clean apperr.NewConflictError, so pass it through as-is rather
		// than flattening it into a generic 500.
		var appErr *apperr.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, apperr.NewInternalError(err)
	}

	if err := tokenRepo.MarkConsumed(ctx, token.ID()); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	return created, nil
}
