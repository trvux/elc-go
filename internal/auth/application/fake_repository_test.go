package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
)

// fakeUserRepository is an in-memory stand-in for the pgx repository, used
// only in tests so the application layer can be tested without a real DB.
type fakeUserRepository struct {
	users map[string]*domain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{users: map[string]*domain.User{}}
}

func (r *fakeUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	id := fmt.Sprintf("user-%d", len(r.users)+1)
	created := domain.RehydrateUser(id, user.Username(), user.Email(), user.PasswordHash(), user.Name(), user.Phone(), user.AvatarURL(), user.Role(), user.Status(), user.LastLoginAt(), user.CreatedAt(), user.UpdatedAt())
	r.users[id] = created
	return created, nil
}

func (r *fakeUserRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	r.users[user.ID()] = user
	return user, nil
}

func (r *fakeUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.users[id], nil
}

func (r *fakeUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email(), email) {
			return u, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepository) GetByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email(), identifier) || strings.EqualFold(u.Username(), identifier) {
			return u, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Username(), username) || strings.EqualFold(u.Email(), email) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeUserRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	result := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		result = append(result, u)
	}
	return result, nil
}

// fakeTokenRepository is an in-memory stand-in for the verification_tokens
// pgx repository.
type fakeTokenRepository struct {
	tokens map[string]*domain.VerificationToken
}

func newFakeTokenRepository() *fakeTokenRepository {
	return &fakeTokenRepository{tokens: map[string]*domain.VerificationToken{}}
}

func (r *fakeTokenRepository) Create(ctx context.Context, token *domain.VerificationToken) (*domain.VerificationToken, error) {
	id := fmt.Sprintf("token-%d", len(r.tokens)+1)
	created := domain.RehydrateVerificationToken(id, token.Purpose(), token.TokenHash(), token.Email(), token.Role(), token.InvitedBy(), token.UserID(), token.ExpiresAt(), nil, token.CreatedAt())
	r.tokens[id] = created
	return created, nil
}

func (r *fakeTokenRepository) GetByHash(ctx context.Context, purpose domain.TokenPurpose, tokenHash string) (*domain.VerificationToken, error) {
	for _, t := range r.tokens {
		if t.Purpose() == purpose && t.TokenHash() == tokenHash {
			return t, nil
		}
	}
	return nil, nil
}

func (r *fakeTokenRepository) MarkConsumed(ctx context.Context, id string) error {
	t, ok := r.tokens[id]
	if !ok {
		return errors.New("token not found")
	}
	consumed := domain.RehydrateVerificationToken(t.ID(), t.Purpose(), t.TokenHash(), t.Email(), t.Role(), t.InvitedBy(), t.UserID(), t.ExpiresAt(), ptrNow(), t.CreatedAt())
	r.tokens[id] = consumed
	return nil
}

// fakeSessionRepository is an in-memory stand-in for the sessions pgx
// repository.
type fakeSessionRepository struct {
	sessions map[string]*domain.Session
}

func newFakeSessionRepository() *fakeSessionRepository {
	return &fakeSessionRepository{sessions: map[string]*domain.Session{}}
}

func (r *fakeSessionRepository) Create(ctx context.Context, session *domain.Session) (*domain.Session, error) {
	id := fmt.Sprintf("session-%d", len(r.sessions)+1)
	created := domain.RehydrateSession(id, session.UserID(), session.TokenHash(), session.UserAgent(), session.IPAddress(), session.ExpiresAt(), nil, session.CreatedAt())
	r.sessions[id] = created
	return created, nil
}

func (r *fakeSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	for _, s := range r.sessions {
		if s.TokenHash() == tokenHash {
			return s, nil
		}
	}
	return nil, nil
}

func (r *fakeSessionRepository) Revoke(ctx context.Context, id string) error {
	s, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	s.Revoke(time.Now())
	return nil
}

func (r *fakeSessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	for _, s := range r.sessions {
		if s.UserID() == userID {
			s.Revoke(time.Now())
		}
	}
	return nil
}

// fakePasswordHasher avoids a real bcrypt dependency in application-layer
// tests; it just checks for equality with a fixed prefix.
type fakePasswordHasher struct{}

func (fakePasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakePasswordHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

// fakeTokenIssuer avoids a real JWT dependency in application-layer tests.
type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueAccessToken(userID string, role domain.Role) (string, error) {
	return "access-token-for-" + userID, nil
}

// fakeEmailSender records what would have been sent, so tests can assert on
// it instead of needing a real SMTP server.
type fakeEmailSender struct {
	invitesSent []string
	resetsSent  []string
}

func (f *fakeEmailSender) SendInvite(ctx context.Context, to string, rawToken string, role domain.Role) error {
	f.invitesSent = append(f.invitesSent, to)
	return nil
}

func (f *fakeEmailSender) SendPasswordReset(ctx context.Context, to string, rawToken string) error {
	f.resetsSent = append(f.resetsSent, to)
	return nil
}

func ptrNow() *time.Time {
	t := time.Now()
	return &t
}
