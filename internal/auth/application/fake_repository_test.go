package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
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
	created := domain.RehydrateUser(id, user.Username(), user.Email(), user.PasswordHash(), user.Name(), user.Phone(), user.AvatarURL(), user.GoogleSub(), user.Role(), user.Status(), user.LastLoginAt(), user.CreatedAt(), user.UpdatedAt())
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

func (r *fakeUserRepository) GetByGoogleSub(ctx context.Context, sub string) (*domain.User, error) {
	for _, u := range r.users {
		if u.GoogleSub() != nil && *u.GoogleSub() == sub {
			return u, nil
		}
	}
	return nil, nil
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
	created := domain.RehydrateVerificationToken(id, token.Purpose(), token.TokenHash(), token.Email(), token.Code(), token.Attempts(), token.ExpiresAt(), nil, token.CreatedAt())
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

func (r *fakeTokenRepository) GetLatestActiveByEmail(ctx context.Context, purpose domain.TokenPurpose, email string) (*domain.VerificationToken, error) {
	var latest *domain.VerificationToken
	for _, t := range r.tokens {
		if t.Purpose() != purpose || !strings.EqualFold(t.Email(), email) || !t.IsUsable() {
			continue
		}
		if latest == nil || t.CreatedAt().After(latest.CreatedAt()) {
			latest = t
		}
	}
	return latest, nil
}

func (r *fakeTokenRepository) MarkConsumed(ctx context.Context, id string) error {
	t, ok := r.tokens[id]
	if !ok {
		return errors.New("token not found")
	}
	consumed := domain.RehydrateVerificationToken(t.ID(), t.Purpose(), t.TokenHash(), t.Email(), t.Code(), t.Attempts(), t.ExpiresAt(), ptrNow(), t.CreatedAt())
	r.tokens[id] = consumed
	return nil
}

func (r *fakeTokenRepository) IncrementAttempts(ctx context.Context, id string) (int, error) {
	t, ok := r.tokens[id]
	if !ok {
		return 0, errors.New("token not found")
	}
	updated := domain.RehydrateVerificationToken(t.ID(), t.Purpose(), t.TokenHash(), t.Email(), t.Code(), t.Attempts()+1, t.ExpiresAt(), nil, t.CreatedAt())
	r.tokens[id] = updated
	return updated.Attempts(), nil
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

// fakeTokenIssuer avoids a real JWT dependency in application-layer tests.
type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueAccessToken(userID string, role domain.Role) (string, error) {
	return "access-token-for-" + userID, nil
}

// fakeGoogleAuthenticator avoids a real Google API dependency in
// application-layer tests — it just echoes back whatever info was queued.
type fakeGoogleAuthenticator struct {
	info *domain.GoogleUserInfo
	err  error
}

func (f *fakeGoogleAuthenticator) Exchange(ctx context.Context, code string, redirectURI string) (*domain.GoogleUserInfo, error) {
	return f.info, f.err
}

// fakeEmailSender records what would have been sent, so tests can assert on
// it instead of needing a real SMTP server.
type fakeEmailSender struct {
	magicLinksSent []string
}

func (f *fakeEmailSender) SendMagicLink(ctx context.Context, to string, rawToken string, code string) error {
	f.magicLinksSent = append(f.magicLinksSent, to)
	return nil
}

func ptrNow() *time.Time {
	t := time.Now()
	return &t
}

// seedActiveUser seeds an active user directly (bypassing any use case) for
// tests that need one to already exist — no password, since accounts are no
// longer created by setting one.
func seedActiveUser(t *testing.T, repo *fakeUserRepository, username, email string, role domain.Role) *domain.User {
	t.Helper()
	user, err := domain.NewUser(username, email, "unused-hash", "", "", role)
	if err != nil {
		t.Fatalf("failed to build user: %v", err)
	}
	return mustCreate(t, repo, user)
}

func mustCreate(t *testing.T, repo *fakeUserRepository, user *domain.User) *domain.User {
	t.Helper()
	created, err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	return created
}
