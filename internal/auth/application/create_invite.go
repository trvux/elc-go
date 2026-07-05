package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type CreateInviteInput struct {
	Email string
	Role  domain.Role
}

// CreateInvite is the only path that lets a new admin account come into
// existence: an already-authenticated admin/super_admin invites an email
// address, which later accepts the invite (application.AcceptInvite) to set
// its own username/password. There is no public registration endpoint.
func CreateInvite(
	ctx context.Context,
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	emailSender domain.EmailSender,
	inviter *domain.User,
	input CreateInviteInput,
) (*domain.VerificationToken, error) {
	if !inviter.CanInvite(input.Role) {
		return nil, apperr.NewForbiddenError("you are not allowed to invite this role")
	}

	existing, err := userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if existing != nil {
		return nil, apperr.NewConflictError("a user with this email already exists")
	}

	token, raw, err := domain.NewInviteToken(input.Email, input.Role, inviter.ID(), InviteTokenTTL)
	if err != nil {
		return nil, err
	}

	created, err := tokenRepo.Create(ctx, token)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	if err := emailSender.SendInvite(ctx, created.Email(), raw, created.Role()); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	return created, nil
}
